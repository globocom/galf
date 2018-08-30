package galf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/check.v1"
)

func TestAlf(t *testing.T) {
	check.TestingT(t)
}

func newTestServerCustom(handle func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	handlerWrapper := func(w http.ResponseWriter, r *http.Request) {
		println("Url: " + r.RequestURI)
		handle(w, r)
	}

	return httptest.NewServer(http.HandlerFunc(handlerWrapper))
}

func newTestServerToken(expireIn ...int) *httptest.Server {
	expire := 15
	if len(expireIn) > 0 {
		expire = expireIn[0]
	}

	ts := newTestServerCustom(handleToken(expire))

	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
	)
	SetDefaultTokenManager(tm)

	return ts
}

func handleToken(expire int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token := Token{
			AccessToken: "nonenoenoe",
			TokenType:   "bearer",
			ExpiresIn:   expire,
		}
		content, _ := json.Marshal(token)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(content)
	}
}
