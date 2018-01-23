package galf

import "time"

const (
	// DefaultClientTimeout defines the default timeout used when a request is issued
	DefaultClientTimeout = 20 * time.Second
	// DefaultClientMaxRetries defines how many times a request should be attempted by default
	DefaultClientMaxRetries = 2
	// DefaultContentType defines the default value to the Content-Type header
	DefaultContentType = "application/json"
)

// ClientOptions maps the configurations that can be used to instantiate a Client
type ClientOptions struct {
	ContentType   string
	Timeout       time.Duration
	Backoff       BackoffStrategy
	MaxRetries    int
	ShowDebug     bool
	HystrixConfig *HystrixConfig
}

var (
	defaultClientOptions = ClientOptions{
		ContentType:   DefaultContentType,
		Timeout:       DefaultClientTimeout,
		MaxRetries:    DefaultClientMaxRetries,
		Backoff:       ConstantBackOff,
		ShowDebug:     false,
		HystrixConfig: nil,
	}
)

// NewClientOptions creates a new instance of ClientOptions with the supplied values
func NewClientOptions(timeout time.Duration, debug bool, maxRetries int, hystrixConfigName string, backoff ...BackoffStrategy) ClientOptions {
	clientBackoff := ConstantBackOff
	if len(backoff) > 0 {
		clientBackoff = backoff[0]
	}

	var hystrixConfig *HystrixConfig
	if hystrixConfigName != "" {
		hystrixConfig = NewHystrixConfig(hystrixConfigName)
	}

	return ClientOptions{
		Timeout:       timeout,
		ShowDebug:     debug,
		MaxRetries:    maxRetries,
		Backoff:       clientBackoff,
		HystrixConfig: hystrixConfig,
	}
}
