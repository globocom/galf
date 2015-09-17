package galf

import "time"

const (
	DefaultTokenMaxRetries    = 2
	DefaultTokenClientTimeout = 1 * time.Second
)

type (
	TokenOptions struct {
		Timeout       time.Duration
		Backoff       BackoffStrategy
		MaxRetries    int
		ShowDebug     bool
		HystrixConfig *HystrixConfig
		useHystrix    bool
	}
)

var (
	defaultTokenOptions = TokenOptions{
		Timeout:       DefaultTokenClientTimeout,
		MaxRetries:    DefaultTokenMaxRetries,
		Backoff:       ConstantBackOff,
		ShowDebug:     false,
		HystrixConfig: nil,
	}
)

func NewTokenOptions(timeout time.Duration, debug bool, maxRetries int, circuitName string, backoff ...BackoffStrategy) TokenOptions {
	tokenBackoff := ConstantBackOff
	if len(backoff) > 0 {
		tokenBackoff = backoff[0]
	}

	return TokenOptions{
		Timeout:       timeout,
		ShowDebug:     debug,
		MaxRetries:    maxRetries,
		Backoff:       tokenBackoff,
		HystrixConfig: &HystrixConfig{Name: circuitName},
	}
}
