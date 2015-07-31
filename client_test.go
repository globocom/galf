package alf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.globoi.com/bastian/falkor/settings"
	. "gopkg.in/check.v1"
)

func TestClient(t *testing.T) {
	TestingT(t)
}

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) TestAlfClient(c *C) {
	// testServer
	ts := *httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "GET":
					w.WriteHeader(200)
				case "POST":
					w.WriteHeader(201)
				case "PUT":
					w.WriteHeader(204)
				case "DELETE":
					w.WriteHeader(204)
				}
				fmt.Fprintf(w, "{}")
			}))
	defer ts.Close()

	// tokenManager
	// Altera os valores do settings para teste
	settings.Backtage.Token.Url = fmt.Sprintf("%s/token", ts.URL)
	settings.Backtage.Token.ClientId = "foo"
	settings.Backtage.Token.ClientSecret = "bar"

	tm := NewTokenManager(settings.Backtage.Token.Url,
		settings.Backtage.Token.ClientId,
		settings.Backtage.Token.ClientSecret,
		settings.Backtage.Token.Debug,
		settings.Backtage.Token.Timeout)

	SetDefaultTokenManager(tm)

	// NewClient retorna um http.Client
	client := NewClient()
	c.Assert(client.httpClient, FitsTypeOf, &http.Client{})

	// Testa métodos do Client
	urlStr := fmt.Sprintf("%s/feed/1", ts.URL)
	body := strings.NewReader(
		`{"feedIdentifier": "http://feed.globo.com", "product": "GSHOW", "type": "materia", "content":{"title":"abc"}}`)

	// GET
	resp, err := client.Get(urlStr)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	// POST
	resp, err = client.Post(urlStr, body)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusCreated)

	// PUT
	resp, err = client.Put(urlStr, body)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	// DELETE
	resp, err = client.Delete(urlStr)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)

	// Retorna os valores padrões
	settings.LoadConfig()
}
