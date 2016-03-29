package galf

import "errors"

var TokenExpiredError = errors.New("Token expired")

type HTTP struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *HTTP) Error() string {
	return e.Message
}

func NewHttpError(code int, message string) *HTTP {
	return &HTTP{Code: code, Message: message}
}
