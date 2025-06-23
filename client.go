package galf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
)

// Client is the galf client's structure
type Client struct {
	TokenManager TokenManager
	Options      ClientOptions
	transport    http.RoundTripper
	httpClient   *http.Client
	metrics      struct {
		requestCount      int64
		connectionErrors  int64
		responseTimeTotal time.Duration
	}
}

var (
	defaultDialer = &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 30 * time.Second,
	}
	defaultTransport http.RoundTripper = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           defaultDialer.DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   250,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}
	httpClientInstance = &http.Client{Transport: defaultTransport}
)

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
	client := &Client{
		TokenManager: tokenManager,
		Options:      options,
	}

	if options.KeepAliveConfig != nil {
		client.transport = client.createTransport()
		client.httpClient = &http.Client{
			Transport: client.transport,
			Timeout:   options.Timeout,
		}
	}

	return client
}

// SetCustomHTTPClient allow the override of both http Client and Transport.
// The new custom client's transport is used as the new transport
func SetCustomHTTPClient(client *http.Client) {
	httpClientInstance = client
	defaultTransport = client.Transport
}

// Get issues a GET to the specified URL. If reqOptions are given, these values are added to request headers
func (c *Client) Get(url string, reqOptions ...*requestOptions) (*Response, error) {
	return c.retry(http.MethodGet, url, nil, reqOptions...)
}

// Post issues a POST to the specified URL, with `body` as payload. If reqOptions are given, these values are added to request headers
func (c *Client) Post(url string, body interface{}, reqOptions ...*requestOptions) (*Response, error) {
	return c.retry(http.MethodPost, url, body, reqOptions...)
}

// Put issues a PUT to the specified URL, with `body` as payload. If reqOptions are given, these values are added to request headers
func (c *Client) Put(url string, body interface{}, reqOptions ...*requestOptions) (*Response, error) {
	return c.retry(http.MethodPut, url, body, reqOptions...)
}

// Delete issues a DELETE to the specified URL. If reqOptions are given, these values are added to request headers
func (c *Client) Delete(url string, reqOptions ...*requestOptions) (*Response, error) {
	return c.retry(http.MethodDelete, url, nil, reqOptions...)
}

func (c *Client) retry(method string, url string, body interface{}, reqOptions ...*requestOptions) (resp *Response, err error) {
	if c.TokenManager == nil {
		return nil, errors.New("configure tokenManager or SetDefaultTokenManager")
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

func (c *Client) do(method string, url string, body interface{}, reqOption *requestOptions) (*Response, error) {
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

func (c *Client) requestHystrix(authorization string, method string, url string, body interface{}, reqOption *requestOptions) (*Response, error) {
	output := make(chan *Response, 1)
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

func (c *Client) createTransport() http.RoundTripper {
	keepAliveConfig := c.Options.KeepAliveConfig
	if keepAliveConfig == nil {
		keepAliveConfig = DefaultKeepAliveConfig()
	}

	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: keepAliveConfig.KeepAliveTime,
	}

	return &http.Transport{
		DialContext:           dialer.DialContext,
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          keepAliveConfig.MaxIdleConns,
		MaxIdleConnsPerHost:   keepAliveConfig.MaxIdleConnsPerHost,
		IdleConnTimeout:       keepAliveConfig.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     !keepAliveConfig.Enabled,
		MaxConnsPerHost:       keepAliveConfig.MaxIdleConnsPerHost * 2,
		WriteBufferSize:       4096,
		ReadBufferSize:        4096,
	}
}

func (c *Client) request(authorization string, method string, url string, body interface{}, reqOption *requestOptions) (*Response, error) {
	var startTime time.Time
	if c.Options.EnableMetrics {
		startTime = time.Now()
		atomic.AddInt64(&c.metrics.requestCount, 1)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	contentType := c.getContentType()
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Connection", "keep-alive")

	if reqOption != nil && reqOption.headers != nil {
		for _, header := range reqOption.headers {
			req.Header.Set(header.name, header.value)
		}
	}

	if body != nil {
		bodyReader, err := prepareRequestBody(body)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		req.Body = io.NopCloser(bodyReader)
		if rc, ok := body.(io.ReadCloser); ok {
			req.Body = rc
		}

		if sizedReader, ok := bodyReader.(interface{ Len() int }); ok {
			req.ContentLength = int64(sizedReader.Len())
		}
	}

	var httpClient *http.Client
	if c.httpClient == nil {
		if c.transport == nil {
			if c.Options.KeepAliveConfig != nil {
				c.transport = c.createTransport()
			} else {
				c.transport = defaultTransport
			}
		}

		c.httpClient = &http.Client{
			Transport: c.transport,
			Timeout:   c.Options.Timeout,
		}
	}
	httpClient = c.httpClient

	resp, err := httpClient.Do(req)
	if err != nil {
		if c.Options.EnableMetrics {
			atomic.AddInt64(&c.metrics.connectionErrors, 1)
		}
		return nil, stackerr.Wrap(err)
	}

	if c.Options.EnableMetrics {
		elapsed := time.Since(startTime)
		atomic.AddInt64((*int64)(&c.metrics.responseTimeTotal), int64(elapsed))
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       Body{content: resp.Body},
		raw:        resp,
	}, nil
}

func (c *Client) getContentType() (contentType string) {
	contentType = c.Options.ContentType
	if contentType == "" {
		contentType = DefaultContentType
	}
	return contentType
}

func prepareRequestBody(body interface{}) (io.Reader, error) {
	switch b := body.(type) {
	case string:
		return bytes.NewReader([]byte(b)), nil
	case []byte:
		return bytes.NewReader(b), nil
	case io.Reader:
		return b, nil
	case nil:
		return nil, nil
	default:
		j, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(j), nil
	}
}

func copyBody(b interface{}) ([]byte, error) {
	switch b := b.(type) {
	case string:
		return []byte(b), nil
	case io.Reader:
		var originalBody bytes.Buffer
		_, err := io.Copy(&originalBody, b)
		if err != nil {
			return nil, err
		}
		return originalBody.Bytes(), nil
	case []byte:
		return b, nil
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
