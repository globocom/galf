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

type (
	TokenManager interface {
		GetToken() (*Token, error)
	}

	TokenOptions struct {
		Timeout       time.Duration
		Backoff       BackoffStrategy
		MaxRetries    int
		ShowDebug     bool
		CircuitConfig *CircuitConfig
	}

	OAuthTokenManager struct {
		TokenEndPoint string
		ClientId      string
		ClientSecret  string
		Authorization string
		Options       TokenOptions
		token         *Token
	}
)

var (
	defaultTokenOptions = TokenOptions{
		Timeout:    DefaultClientTimeout,
		MaxRetries: DefaultClientMaxRetries,
		Backoff:    ConstantBackOff,
		ShowDebug:  false,
	}

	defaultTokenManager TokenManager
)

func SetDefaultTokenManager(tokenManager TokenManager) {
	defaultTokenManager = tokenManager
}

func NewTokenOptions(timeout time.Duration, debug bool, maxRetries int, circuitConfig CircuitConfig, backoff ...BackoffStrategy) TokenOptions {
	tokenBackoff := ConstantBackOff
	if len(backoff) > 0 {
		tokenBackoff = backoff[0]
	}

	return TokenOptions{
		Timeout:       timeout,
		ShowDebug:     debug,
		MaxRetries:    maxRetries,
		Backoff:       tokenBackoff,
		CircuitConfig: &circuitConfig,
	}
}

func NewTokenManager(tokenEndPoint, clientId, clientSecret string, options ...TokenOptions) *OAuthTokenManager {
	tokenOptions := defaultTokenOptions
	if len(options) > 0 {
		tokenOptions = options[0]
	}

	authorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientId+":"+clientSecret))
	tm := &OAuthTokenManager{
		TokenEndPoint: tokenEndPoint,
		ClientId:      clientId,
		ClientSecret:  clientSecret,
		Authorization: authorization,
		Options:       tokenOptions,
	}

	return tm
}

func (tm *OAuthTokenManager) GetToken() (*Token, error) {

	if tm.token == nil || !tm.token.isValid() {
		var err error
		var resp *goreq.Response

		for i := 0; i < tm.Options.MaxRetries; i++ {
			resp, err = requestToken(tm)

			if err != nil {
				if i < tm.Options.MaxRetries-1 {
					time.Sleep(tm.Options.Backoff(i))
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
			ShowDebug:   tm.Options.ShowDebug,
			Timeout:     tm.Options.Timeout,
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
