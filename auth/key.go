package auth

import (
	"fmt"
	"net/http"
)

type ApiKeyAuth struct {
	token string
}

func (ta ApiKeyAuth) AuthenticateRequest(req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ta.token))

	return nil
}

func NewApiKeyAuth(token string) ApiKeyAuth {
	return ApiKeyAuth{token}
}
