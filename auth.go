package mcroutersync

import (
	"errors"
	"net/http"
)

type AuthType string

const (
	AuthTypeApiKey AuthType = "apikey"
	AuthTypeNone   AuthType = "none"
)

var (
	ErrInvalidAuthType = errors.New("invalid auth type")
)

type Auth interface {
	AuthenticateRequest(req *http.Request) error
}

func GetAuthType(s string) (AuthType, error) {
	switch s {
	case string(AuthTypeApiKey):
		return AuthTypeApiKey, nil
	case string(AuthTypeNone):
		return AuthTypeNone, nil
	default:
		return AuthTypeNone, ErrInvalidAuthType
	}
}
