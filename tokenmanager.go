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

const (
	grantType                 = "grant_type=client_credentials"
	DefaultTokenMaxRetries    = 2
	DefaultTokenClientTimeout = 1 * time.Second
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
	MaxRetries    int
	Backoff       BackoffStrategy
}

func NewTokenManager(tokenEndPoint, clientId, clientSecret string, backOff BackoffStrategy,
	maxRetries int, debug bool, timeout ...time.Duration) *OAuthTokenManager {
	timeOut := DefaultTokenClientTimeout
	if len(timeout) > 0 {
		timeOut = timeout[0]
	}

	authorizationString := []byte(clientId + ":" + clientSecret)

	tm := &OAuthTokenManager{
		TokenEndPoint: tokenEndPoint,
		ClientId:      clientId,
		ClientSecret:  clientSecret,
		Authorization: "Basic " + base64.StdEncoding.EncodeToString(authorizationString),
		Debug:         debug,
		Timeout:       timeOut,
		MaxRetries:    maxRetries,
		Backoff:       backOff,
	}

	return tm
}

func (tm *OAuthTokenManager) GetToken() (*Token, error) {

	if tm.token == nil || !tm.token.isValid() {
		var err error
		var resp *goreq.Response

		for i := 0; i < tm.MaxRetries; i++ {
			resp, err = requestToken(tm)

			if err != nil {
				if i < tm.MaxRetries-1 {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				return nil, err
			}

			defer resp.Body.Close()
			if tm.token, err = newToken(resp.Body); err != nil {
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
			Body:        grantType,
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
				"Body":          grantType,
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
