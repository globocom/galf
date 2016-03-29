package galf

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/facebookgo/stackerr"
)

type Token struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	ExpiresIn     int    `json:"expires_in"`
	Authorization string
	expiresOn     time.Time
}

func (t *Token) isValid() bool {
	return time.Now().Before(t.expiresOn)
}

func newToken(body io.Reader) (*Token, error) {
	var token Token
	err := json.NewDecoder(body).Decode(&token)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	token.TokenType = strings.Title(token.TokenType)
	token.expiresOn = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	token.Authorization = fmt.Sprintf("%s %s", token.TokenType, token.AccessToken)
	return &token, nil
}
