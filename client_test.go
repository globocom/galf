package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

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

	tm := NewTokenManager(
		ts.URL+"/token",
		"ClientId",
		"ClientSecret",
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

}
