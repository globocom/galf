package galf

import "time"

const (
	DefaultClientTimeout    = 20 * time.Second
	DefaultClientMaxRetries = 2
	DefaultContentType      = "application/json"
)

type (
	ClientOptions struct {
		ContentType   string
		Timeout       time.Duration
		Backoff       BackoffStrategy
		MaxRetries    int
		ShowDebug     bool
		HystrixConfig *HystrixConfig
	}
)

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
