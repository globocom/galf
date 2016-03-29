package galf

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"gopkg.in/check.v1"
)

type clientSuite struct{}

var _ = check.Suite(&clientSuite{})

func (s *clientSuite) TestAlfClient(c *check.C) {
	tst := newTestServerToken()
	defer tst.Close()

	client := NewClient()

	ts := newTestServer("tokenClient")
	defer ts.Close()

	// Testa métodos do Client
	urlStr := fmt.Sprintf("%s/feed/1", ts.URL)
	body := strings.NewReader(
		`{"feedIdentifier": "http://feed.globo.com", "product": "GSHOW", "type": "materia", "content":{"title":"abc"}}`)

	// GET
	resp, err := client.Get(urlStr)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	// POST
	resp, err = client.Post(urlStr, body)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusCreated)

	// PUT
	resp, err = client.Put(urlStr, body)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusNoContent)

	// DELETE
	resp, err = client.Delete(urlStr)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusNoContent)

}

func newTestServer(name string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			println(name + " - " + r.RequestURI)
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

	return ts
}
