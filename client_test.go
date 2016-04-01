package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/franela/goreq"

	"gopkg.in/check.v1"
)

type clientSuite struct {
	server *httptest.Server
}

var _ = check.Suite(&clientSuite{})

func (cs *clientSuite) SetUpSuite(c *check.C) {
	cs.server = newTestServerToken()
}

func (cs *clientSuite) TearDownSuite(c *check.C) {
	cs.server.Close()
}

func (cs *clientSuite) TestGetClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"method": "GET"}`)
	})
	defer ts.Close()

	client := NewClient()
	url := fmt.Sprintf("%s/get/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	body, _ := resp.Body.ToString()
	c.Assert(body, check.Equals, `{"method": "GET"}`)
}

func (cs *clientSuite) TestPostClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"method": "POST"}`)
	})
	defer ts.Close()

	assertPost := func(resp *goreq.Response, err error) {
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, http.StatusCreated)

		body, _ := resp.Body.ToString()
		c.Assert(body, check.Equals, `{"method": "POST"}`)
	}

	client := NewClient()
	url := fmt.Sprintf("%s/post/feed/1", ts.URL)

	// body post == nil
	resp, err := client.Post(url, nil)
	assertPost(resp, err)

	// body post == io.Reader
	bodyReader := strings.NewReader(`{"body": "test"}`)
	resp, err = client.Post(url, bodyReader)
	assertPost(resp, err)

	// body post == string
	bodyString := "{'bodyPost': 'test'}"
	resp, err = client.Post(url, bodyString)
	assertPost(resp, err)

	// body post == []byte
	bodyBytes := []byte("{'bodyPost': 'test'}")
	resp, err = client.Post(url, bodyBytes)
	assertPost(resp, err)

}

func (cs *clientSuite) TestPutClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"method": "PUT"}`)
	})
	defer ts.Close()

	client := NewClient()
	url := fmt.Sprintf("%s/put/feed/1", ts.URL)
	resp, err := client.Delete(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	body, _ := resp.Body.ToString()
	c.Assert(body, check.Equals, `{"method": "PUT"}`)
}

func (cs *clientSuite) TestDeleteClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	client := NewClient()
	url := fmt.Sprintf("%s/delete/feed/1", ts.URL)
	resp, err := client.Delete(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusNoContent)

	body, _ := resp.Body.ToString()
	c.Assert(body, check.Equals, "")
}

func (cs *clientSuite) TestStatusUnauthorizedClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer ts.Close()

	client := NewClient()
	url := fmt.Sprintf("%s/StatusUnauthorized/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusUnauthorized)
}

func (cs *clientSuite) TestClientOptionsClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	clientOptions := ClientOptions{
		Timeout:       DefaultClientTimeout,
		MaxRetries:    DefaultClientMaxRetries,
		Backoff:       ConstantBackOff,
		ShowDebug:     false,
		HystrixConfig: nil,
	}

	client := NewClient(clientOptions)
	url := fmt.Sprintf("%s/ClientOptions/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
}

func (cs *clientSuite) TestHystrixClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"hystrix": "OK"}`)
	})
	defer ts.Close()

	getHystrixConfig := hystrix.CommandConfig{
		Timeout:                5000,
		SleepWindow:            2000,
		RequestVolumeThreshold: 50,
		MaxConcurrentRequests:  100,
	}

	HystrixConfigureCommand("getHystrixConfig", getHystrixConfig)
	clientOptions := NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		"getHystrixConfig",
	)

	client := NewClient(clientOptions)
	url := fmt.Sprintf("%s/ClientOptions/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	body, _ := resp.Body.ToString()
	c.Assert(body, check.Equals, `{"hystrix": "OK"}`)
}
