package galf

import (
	"bytes"
	"io"
	"net/http"
)

// Response is a wrapper around http.Response for compatibility with existing code
type Response struct {
	StatusCode int
	Header     http.Header
	Body       Body
	raw        *http.Response
}

// Body represents the response body
type Body struct {
	content io.ReadCloser
}

// ToString reads the body content and returns it as a string
func (b *Body) ToString() (string, error) {
	if b.content == nil {
		return "", nil
	}

	// Use a more efficient approach
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, b.content)
	if err != nil {
		return "", err
	}

	// Close the original body
	b.content.Close()

	return buf.String(), nil
}

// Close closes the body
func (b *Body) Close() error {
	if b.content != nil {
		return b.content.Close()
	}
	return nil
}
