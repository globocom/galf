package galf

import "time"

const (
	// DefaultClientTimeout defines the default timeout used when a request is issued
	DefaultClientTimeout = 20 * time.Second
	// DefaultClientMaxRetries defines how many times a request should be attempted by default
	DefaultClientMaxRetries = 2
	// DefaultContentType defines the default value to the Content-Type header
	DefaultContentType = "application/json"
	// DefaultKeepAliveEnabled defines if keep-alive connections are enabled by default
	DefaultKeepAliveEnabled = true
	// DefaultKeepAliveTime defines how long to keep a connection alive by default
	DefaultKeepAliveTime = 30 * time.Second
	// DefaultMaxIdleConns defines the maximum number of idle connections
	DefaultMaxIdleConns = 100
	// DefaultMaxIdleConnsPerHost defines the maximum number of idle connections per host
	DefaultMaxIdleConnsPerHost = 250
	// DefaultIdleConnTimeout defines how long an idle connection is kept in the pool
	DefaultIdleConnTimeout = 90 * time.Second
)

// KeepAliveConfig holds the configuration for HTTP connection reuse
type KeepAliveConfig struct {
	// Enabled determines if keep-alive is used for connections
	Enabled bool
	// KeepAliveTime is how long to keep a connection alive
	KeepAliveTime time.Duration
	// MaxIdleConns is the maximum number of idle connections across all hosts
	MaxIdleConns int
	// MaxIdleConnsPerHost is the maximum number of idle connections per host
	MaxIdleConnsPerHost int
	// IdleConnTimeout is how long an idle connection remains in the pool
	IdleConnTimeout time.Duration
}

// ClientOptions maps the configurations that can be used to instantiate a Client
type ClientOptions struct {
	ContentType     string
	Timeout         time.Duration
	Backoff         BackoffStrategy
	MaxRetries      int
	ShowDebug       bool
	HystrixConfig   *HystrixConfig
	KeepAliveConfig *KeepAliveConfig
	EnableMetrics   bool
}

// DefaultKeepAliveConfig returns the default configuration for connection reuse
func DefaultKeepAliveConfig() *KeepAliveConfig {
	return &KeepAliveConfig{
		Enabled:             DefaultKeepAliveEnabled,
		KeepAliveTime:       DefaultKeepAliveTime,
		MaxIdleConns:        DefaultMaxIdleConns,
		MaxIdleConnsPerHost: DefaultMaxIdleConnsPerHost,
		IdleConnTimeout:     DefaultIdleConnTimeout,
	}
}

var (
	defaultClientOptions = ClientOptions{
		ContentType:     DefaultContentType,
		Timeout:         DefaultClientTimeout,
		MaxRetries:      DefaultClientMaxRetries,
		Backoff:         ConstantBackOff,
		ShowDebug:       false,
		HystrixConfig:   nil,
		KeepAliveConfig: DefaultKeepAliveConfig(),
		EnableMetrics:   false,
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
		Timeout:         timeout,
		ShowDebug:       debug,
		MaxRetries:      maxRetries,
		Backoff:         clientBackoff,
		HystrixConfig:   hystrixConfig,
		KeepAliveConfig: DefaultKeepAliveConfig(),
		EnableMetrics:   false,
	}
}

// WithKeepAlive configures keep-alive settings for the HTTP client
func (c ClientOptions) WithKeepAlive(enabled bool, keepAliveTime, idleConnTimeout time.Duration, maxIdleConns, maxIdleConnsPerHost int) ClientOptions {
	c.KeepAliveConfig = &KeepAliveConfig{
		Enabled:             enabled,
		KeepAliveTime:       keepAliveTime,
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
	}
	return c
}

// WithoutKeepAlive disables keep-alive for this client's connections
func (c ClientOptions) WithoutKeepAlive() ClientOptions {
	c.KeepAliveConfig = &KeepAliveConfig{
		Enabled: false,
	}
	return c
}

// WithMetrics enables collection of request metrics
func (c ClientOptions) WithMetrics() ClientOptions {
	c.EnableMetrics = true
	return c
}

// WithContentType sets a custom content type for the requests
func (c ClientOptions) WithContentType(contentType string) ClientOptions {
	c.ContentType = contentType
	return c
}
