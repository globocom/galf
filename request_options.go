/*
* Go OAuth2 Client
*
* MIT License
*
* Copyright (c) 2015 Globo.com
 */

package galf

type requestOptions struct {
	headers []headerTuple
}

type headerTuple struct {
	name  string
	value string
}

func NewRequestOptions() *requestOptions {
	ro := &requestOptions{
		headers: []headerTuple{},
	}
	return ro
}

func (ro *requestOptions) AddHeader(name string, value string) {
	ro.headers = append(ro.headers, headerTuple{name: name, value: value})
}

func (ro *requestOptions) AddHeaders(headers map[string]string) {
	for name, value := range headers {
		ro.AddHeader(name, value)
	}
}
