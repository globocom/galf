package galf

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/afex/hystrix-go/hystrix"

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

	assertPost := func(resp *Response, err error) {
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, http.StatusCreated)

		body, _ := resp.Body.ToString()
		c.Assert(body, check.Equals, `{"method": "POST"}`)
	}

	client := NewClient()
	url := fmt.Sprintf("%s/post/feed/1", ts.URL)

	resp, err := client.Post(url, nil)
	assertPost(resp, err)

	bodyReader := strings.NewReader(`{"body": "test"}`)
	resp, err = client.Post(url, bodyReader)
	assertPost(resp, err)

	bodyString := "{'bodyPost': 'test'}"
	resp, err = client.Post(url, bodyString)
	assertPost(resp, err)

	bodyBytes := []byte("{'bodyPost': 'test'}")
	resp, err = client.Post(url, bodyBytes)
	assertPost(resp, err)

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

	assertPut := func(resp *Response, err error) {
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

		body, _ := resp.Body.ToString()
		c.Assert(body, check.Equals, `{"method": "PUT"}`)
	}

	client := NewClient()
	url := fmt.Sprintf("%s/put/feed/1", ts.URL)

	resp, err := client.Put(url, nil)
	assertPut(resp, err)

	bodyReader := strings.NewReader(`{"bodyPut": "test"}`)
	resp, err = client.Put(url, bodyReader)
	assertPut(resp, err)

	bodyString := "{'bodyPut': 'test'}"
	resp, err = client.Put(url, bodyString)
	assertPut(resp, err)

	bodyBytes := []byte("{'bodyPut': 'test'}")
	resp, err = client.Put(url, bodyBytes)
	assertPut(resp, err)

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

	clientOptions := ClientOptions{
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

func (cs *clientSuite) TestGetClientRequestOptionsHeader(c *check.C) {
	ts := newTestServerCustom(func(rw http.ResponseWriter, r *http.Request) {
		responseHeaders(rw, r)
		rw.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	url := fmt.Sprintf("%s/requestOptions/feed/1", ts.URL)

	requestOptions := NewRequestOptions()
	requestOptions.AddHeader("token", "1234567890")

	client := NewClient()
	resp, err := client.Get(url, requestOptions)

	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
	c.Assert(resp.Header.Get("token"), check.Equals, "1234567890")
}

func (cs *clientSuite) TestGetClientRequestOptionsHeaders(c *check.C) {
	ts := newTestServerCustom(func(rw http.ResponseWriter, r *http.Request) {
		responseHeaders(rw, r)
		rw.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	url := fmt.Sprintf("%s/requestOptions/feed/2", ts.URL)

	headers := map[string]string{
		"header1": "123",
		"header2": "456",
	}
	requestOptions := NewRequestOptions()
	requestOptions.AddHeaders(headers)

	client := NewClient()
	resp, err := client.Get(url, requestOptions)

	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
	c.Assert(resp.Header.Get("header1"), check.Equals, "123")
	c.Assert(resp.Header.Get("header2"), check.Equals, "456")
}

func (cs *clientSuite) TestSetCustomHTTPClient(c *check.C) {
	originalTransport := defaultTransport
	originalClient := httpClientInstance

	defer func() {
		defaultTransport = originalTransport
		httpClientInstance = originalClient
	}()

	customClient := &http.Client{Timeout: time.Hour}
	SetCustomHTTPClient(customClient)

	c.Assert(httpClientInstance, check.Not(check.Equals), originalClient)
	c.Assert(httpClientInstance.Timeout, check.Equals, time.Hour)
}

func (cs *clientSuite) TestKeepAliveConnections(c *check.C) {
	var connectionCounter int32

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=30, max=1000")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status": "ok"}`)
	}))

	countingListener := &countingListener{
		Listener: ts.Listener,
		counter:  &connectionCounter,
	}

	ts.Listener = countingListener
	ts.Start()
	defer ts.Close()

	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		ForceAttemptHTTP2:     false,
	}

	customClient := &http.Client{Transport: transport}
	SetCustomHTTPClient(customClient)

	client := NewClient()
	url := ts.URL + "/connection-reuse-test"

	numRequests := 10
	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(url)
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

		body, err := resp.Body.ToString()
		c.Assert(err, check.IsNil)
		c.Assert(body, check.Equals, `{"status": "ok"}`)

		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	connCount := atomic.LoadInt32(&connectionCounter)
	c.Logf("Made %d requests with %d connections", numRequests, connCount)

	c.Assert(connCount < int32(numRequests), check.Equals, true)
}

type countingListener struct {
	net.Listener
	counter *int32
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		atomic.AddInt32(l.counter, 1)
	}
	return conn, err
}

func responseHeaders(rw http.ResponseWriter, r *http.Request) {
	for name, headers := range r.Header {
		for _, v := range headers {
			rw.Header().Set(name, v)
		}
	}
}

func (cs *clientSuite) TestBackstageScenario(c *check.C) {
	type testMetrics struct {
		totalRequests   int
		successRequests int32
		failedRequests  int32
		startTime       time.Time
	}

	metrics := testMetrics{
		totalRequests: 100000,
		startTime:     time.Now(),
	}

	var connectionCounter int32

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=60, max=10000")

		path := r.URL.Path
		if strings.Contains(path, "/error") {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "Something went wrong"}`)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": 123, "name": "backstage-item", "status": "active"}`)
	}))

	countingListener := &countingListener{
		Listener: ts.Listener,
		counter:  &connectionCounter,
	}

	ts.Listener = countingListener
	ts.Start()
	defer ts.Close()

	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 60 * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
		MaxConnsPerHost:       1000,
	}

	customClient := &http.Client{Transport: transport}
	SetCustomHTTPClient(customClient)

	clientOptions := ClientOptions{
		Timeout:    5 * time.Second,
		MaxRetries: 2,
		KeepAliveConfig: &KeepAliveConfig{
			Enabled:             true,
			KeepAliveTime:       60 * time.Second,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
		},
		EnableMetrics: true,
	}

	client := NewClient(clientOptions)
	baseURL := ts.URL

	const numWorkers = 50
	requestChan := make(chan int, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				var url string
				if rand.Intn(10) < 1 {
					url = fmt.Sprintf("%s/error/%d", baseURL, rand.Intn(10000))
				} else {
					url = fmt.Sprintf("%s/item/%d", baseURL, rand.Intn(10000))
				}

				resp, err := client.Get(url)

				if err != nil {
					atomic.AddInt32(&metrics.failedRequests, 1)
					continue
				}

				_, _ = resp.Body.ToString()
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt32(&metrics.successRequests, 1)
				} else {
					atomic.AddInt32(&metrics.failedRequests, 1)
				}

				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
			}
		}()
	}

	for i := 0; i < metrics.totalRequests; i++ {
		requestChan <- i
	}

	close(requestChan)
	wg.Wait()

	duration := time.Since(metrics.startTime)
	finalConnectionCount := atomic.LoadInt32(&connectionCounter)
	successCount := atomic.LoadInt32(&metrics.successRequests)
	failedCount := atomic.LoadInt32(&metrics.failedRequests)

	connectionsReused := metrics.totalRequests - int(finalConnectionCount)
	reusePercentage := float64(connectionsReused) * 100 / float64(metrics.totalRequests)
	requestsPerConnection := float64(metrics.totalRequests) / float64(finalConnectionCount)

	c.Logf("BACKSTAGE TEST RESULTS - %d requests in %v:\n"+
		"  • Successful requests: %d\n"+
		"  • Failed requests: %d\n"+
		"  • TCP connections opened: %d\n"+
		"  • Connection reuse: %.2f%%\n"+
		"  • Requests per connection: %.2f\n"+
		"  • Avg response time: %v",
		metrics.totalRequests, duration,
		successCount, failedCount,
		finalConnectionCount,
		reusePercentage,
		requestsPerConnection,
		client.GetAverageResponseTime())

	c.Assert(finalConnectionCount < int32(metrics.totalRequests/10), check.Equals, true,
		check.Commentf("Expected at least 90%% connection reuse, but got only %.1f%%", reusePercentage))

	c.Assert(successCount > 0, check.Equals, true)
}

func (c *Client) GetAverageResponseTime() time.Duration {
	requests := atomic.LoadInt64(&c.metrics.requestCount)
	if requests == 0 {
		return 0
	}

	totalTime := atomic.LoadInt64((*int64)(&c.metrics.responseTimeTotal))
	return time.Duration(totalTime) / time.Duration(requests)
}

func (cs *clientSuite) TestKeepAliveComparison(c *check.C) {
	c.Log("Starting Keep-Alive Comparison Test...")

	serverHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Keep-Alive", "timeout=60, max=10000")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": 123, "status": "ok"}`)
	}

	runTest := func(keepAliveEnabled bool) (int, int32, time.Duration) {
		var connectionCounter int32
		ts := httptest.NewUnstartedServer(http.HandlerFunc(serverHandler))

		countingListener := &countingListener{
			Listener: ts.Listener,
			counter:  &connectionCounter,
		}

		ts.Listener = countingListener
		ts.Start()
		defer ts.Close()

		c.Logf("Running test with keep-alive %s", map[bool]string{true: "ENABLED", false: "DISABLED"}[keepAliveEnabled])

		dialer := &net.Dialer{
			Timeout:   time.Second,
			KeepAlive: 30 * time.Second,
		}

		transport := &http.Transport{
			DialContext:           dialer.DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     !keepAliveEnabled,
		}

		customClient := &http.Client{Transport: transport}
		SetCustomHTTPClient(customClient)

		var keepAliveConfig *KeepAliveConfig
		if keepAliveEnabled {
			keepAliveConfig = &KeepAliveConfig{
				Enabled:             true,
				KeepAliveTime:       30 * time.Second,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			}
		} else {
			keepAliveConfig = &KeepAliveConfig{
				Enabled: false,
			}
		}

		clientOptions := ClientOptions{
			Timeout:         5 * time.Second,
			MaxRetries:      1,
			EnableMetrics:   true,
			KeepAliveConfig: keepAliveConfig,
		}

		client := NewClient(clientOptions)

		const totalRequests = 1000
		const numWorkers = 10

		requestChan := make(chan int, numWorkers)
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range requestChan {
					path := fmt.Sprintf("/test/%d", rand.Intn(100))
					url := ts.URL + path

					resp, err := client.Get(url)
					if err == nil {
						_, _ = resp.Body.ToString()
					}

					time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				}
			}()
		}

		startTime := time.Now()

		for i := 0; i < totalRequests; i++ {
			requestChan <- i
		}

		close(requestChan)
		wg.Wait()

		duration := time.Since(startTime)

		c.Logf("  Test completed: %d requests, %d connections in %v",
			totalRequests, atomic.LoadInt32(&connectionCounter), duration)

		return totalRequests, atomic.LoadInt32(&connectionCounter), duration
	}

	c.Log("=== TESTING WITH KEEP-ALIVE ENABLED ===")
	totalRequests, withKeepAliveConnections, withKeepAliveDuration := runTest(true)

	c.Log("=== TESTING WITH KEEP-ALIVE DISABLED ===")
	_, withoutKeepAliveConnections, withoutKeepAliveDuration := runTest(false)

	withKeepAliveReused := totalRequests - int(withKeepAliveConnections)
	withKeepAliveReusePercentage := float64(withKeepAliveReused) * 100 / float64(totalRequests)
	withKeepAliveReqPerConn := float64(totalRequests) / float64(withKeepAliveConnections)

	withoutKeepAliveReused := totalRequests - int(withoutKeepAliveConnections)
	withoutKeepAliveReusePercentage := float64(withoutKeepAliveReused) * 100 / float64(totalRequests)
	withoutKeepAliveReqPerConn := float64(totalRequests) / float64(withoutKeepAliveConnections)

	speedImprovement := float64(withoutKeepAliveDuration) / float64(withKeepAliveDuration)

	resultStr := fmt.Sprintf("\nKEEP-ALIVE CONNECTION REUSE COMPARISON (%d requests):\n\n"+
		"WITH KEEP-ALIVE:\n"+
		"  • TCP connections opened: %d\n"+
		"  • Connection reuse: %.2f%%\n"+
		"  • Requests per connection: %.2f\n"+
		"  • Total duration: %v\n\n"+
		"WITHOUT KEEP-ALIVE:\n"+
		"  • TCP connections opened: %d\n"+
		"  • Connection reuse: %.2f%%\n"+
		"  • Requests per connection: %.2f\n"+
		"  • Total duration: %v\n\n"+
		"PERFORMANCE IMPACT:\n"+
		"  • Speed improvement with keep-alive: %.2fx faster\n"+
		"  • Connections saved: %d\n",
		totalRequests,
		withKeepAliveConnections, withKeepAliveReusePercentage, withKeepAliveReqPerConn, withKeepAliveDuration,
		withoutKeepAliveConnections, withoutKeepAliveReusePercentage, withoutKeepAliveReqPerConn, withoutKeepAliveDuration,
		speedImprovement, withoutKeepAliveConnections-withKeepAliveConnections)

	c.Log("=== KEEP-ALIVE TEST RESULTS ===")
	c.Log(resultStr)
	fmt.Fprintln(os.Stderr, "\n"+resultStr)

	os.WriteFile("keepalive_test_results.txt", []byte(resultStr), 0644)

	c.Assert(withKeepAliveConnections < withoutKeepAliveConnections/2, check.Equals, true,
		check.Commentf("Keep-alive not working properly.\n\n%s", resultStr))

	c.Assert(withKeepAliveDuration < withoutKeepAliveDuration, check.Equals, true,
		check.Commentf("Keep-alive did not improve performance"))

	c.Log("Keep-alive test completed successfully. Full results in keepalive_test_results.txt")
}
