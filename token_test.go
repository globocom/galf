package alf

import (
	"time"

	"gopkg.in/check.v1"
)

type tokenSuite struct{}

var _ = check.Suite(&tokenSuite{})

func (s *tokenSuite) TestTokenIsValidIfActualTimeIsLowerThanExpirationTime(c *check.C) {
	token := Token{
		expires_on: time.Date(2999, time.December, 30, 12, 0, 0, 0, time.UTC),
	}
	c.Assert(token.isValid(), check.Equals, true)
}

func (s *tokenSuite) TestTokenIsInvalidIfActualTimeIsBiggerThanExpirationTime(c *check.C) {
	token := Token{
		expires_on: time.Date(2000, time.December, 30, 12, 0, 0, 0, time.UTC),
	}
	c.Assert(token.isValid(), check.Equals, false)
}
