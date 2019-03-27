/*
* Go OAuth2 Client
*
* MIT License
*
* Copyright (c) 2015 Globo.com
 */

package galf

import (
	"strings"
	"time"

	"gopkg.in/check.v1"
)

type tokenSuite struct{}

var _ = check.Suite(&tokenSuite{})

func (s *tokenSuite) TestCreateNewToken(c *check.C) {
	bodyToken := strings.NewReader(
		`{"access_token": "nonenone", "token_type": "bearer", "expires_in": 15}`,
	)
	token, err := newToken(bodyToken)

	c.Assert(err, check.IsNil)
	c.Assert(token.TokenType, check.Equals, "Bearer")
	c.Assert(token.Authorization, check.Equals, "Bearer nonenone")
}

func (s *tokenSuite) TestTokenIsValid(c *check.C) {
	bodyToken := strings.NewReader(
		`{"access_token": "nonenone", "token_type": "bearer", "expires_in": 1}`,
	)
	token, err := newToken(bodyToken)

	c.Assert(err, check.IsNil)
	c.Assert(token.isValid(), check.Equals, true)
}

func (s *tokenSuite) TestTokenNotIsValid(c *check.C) {
	bodyToken := strings.NewReader(
		`{"access_token": "nonenone", "token_type": "bearer", "expires_in": 1}`,
	)
	token, err := newToken(bodyToken)
	time.Sleep(1 * time.Second)

	c.Assert(err, check.IsNil)
	c.Assert(token.isValid(), check.Equals, false)
}
