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
	return httptest.NewServer(http.HandlerFunc(handle))
}

func newTestServerToken() *httptest.Server {
	handleGetToken := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"access_token": "nonenoenoe", "token_type": "bearer", "expires_in": 15}`)
	}

	ts := newTestServerCustom(handleGetToken)
	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
	)
	SetDefaultTokenManager(tm)

	return ts
}
