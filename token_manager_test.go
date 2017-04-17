package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/afex/hystrix-go/hystrix"

	check "gopkg.in/check.v1"
)

type tokenManagerSuite struct {
	server *httptest.Server
}

var (
	bsGatewayTokenConfig = hystrix.CommandConfig{
		Timeout:                5000,
		SleepWindow:            2000,
		RequestVolumeThreshold: 50,
		MaxConcurrentRequests:  100,
	}

	bsGatewayToken = "BSGatewayToken"

	tokenOptions = NewTokenOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		bsGatewayToken,
	)
)

var _ = check.Suite(&tokenManagerSuite{})

func (tms *tokenManagerSuite) TestTokenManagerWithHystrix(c *check.C) {
	var expire = 100

	HystrixConfigureCommand(bsGatewayToken, bsGatewayTokenConfig)
	ts := newTestServerCustom(handleToken(expire))
	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
		tokenOptions,
	)

	c.Assert(tm.Authorization, check.Equals, "Basic Q2xpZW50SWQ6Q2xpZW50U2VjcmV0")

	token, err := tm.GetToken()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.NotNil)
	c.Assert(token.TokenType, check.Equals, "Bearer")
	c.Assert(token.isValid(), check.Equals, true)
}

func (tms *tokenManagerSuite) TestTokenManagerInvalid(c *check.C) {
	var expire = 0

	HystrixConfigureCommand(bsGatewayToken, bsGatewayTokenConfig)
	ts := newTestServerCustom(handleToken(expire))
	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
		tokenOptions,
	)

	token, err := tm.GetToken()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.NotNil)
	c.Assert(token.TokenType, check.Equals, "Bearer")
	c.Assert(token.isValid(), check.Equals, false)
}

func (tms *tokenManagerSuite) TestTokenManagerRetryFail(c *check.C) {
	var retries = 0

	HystrixConfigureCommand(bsGatewayToken, bsGatewayTokenConfig)
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		retries = retries + 1
		w.WriteHeader(http.StatusBadGateway)
	})

	tokenOptionsLinear := NewTokenOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		bsGatewayToken,
		LinearBackoff,
	)

	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
		tokenOptionsLinear,
	)

	token, err := tm.GetToken()
	c.Assert(err, check.NotNil)
	c.Assert(err, check.ErrorMatches, "Failed to request token .*")
	c.Assert(token, check.IsNil)
	c.Assert(retries, check.Equals, DefaultTokenMaxRetries)
}

func (tms *tokenManagerSuite) TestTokenManagerRetryOk(c *check.C) {
	var retries = 0

	HystrixConfigureCommand(bsGatewayToken, bsGatewayTokenConfig)
	ts := newTestServerCustom(func(w http.ResponseWriter, r *http.Request) {
		retries = retries + 1
		if retries > 1 {
			fmt.Fprint(w, fmt.Sprintf(`{"access_token": "nonenoenoe", "token_type": "bearer", "expires_in": %d}`, 100))
		} else {
			w.WriteHeader(http.StatusBadGateway)
		}
	})

	var tokenOptionsExponential = tokenOptions
	tokenOptionsExponential.Backoff = ExponentialBackoff

	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
		tokenOptionsExponential,
	)

	token, err := tm.GetToken()
	c.Assert(err, check.IsNil)
	c.Assert(token, check.NotNil)
	c.Assert(retries, check.Equals, DefaultTokenMaxRetries)
	c.Assert(token.TokenType, check.Equals, "Bearer")
	c.Assert(token.isValid(), check.Equals, true)
}
