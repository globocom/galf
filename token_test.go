package galf

import (
	"strings"
	"time"

	"gopkg.in/check.v1"
)

type tokenSuite struct{}

var _ = check.Suite(&tokenSuite{})

func (s *tokenSuite) TestCreatesANewToken(c *check.C) {
	foo := strings.NewReader(
		`{"access_token": "qweasdzxc", "token_type": "neil peart", "expires_in": 15}`,
	)
	token, err := newToken(foo)

	c.Assert(err, check.IsNil)
	c.Assert(token.TokenType, check.Equals, "Neil Peart")
	c.Assert(time.Now().Add(time.Duration(15)*time.Second).After(token.expires_on), check.Equals, true)
	c.Assert(token.Authorization, check.Equals, "Neil Peart qweasdzxc")
}

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
