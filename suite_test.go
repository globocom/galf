package galf

import (
	"fmt"
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
		fmt.Fprint(w, fmt.Sprintf(`{"access_token": "nonenoenoe", "token_type": "bearer", "expires_in": %d}`, expire))
	}
}
