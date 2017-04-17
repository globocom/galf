package galf

import (
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

	clientOptions = NewClientOptions(
		DefaultClientTimeout,
		false,
		DefaultClientMaxRetries,
		"",
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
