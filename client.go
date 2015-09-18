package galf

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

type (
	Client struct {
		TokenManager TokenManager
		Options      ClientOptions
		useHystrix   bool
	}
)

func NewClient(options ...ClientOptions) *Client {
	clientOptions := defaultClientOptions
	if len(options) > 0 {
		clientOptions = options[0]
	}
	return NewClientCustom(defaultTokenManager, clientOptions)
}

func NewClientCustom(tokenManager TokenManager, options ClientOptions) *Client {
	return &Client{
		TokenManager: tokenManager,
		Options:      options,
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

	if c.TokenManager == nil {
		return nil, errors.New("Configure tokenManager or SetDefaultTokenManager")
	}

	var originalBody []byte
	if body != nil {
		if originalBody, err = ioutil.ReadAll(body); err != nil {
			return nil, stackerr.Wrap(err)
		}
	}

	for i := 0; i < c.Options.MaxRetries; i++ {
		if len(originalBody) > 0 {
			body = bytes.NewBuffer(originalBody)
		}

		resp, err = c.do(method, urlStr, body)
		if err == nil && resp.StatusCode < 300 {
			return resp, nil
		}

		if i < c.Options.MaxRetries-1 {
			time.Sleep(c.Options.Backoff(i))
		}
	}

	return resp, err
}

func (c *Client) do(method string, urlStr string, body io.Reader) (resp *goreq.Response, err error) {
	if c.Options.HystrixConfig == nil {
		return c.request(method, urlStr, body)
	}

	if err = c.Options.HystrixConfig.valid(); err != nil {
		return nil, err
	}
	return c.requestHystrix(method, urlStr, body)
}

func (c *Client) requestHystrix(method string, urlStr string, body io.Reader) (*goreq.Response, error) {

	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go(c.Options.HystrixConfig.configName, func() error {

		resp, err := c.request(method, urlStr, body)
		if err != nil {
			return err
		}
		output <- resp

		return nil
	}, nil)

	select {
	case out := <-output:
		return out, nil
	case err := <-errors:
		return nil, err
	}
}

func (c *Client) request(method string, urlStr string, body io.Reader) (*goreq.Response, error) {
	token, err := c.TokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	resp, err := goreq.Request{
		Method:      method,
		ContentType: "application/json",
		Uri:         urlStr,
		Body:        body,
		Timeout:     c.Options.Timeout,
		ShowDebug:   c.Options.ShowDebug,
	}.WithHeader("Authorization", token.Authorization).Do()

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	return resp, nil
}
