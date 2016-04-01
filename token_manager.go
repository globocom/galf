package galf

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/afex/hystrix-go/hystrix"

	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

const (
	grantType = "grant_type=client_credentials"
)

type (
	TokenManager interface {
		GetToken() (Token, error)
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
	defaultTokenManager TokenManager
)

func SetDefaultTokenManager(tokenManager TokenManager) {
	defaultTokenManager = tokenManager
}

func NewTokenManager(tokenEndPoint string, clientId string, clientSecret string, options ...TokenOptions) *OAuthTokenManager {
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

func (tm *OAuthTokenManager) GetToken() (Token, error) {

	if tm.token == nil || !tm.token.isValid() {
		for i := 1; i <= tm.Options.MaxRetries; i++ {
			token, err := tm.do()

			if err == nil && !token.isValid() {
				err = TokenExpiredError
			}

			if err != nil {
				if i < tm.Options.MaxRetries {
					time.Sleep(tm.Options.Backoff(i))
					continue
				}
				return Token{}, err
			}

			tm.token = token
			return *tm.token, nil
		}
	}

	return *tm.token, nil
}

func (tm *OAuthTokenManager) do() (token *Token, err error) {
	var resp *goreq.Response
	if tm.Options.HystrixConfig == nil {
		if resp, err = tm.request(); err != nil {
			return nil, err
		}
	} else {
		if err = tm.Options.HystrixConfig.valid(); err != nil {
			return nil, err
		}
		if resp, err = tm.requestHystrix(); err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()
	if token, err = newToken(resp.Body); err != nil {
		return nil, err
	}

	return token, nil
}

func (tm *OAuthTokenManager) requestHystrix() (*goreq.Response, error) {

	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go(tm.Options.HystrixConfig.configName, func() error {

		resp, err := tm.request()
		if err != nil {
			return err
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

func (tm *OAuthTokenManager) request() (*goreq.Response, error) {

	resp, err := goreq.Request{
		Method:      "POST",
		ContentType: "application/x-www-form-urlencoded",
		Uri:         tm.TokenEndPoint,
		Body:        grantType,
		ShowDebug:   tm.Options.ShowDebug,
		Timeout:     tm.Options.Timeout,
	}.WithHeader("Authorization", tm.Authorization).Do()

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	if resp.StatusCode >= 300 {
		var body string
		if body, err = resp.Body.ToString(); err != nil {
			return nil, stackerr.Wrap(err)
		}
		resp.Body.Close()

		erroMsg := fmt.Sprintf("Failed to request token url: %s - statusCode: %d - body: %s", resp.Request.URL, resp.StatusCode, body)
		return nil, NewHttpError(resp.StatusCode, erroMsg)
	}
	return resp, nil
}
