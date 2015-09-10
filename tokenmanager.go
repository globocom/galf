package alf

import (
	"encoding/base64"
	"time"

	log "github.com/Sirupsen/logrus"
	"gitlab.globoi.com/bastian/falkor/errors"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

const GrantType = "grant_type=client_credentials"

var (
	defaultTokenMaxRetries = 2
	defaultTokenManager    TokenManager
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

func SetDefaultTokenMaxRetries(retries int) {
	defaultTokenMaxRetries = retries
}

func SetDefaultTokenManager(t TokenManager) {
	defaultTokenManager = t
}

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
		var err error
		var resp *goreq.Response

		for i := 0; i < defaultTokenMaxRetries; i++ {
			resp, err = requestToken(tm)

			if i < c.MaxRetries-1 {
				// wait
				time.Sleep(c.Backoff(i))
			}

			if err != nil {
				if _, ok := err.(*errors.HTTP); ok {
					// wait 50ms
					time.Sleep(50 * time.Millisecond)
				}
			}

			defer resp.Body.Close()
			tm.token, err = newToken(resp.Body)
			if err != nil {
				return nil, err
			}
			return tm.token, nil
		}

		return nil, err
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

		if err != nil {
			return stackerr.Wrap(err)
		}

		if resp.StatusCode >= 300 {
			log.WithFields(log.Fields{
				"uri":           tm.TokenEndPoint,
				"statusCode":    resp.StatusCode,
				"authorization": tm.Authorization,
			}).Error("Erro ao pegar um token do Backstage API")

			body, _ := resp.Body.ToString()
			return errors.NewHttpError(resp.StatusCode, body)
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
