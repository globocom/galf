package galf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/globocom/goreq"
)

type (
	Client struct {
		TokenManager TokenManager
		Options      ClientOptions
		clientHTTP   goreq.Client
	}
)

func init() {}

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
		clientHTTP: goreq.NewClient(goreq.Options{
			Timeout:             options.Timeout,
			MaxIdleConnsPerHost: 250,
		}),
	}
}

func (c *Client) Get(url string, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodGet, url, nil, reqOptions...)
}

func (c *Client) Post(url string, body interface{}, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodPost, url, body, reqOptions...)
}

func (c *Client) Put(url string, body interface{}, reqOptions ...*requestOptions) (*goreq.Response, error) {
	return c.retry(http.MethodPut, url, body, reqOptions...)
}

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
		ShowDebug:   c.Options.ShowDebug,
	}
	req.AddHeader("Authorization", authorization)

	if reqOption != nil && reqOption.headers != nil {
		for _, header := range reqOption.headers {
			req.AddHeader(header.name, header.value)
		}
	}

	resp, err := c.clientHTTP.Do(req)

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
	switch v := b.(type) {
	case string:
		return []byte(v), nil

	case io.Reader:
		var originalBody bytes.Buffer
		_, err := io.Copy(&originalBody, v.(io.Reader))
		if err != nil {
			return nil, err
		}
		return originalBody.Bytes(), nil

	case []byte:
		return v, nil

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
