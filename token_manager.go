package galf

import (
	"encoding/base64"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/afex/hystrix-go/hystrix"

	"github.com/facebookgo/stackerr"
	"github.com/franela/goreq"
)

const (
	grantType = "grant_type=client_credentials"
)

type (
	TokenManager interface {
		GetToken() (*Token, error)
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
			resp, err = tm.do()

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

func (tm *OAuthTokenManager) do() (resp *goreq.Response, err error) {
	if tm.Options.HystrixConfig == nil {
		return tm.request()
	}

	if err = tm.Options.HystrixConfig.valid(); err != nil {
		return nil, err
	}
	return tm.requestHystrix()
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
		log.WithFields(log.Fields{
			"uri":           tm.TokenEndPoint,
			"statusCode":    resp.StatusCode,
			"authorization": tm.Authorization,
			"Body":          grantType,
		}).Error("Erro ao pegar um token do Backstage API")

		body, _ := resp.Body.ToString()
		return nil, NewHttpError(resp.StatusCode, body)
	}
	return resp, nil
}
