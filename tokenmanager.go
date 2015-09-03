package alf

import (
	"encoding/base64"
	"time"

	goerrors "errors"

	"gitlab.globoi.com/bastian/falkor/errors"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

const GrantType = "grant_type=client_credentials"

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

func SetDefaultTokenManager(t TokenManager) {
	defaultTokenManager = t
}

func (tm *OAuthTokenManager) GetToken() (*Token, error) {

	if tm.token == nil || !tm.token.isValid() {
		var err error
		var resp *goreq.Response

		maxRetries := 2
		for i := 0; i < maxRetries; i++ {
			if resp, err = requestToken(tm); err != nil {
				return nil, stackerr.Wrap(err)
			}

			// 200 and 300 level errors are considered success and we are done
			if resp.StatusCode < 400 {
				defer resp.Body.Close()
				tm.token, err = newToken(resp.Body)
				if err != nil {
					return nil, err
				}
				return tm.token, nil
			}

			// wait 50ms
			time.Sleep(50 * time.Millisecond)
		}
		return nil, errors.NewHttpError(resp.StatusCode, "Não foi possível recupera o token")
	}

	return tm.token, nil
}

func requestToken(tm *OAuthTokenManager) (*goreq.Response, error) {
	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go("circuit_backstage_api_token", func() error {
		resp, err := goreq.Request{
			Method:      "POST",
			ContentType: "application/x-www-form-urlencoded",
			Uri:         tm.TokenEndPoint,
			Body:        GrantType,
			ShowDebug:   tm.Debug,
			Timeout:     tm.Timeout,
		}.WithHeader("Authorization", tm.Authorization).Do()

		if err != nil || resp.StatusCode >= 400 {
			return goerrors.New("Erro ao pegar um token do Backstage API.")
		}

		output <- resp

		return nil
	}, nil)

	select {
	case out := <-output:
		return out, nil
	case err := <-errors:
		return nil, err
	}
}
