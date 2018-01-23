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

// Client is the galf client's structure
type Client struct {
	TokenManager TokenManager
	Options      ClientOptions
}

var (
	defaultDialer                      = &net.Dialer{Timeout: time.Second}
	defaultTransport http.RoundTripper = &http.Transport{
		Dial:                defaultDialer.Dial,
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConnsPerHost: 250,
	}
	defaultClient = &http.Client{Transport: defaultTransport}
)

func init() {
	SetCustomHTTPClient(defaultClient)
}

// NewClient creates a new instance of Client
// `options` is used to customize client's configurations. Otherwise the default configurations are used
func NewClient(options ...ClientOptions) *Client {
	clientOptions := defaultClientOptions
	if len(options) > 0 {
		clientOptions = options[0]
	}
	return NewClientCustom(defaultTokenManager, clientOptions)
}

// NewClientCustom create a custom instance of Client
// Unlike NewClient, tokenManager and options are required
func NewClientCustom(tokenManager TokenManager, options ClientOptions) *Client {
	return &Client{
		TokenManager: tokenManager,
		Options:      options,
	}
}

// SetCustomHTTPClient allow the override of both http Client and Transport.
// The new custom client's transport is used as the new transport
func SetCustomHTTPClient(client *http.Client) {
	goreq.DefaultTransport = client.Transport
	goreq.DefaultClient = client
}

// Get issues a GET to the specified URL. If reqOptions are given, these values are added to request headers
func (c *Client) Get(url string, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodGet, url, nil, reqOptions...)
}

// Post issues a POST to the specified URL, with `body` as payload. If reqOptions are given, these values are added to request headers
func (c *Client) Post(url string, body interface{}, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodPost, url, body, reqOptions...)
}

// Put issues a PUT to the specified URL, with `body` as payload. If reqOptions are given, these values are added to request headers
func (c *Client) Put(url string, body interface{}, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodPut, url, body, reqOptions...)
}

// Delete issues a DELETE to the specified URL. If reqOptions are given, these values are added to request headers
func (c *Client) Delete(url string, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodDelete, url, nil, reqOptions...)
}

func (c *Client) retry(method string, url string, body interface{}, reqOptions ...*requestOptions) (resp *goreq.Response, err error) {

	if c.TokenManager == nil {
		return nil, errors.New("Configure tokenManager or SetDefaultTokenManager")
	}

	var reqOption *requestOptions
	if len(reqOptions) > 0 {
		reqOption = reqOptions[0]
	}

	var originalBody []byte
	if originalBody, err = copyBody(body); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	for i := 1; i <= c.Options.MaxRetries; i++ {

		if len(originalBody) > 0 {
			bodyReader = bytes.NewBuffer(originalBody)
		}

		if resp, err = c.do(method, url, bodyReader, reqOption); err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusUnauthorized {
			return resp, nil
		}

		if i < c.Options.MaxRetries {
			c.TokenManager.ResetToken()
			time.Sleep(c.Options.Backoff(i))
		}
	}

	return resp, err
}

func (c *Client) do(method string, url string, body interface{}, reqOption *requestOptions) (*goreq.Response, error) {

	token, err := c.TokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	if c.Options.HystrixConfig == nil {
		return c.request(token.Authorization, method, url, body, reqOption)
	}

	if err := c.Options.HystrixConfig.valid(); err != nil {
		return nil, err
	}
	return c.requestHystrix(token.Authorization, method, url, body, reqOption)
}

func (c *Client) requestHystrix(authorization string, method string, url string, body interface{}, reqOption *requestOptions) (*goreq.Response, error) {

	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go(c.Options.HystrixConfig.Name, func() error {

		resp, err := c.request(authorization, method, url, body, reqOption)
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

func (c *Client) request(authorization string, method string, url string, body interface{}, reqOption *requestOptions) (*goreq.Response, error) {
	req := goreq.Request{
		Method:      method,
		ContentType: c.getContentType(),
		Uri:         url,
		Body:        body,
		Timeout:     c.Options.Timeout,
		ShowDebug:   c.Options.ShowDebug,
	}.WithHeader("Authorization", authorization)

	if reqOption != nil && reqOption.headers != nil {
		for _, header := range reqOption.headers {
			req.AddHeader(header.name, header.value)
		}
	}

	resp, err := req.Do()

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	return resp, nil
}

func (c *Client) getContentType() (contentType string) {
	contentType = c.Options.ContentType
	if contentType == "" {
		contentType = DefaultContentType
	}
	return contentType
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
