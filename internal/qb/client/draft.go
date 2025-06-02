package client

import (
	"fmt"
	"net/http"
	"net/url"
)

type Request struct {
	Method     string
	Path       string
	Params     url.Values
	Payload    []byte
	AuthCookie *http.Cookie
}

func NewRequest(method string, path string, params url.Values, payload []byte, authCookie *http.Cookie) *Request {
	return &Request{
		Method:     method,
		Path:       path,
		Params:     params,
		Payload:    payload,
		AuthCookie: authCookie,
	}
}

func (req *Request) String() string {
	return fmt.Sprintf("%s %s", req.Method, req.Path)
}
