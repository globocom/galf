package galf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

type (
	Client struct {
		TokenManager TokenManager
		Options      ClientOptions
	}
)

var (
	defaultDialer                      = &net.Dialer{Timeout: 1000 * time.Second}
	defaultTransport http.RoundTripper = &http.Transport{
		Dial:                defaultDialer.Dial,
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConnsPerHost: 250,
	}
	defaultClient = &http.Client{Transport: defaultTransport}
)

func init() {
	goreq.DefaultTransport = defaultTransport
	goreq.DefaultClient = defaultClient
}

func NewClient(options ...ClientOptions) *Client {
	clientOptions := defaultClientOptions
	if len(options) > 0 {
		clientOptions = options[0]
	}
	return NewClientCustom(defaultTokenManager, clientOptions)
}

func NewClientNoWithTokenManager(options ...ClientOptions) *Client {
	clientOptions := defaultClientOptions
	if len(options) > 0 {
		clientOptions = options[0]
	}
	return NewClientCustom(nil, clientOptions)
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

func (c *Client) Post(urlStr string, body interface{}) (*goreq.Response, error) {
	return c.retry("POST", urlStr, body)
}

func (c *Client) Put(urlStr string, body interface{}) (*goreq.Response, error) {
	return c.retry("PUT", urlStr, body)
}

func (c *Client) Delete(urlStr string) (*goreq.Response, error) {
	return c.retry("DELETE", urlStr, nil)
}

func (c *Client) retry(method string, urlStr string, body interface{}) (resp *goreq.Response, err error) {

	if c.TokenManager == nil {
		return nil, errors.New("Configure tokenManager or SetDefaultTokenManager")
	}

	originalBody, err := copyBody(body)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	for i := 0; i < c.Options.MaxRetries; i++ {
		if len(originalBody) > 0 {
			bodyReader = bytes.NewBuffer(originalBody)
		}

		if resp, err = c.do(method, urlStr, bodyReader); err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusUnauthorized {
			return resp, nil
		}

		if i < c.Options.MaxRetries-1 {
			time.Sleep(c.Options.Backoff(i))
		}
	}

	return resp, err
}

func (c *Client) do(method string, urlStr string, body interface{}) (*goreq.Response, error) {
	if c.Options.HystrixConfig == nil {
		authorization, err := c.getAuthorization()
		if err != nil {
			return nil, err
		}
		return c.request(method, urlStr, body, authorization)
	}

	if err := c.Options.HystrixConfig.valid(); err != nil {
		return nil, err
	}
	return c.requestHystrix(method, urlStr, body)
}

func (c *Client) requestHystrix(method string, urlStr string, body interface{}) (*goreq.Response, error) {

	authorization, err := c.getAuthorization()
	if err != nil {
		return nil, err
	}

	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go(c.Options.HystrixConfig.configName, func() error {

		resp, err := c.request(method, urlStr, body, authorization)
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

func (c *Client) request(method string, urlStr string, body interface{}, authorization *string) (*goreq.Response, error) {
	req := goreq.Request{
		Method:      method,
		ContentType: "application/json",
		Uri:         urlStr,
		Body:        body,
		Timeout:     c.Options.Timeout,
		ShowDebug:   c.Options.ShowDebug,
	}

	if authorization != nil {
		req.WithHeader("Authorization", *authorization)
	}

	resp, err := req.Do()

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	return resp, nil
}

func (c *Client) getAuthorization() (*string, error) {
	if c.TokenManager == nil {
		return nil, nil
	}

	token, err := c.TokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	return &token.Authorization, nil
}

func copyBody(b interface{}) ([]byte, error) {
	switch b.(type) {
	case string:
		return []byte(b.(string)), nil

	case io.Reader:
		var originalBody bytes.Buffer
		_, err := io.Copy(&originalBody, b.(io.Reader))
		if err != nil {
			return nil, err
		}
		return originalBody.Bytes(), nil

	case []byte:
		return b.([]byte), nil

	case nil:
		return nil, nil

	default:
		j, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		return j, nil
	}
}
