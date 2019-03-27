/*
* Go OAuth2 Client
*
* MIT License
*
* Copyright (c) 2015 Globo.com
 */

package galf

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/facebookgo/stackerr"
	"github.com/globocom/goreq"
)

const (
	grantType = "grant_type=client_credentials"
)

type (
	TokenManager interface {
		GetToken() (*Token, error)
		ResetToken()
	}

	OAuthTokenManager struct {
		TokenEndPoint string
		ClientId      string
		ClientSecret  string
		Authorization string
		Options       TokenOptions
		token         *Token
		mutex         *sync.Mutex
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
		mutex:         &sync.Mutex{},
	}

	return tm
}

func (tm *OAuthTokenManager) GetToken() (*Token, error) {
	var err error

	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	if tm.isValid() {
		return tm.token, nil
	}

	for i := 1; i <= tm.Options.MaxRetries; i++ {

		if tm.isValid() {
			return tm.token, nil
		}

		tm.token, err = tm.do()

		if err != nil && i < tm.Options.MaxRetries {
			time.Sleep(tm.Options.Backoff(i))
			continue
		}

		return tm.token, err
	}

	return tm.token, err
}

func (tm *OAuthTokenManager) ResetToken() {
	tm.token = nil
}

func (tm *OAuthTokenManager) isValid() bool {
	return tm.token != nil && tm.token.isValid()
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

	defer resp.Body.Close() // nolint: errcheck
	if token, err = newToken(resp.Body); err != nil {
		return nil, err
	}

	return token, nil
}

func (tm *OAuthTokenManager) requestHystrix() (*goreq.Response, error) {

	output := make(chan *goreq.Response, 1)
	errors := hystrix.Go(tm.Options.HystrixConfig.Name, func() error {

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

	client := goreq.NewClient(goreq.Options{
		Timeout: tm.Options.Timeout,
	})

	req := goreq.Request{
		Method:      "POST",
		ContentType: "application/x-www-form-urlencoded",
		Uri:         tm.TokenEndPoint,
		Body:        grantType,
		ShowDebug:   tm.Options.ShowDebug,
	}
	req.AddHeader("Authorization", tm.Authorization)

	resp, err := client.Do(req)

	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	if resp.StatusCode >= 300 {
		var body string
		if body, err = resp.Body.ToString(); err != nil {
			return nil, stackerr.Wrap(err)
		}
		resp.Body.Close() // nolint:errcheck

		erroMsg := fmt.Sprintf("Failed to request token url: %s - statusCode: %d - body: %s", resp.Request.URL, resp.StatusCode, body)
		return nil, NewHttpError(resp.StatusCode, erroMsg)
	}
	return resp, nil
}
