package galf

import "time"

const (
	DefaultClientTimeout    = 20 * time.Second
	DefaultClientMaxRetries = 2
)

type (
	ClientOptions struct {
		Timeout       time.Duration
		Backoff       BackoffStrategy
		MaxRetries    int
		ShowDebug     bool
		HystrixConfig *HystrixConfig
	}
)

var (
	defaultClientOptions = ClientOptions{
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

	return ClientOptions{
		Timeout:       timeout,
		ShowDebug:     debug,
		MaxRetries:    maxRetries,
		Backoff:       clientBackoff,
		HystrixConfig: &HystrixConfig{Name: hystrixConfigName},
	}
}
