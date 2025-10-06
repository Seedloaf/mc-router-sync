package auth

import (
	"net/http"
)

type NoneAuth struct{}

func (ta NoneAuth) AuthenticateRequest(req *http.Request) error {
	return nil
}

func NewNoneAuth() NoneAuth {
	return NoneAuth{}
}
