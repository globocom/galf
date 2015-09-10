package alf

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

type Client struct {
	tokenManager TokenManager
	Timeout      time.Duration
	MaxRetries   int
	Backoff      BackoffStrategy
	ShowDebug    bool
}

var (
	defaultClientTimeout    = 10 * time.Second
	defaultClientMaxRetries = 2
)

func SetDefaultClientConnectTimeout(duration time.Duration) {
	defaultClientTimeout = duration
}

func SetDefaultClientMaxRetries(retries int) {
	defaultClientMaxRetries = retries
}

func NewClient(timeout ...time.Duration) *Client {
	timeOut := defaultClientTimeout
	if len(timeout) > 0 {
		timeOut = timeout[0]
	}
	return &Client{
		tokenManager: defaultTokenManager,
		Timeout:      timeOut,
		MaxRetries:   defaultClientMaxRetries,
		Backoff:      ConstantBackOff,
		ShowDebug:    false,
	}
}

func (c *Client) Get(urlStr string) (*goreq.Response, error) {
	return c.retry("GET", urlStr, nil)
}

func (c *Client) Post(urlStr string, body io.Reader) (*goreq.Response, error) {
	return c.retry("POST", urlStr, body)
}

func (c *Client) Put(urlStr string, body io.Reader) (*goreq.Response, error) {
	return c.retry("PUT", urlStr, body)
}

func (c *Client) Delete(urlStr string) (*goreq.Response, error) {
	return c.retry("DELETE", urlStr, nil)
}

func (c *Client) retry(method string, urlStr string, body io.Reader) (resp *goreq.Response, err error) {

	var originalBody []byte
	if body != nil {
		if originalBody, err = ioutil.ReadAll(body); err != nil {
			return nil, stackerr.Wrap(err)
		}
	}

	for i := 0; i < c.MaxRetries; i++ {
		if len(originalBody) > 0 {
			body = bytes.NewBuffer(originalBody)
		}
		resp, err = c.do(method, urlStr, body)

		if err == nil && resp.StatusCode < 300 {
			return resp, nil
		}

		if i < c.MaxRetries-1 {
			// wait
			time.Sleep(c.Backoff(i))
		}
	}

	return resp, err
}

func (c *Client) do(method string, urlStr string, body io.Reader) (*goreq.Response, error) {
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	resp, err := goreq.Request{
		Method:      method,
		ContentType: "application/json",
		Uri:         urlStr,
		Body:        body,
		Timeout:     c.Timeout,
		ShowDebug:   c.ShowDebug,
	}.WithHeader("Authorization", token.Authorization).Do()

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	return resp, nil
}
