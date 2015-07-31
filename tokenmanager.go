package alf

import (
	"encoding/base64"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

const (
	GrantType = "grant_type=client_credentials"
)

type TokenManager interface {
	GetToken() (*Token, error)
}

type OAuthTokenManager struct {
	TokenEndPoint string
	ClientId      string
	ClientSecret  string
	Authorization string
	Debug         bool
	Timeout       time.Duration
	token         *Token
}

var defaultTokenManager TokenManager

func NewTokenManager(tokenEndPoint, clientId, clientSecret string, debug bool, timeout time.Duration) *OAuthTokenManager {
	authorizationString := []byte(clientId + ":" + clientSecret)

	tm := &OAuthTokenManager{
		TokenEndPoint: tokenEndPoint,
		ClientId:      clientId,
		ClientSecret:  clientSecret,
		Authorization: "Basic " + base64.StdEncoding.EncodeToString(authorizationString),
		Debug:         debug,
		Timeout:       timeout,
	}

	return tm
}

func (tm *OAuthTokenManager) GetToken() (*Token, error) {

	if tm.token == nil || !tm.token.isValid() {
		resp, err := goreq.Request{
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			Uri:         tm.TokenEndPoint,
			Body:        GrantType,
			ShowDebug:   tm.Debug,
			Timeout:     tm.Timeout,
		}.WithHeader("Authorization", tm.Authorization).Do()

		if err != nil {
			return nil, stackerr.Wrap(err)
		}

		defer resp.Body.Close()
		tm.token, err = newToken(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	return tm.token, nil
}

func SetDefaultTokenManager(t TokenManager) {
	defaultTokenManager = t
}
