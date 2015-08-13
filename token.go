package alf

import (
	"encoding/json"
	"io"
	"strings"
	"time"
)

type Token struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type"`
	ExpiresIn     int    `json:"expires_in"`
	expires_on    time.Time
	Authorization string
}

func (t *Token) isValid() bool {
	return time.Now().Before(t.expires_on)
}

func newToken(body io.Reader) (*Token, error) {
	var token Token
	err := json.NewDecoder(body).Decode(&token)
	if err != nil {
		return nil, err
	}

	token.TokenType = strings.Title(token.TokenType)
	token.expires_on = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	token.Authorization = token.TokenType + " " + token.AccessToken
	return &token, nil
}
