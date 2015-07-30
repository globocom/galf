package alf

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/franela/goreq"
)

type Client struct {
	httpClient   *http.Client
	tokenManager TokenManager
}

var defaultClient = &http.Client{}

func NewClient() *Client {
	return &Client{
		httpClient:   defaultClient,
		tokenManager: defaultTokenManager,
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

	//TODO: melhorar esse trecho
	var bodyCopy *bytes.Buffer
	if body != nil {
		bodyRead, _ := ioutil.ReadAll(body)
		body = bytes.NewBuffer(bodyRead)
		bodyCopy = bytes.NewBuffer(bodyRead)
	}

	resp, err = c.do(method, urlStr, body)
	if err != nil {
		return nil, err
	}

	//-----------------------------
	// caso o status code seja 401,
	// faz um retry
	//-----------------------------
	if resp.StatusCode == http.StatusUnauthorized {
		resp, err = c.do(method, urlStr, bodyCopy)
	}

	return resp, err
}

func (c *Client) do(method string, urlStr string, body io.Reader) (*goreq.Response, error) {
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	req := goreq.Request{
		Method:      method,
		ContentType: "application/json",
		Uri:         urlStr,
		Body:        body,
	}
	req.AddHeader("Authorization", token.Authorization)

	return req.Do()
}
