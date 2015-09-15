package alf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"gitlab.globoi.com/bastian/falkor/settings"
	. "gopkg.in/check.v1"
)

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

	err := settings.LoadSettings()
	c.Assert(err, IsNil)

	// tokenManager
	// Altera os valores do settings para teste
	settings.Backstage.Token.Url = fmt.Sprintf("%s/token", ts.URL)
	settings.Backstage.Token.ClientId = "foo"
	settings.Backstage.Token.ClientSecret = "bar"

	tokenOptions := NewTokenOptions(
		settings.Backstage.Token.Timeout,
		settings.Backstage.Token.Debug,
		DefaultTokenMaxRetries,
		CircuitConfig{Name: "circuit_backstage_gateway_token"},
	)

	tm := NewTokenManager(
		settings.Backstage.Token.Url,
		settings.Backstage.Token.ClientId,
		settings.Backstage.Token.ClientSecret,
		tokenOptions,
	)

	SetDefaultTokenManager(tm)

	client := NewClient()

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
	settings.LoadSettings()
}
