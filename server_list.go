package mcroutersync

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type ServerListClient struct {
	endpoint string
	client   *http.Client
	auth     Auth
}

func NewServerListClient(endpoint string, auth Auth) *ServerListClient {
	return &ServerListClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		auth: auth,
	}
}

func (c *ServerListClient) GetServers() (Routes, error) {
	req, err := http.NewRequest(http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.auth.AuthenticateRequest(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var routes Routes
	if err := routes.Parse(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to parse server list: %w", err)
	}

	return routes, nil
}
