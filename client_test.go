package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

	// body post == struct
	type bodySt struct {
		BodyPost string `json:"bodyPost"`
	}
	bodyStruct := bodySt{"test"}
	resp, err = client.Post(url, bodyStruct)
	assertPost(resp, err)
}

func (cs *clientSuite) TestPutClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"method": "PUT"}`)
	})
	defer ts.Close()

	assertPut := func(resp *goreq.Response, err error) {
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

		body, _ := resp.Body.ToString()
		c.Assert(body, check.Equals, `{"method": "PUT"}`)
	}

	client := NewClient()
	url := fmt.Sprintf("%s/put/feed/1", ts.URL)

	// body put == nil
	resp, err := client.Put(url, nil)
	assertPut(resp, err)

	// body put == io.Reader
	bodyReader := strings.NewReader(`{"bodyPut": "test"}`)
	resp, err = client.Put(url, bodyReader)
	assertPut(resp, err)

	// body put == string
	bodyString := "{'bodyPut': 'test'}"
	resp, err = client.Put(url, bodyString)
	assertPut(resp, err)

	// body put == []byte
	bodyBytes := []byte("{'bodyPut': 'test'}")
	resp, err = client.Put(url, bodyBytes)
	assertPut(resp, err)

	// body put == struct
	type bodySt struct {
		BodyPut string `json:"bodyPut"`
	}
	bodyStruct := bodySt{"test"}
	resp, err = client.Put(url, bodyStruct)
	assertPut(resp, err)
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

func (cs *clientSuite) TestDefaultClientOptionsClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.Header.Get("Content-Type")+"; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	client := NewClient()
	url := fmt.Sprintf("%s/ClientOptions/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
	c.Assert(resp.Header.Get("Content-Type"), check.Equals, "application/json; charset=utf-8")

	clientOptions = ClientOptions{
		Timeout:       DefaultClientTimeout,
		MaxRetries:    DefaultClientMaxRetries,
		Backoff:       ConstantBackOff,
		ShowDebug:     false,
		HystrixConfig: nil,
	}

	client = NewClient(clientOptions)
	url = fmt.Sprintf("%s/ClientOptions/feed/1", ts.URL)
	resp, err = client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
	c.Assert(resp.Header.Get("Content-Type"), check.Equals, "application/json; charset=utf-8")
}

func (cs *clientSuite) TestClientOptionsClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.Header.Get("Content-Type")+"; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	clientOptions := ClientOptions{
		ContentType:   "application/my-custom-type",
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
	c.Assert(resp.Header.Get("Content-Type"), check.Equals, "application/my-custom-type; charset=utf-8")
}

func (cs *clientSuite) TestHystrixClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"hystrix": "OK"}`)
	})
	defer ts.Close()

	hystrixConfig := hystrix.CommandConfig{
		Timeout:                5000,
		SleepWindow:            2000,
		RequestVolumeThreshold: 50,
		MaxConcurrentRequests:  100,
	}

	hystrixConfigName := "hystrixConfigName"
	HystrixConfigureCommand(hystrixConfigName, hystrixConfig)
	clientOptions := NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		hystrixConfigName,
	)

	client := NewClient(clientOptions)
	url := fmt.Sprintf("%s/hystrixconfig/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	body, _ := resp.Body.ToString()
	c.Assert(body, check.Equals, `{"hystrix": "OK"}`)
}

func (cs *clientSuite) TestHystrixConfigTimeoutClient(c *check.C) {
	timeout := 200

	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(timeout+10) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"hystrix": "OK"}`)
	})
	defer ts.Close()

	hystrixConfig := hystrix.CommandConfig{
		Timeout:                timeout,
		SleepWindow:            2000,
		RequestVolumeThreshold: 50,
		MaxConcurrentRequests:  100,
	}

	hystrixConfigName := "hystrixConfigTimeout"
	HystrixConfigureCommand(hystrixConfigName, hystrixConfig)
	clientOptions := NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		hystrixConfigName,
	)

	client := NewClient(clientOptions)
	url := fmt.Sprintf("%s/hystrixconfigtimeout/feed/1", ts.URL)
	resp, err := client.Get(url)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "hystrix: timeout")
	c.Assert(resp, check.IsNil)
}

func (cs *clientSuite) TestHystrixConfigNotFoundClient(c *check.C) {
	hystrixConfigName := "hystrixConfigTimeout"
	clientOptions := NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		hystrixConfigName,
	)

	client := NewClient(clientOptions)
	resp, err := client.Get("/hystrixconfignotfound/feed/1")
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "Hystrix config name not found: "+hystrixConfigName)
	c.Assert(resp, check.IsNil)
}

func (cs *clientSuite) TestHystrixMultithreadedClient(c *check.C) {
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"hystrix": "OK"}`)
	})
	defer ts.Close()

	maxConcurrentRequests := 5
	hystrixConfig := hystrix.CommandConfig{
		Timeout:                1000,
		SleepWindow:            2000,
		RequestVolumeThreshold: 100,
		MaxConcurrentRequests:  maxConcurrentRequests,
	}

	hystrixConfigName := "hystrixConfigNameMultithreaded"
	HystrixConfigureCommand(hystrixConfigName, hystrixConfig)
	clientOptions := NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		hystrixConfigName,
	)

	exceedRequests := 3
	numThreads := maxConcurrentRequests + exceedRequests
	var numCreates int32
	var finishLine sync.WaitGroup
	finishLine.Add(numThreads)
	client := NewClient(clientOptions)
	for i := 0; i < numThreads; i++ {
		go func() {
			defer finishLine.Done()

			url := fmt.Sprintf("%s/hystrixmultithreaded/feed/1", ts.URL)
			resp, err := client.Get(url)
			if err != nil {
				atomic.AddInt32(&numCreates, 1)
				c.Assert(err.Error(), check.Equals, "hystrix: max concurrency")
				c.Assert(resp, check.IsNil)
			} else {
				c.Assert(err, check.IsNil)
				c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
				body, _ := resp.Body.ToString()
				c.Assert(body, check.Equals, `{"hystrix": "OK"}`)
			}
		}()
	}
	finishLine.Wait()

	c.Assert(numCreates, check.Equals, int32(exceedRequests))
}
