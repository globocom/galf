// +build !race

/*
* Go OAuth2 Client
*
* MIT License
*
* Copyright (c) 2015 Globo.com
 */

package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	"github.com/afex/hystrix-go/hystrix"
	check "gopkg.in/check.v1"
)

type clientNoRaceSuite struct {
	server *httptest.Server
}

var _ = check.Suite(&clientNoRaceSuite{})

func (cs *clientNoRaceSuite) SetUpSuite(c *check.C) {
	cs.server = newTestServerToken()
}

func (cs *clientNoRaceSuite) TearDownSuite(c *check.C) {
	cs.server.Close()
}

func (cs *clientNoRaceSuite) TestHystrixMultithreadedClient(c *check.C) {
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
