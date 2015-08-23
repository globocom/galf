package alf

import (
	"bytes"
	"io"
	"io/ioutil"
	"math"
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

var DefaultTimeout = 5 * time.Second

func SetConnectTimeout(duration time.Duration) {
	DefaultTimeout = duration
}

func NewClient(timeout ...time.Duration) *Client {
	timeOut := DefaultTimeout
	if len(timeout) > 0 {
		timeOut = timeout[0]
	}
	return &Client{
		tokenManager: defaultTokenManager,
		Timeout:      timeOut,
		MaxRetries:   2,
		Backoff:      DefaultBackoff,
		ShowDebug:    false,
	}
}

// BackoffStrategy is used to determine how long a retry request should wait until attempted
type BackoffStrategy func(retry int) time.Duration

// DefaultBackoff always returns 100 Millisecond
func DefaultBackoff(_ int) time.Duration {
	return 100 * time.Millisecond
}

// ExponentialBackoff returns ever increasing backoffs by a power of 2
func ExponentialBackoff(i int) time.Duration {
	return time.Duration(math.Pow(2, float64(i))) * time.Second
}

// LinearBackoff returns increasing durations, each a second longer than the last
// n seconds where n is the retry number
func LinearBackoff(i int) time.Duration {
	return time.Duration(i) * time.Second
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
		originalBody, err = ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < c.MaxRetries; i++ {
		if len(originalBody) > 0 {
			body = bytes.NewBuffer(originalBody)
		}
		resp, err = c.do(method, urlStr, body)

		// 200 and 300 level errors are considered success and we are done
		if err == nil && resp.StatusCode < 400 {
			return resp, nil
		}

		// wait
		time.Sleep(c.Backoff(i))
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
