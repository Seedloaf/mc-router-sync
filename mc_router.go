package mcrouterdiscovery

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type McRouterClient struct {
	host   string
	client *http.Client
}

func NewMcRouterClient(host string) *McRouterClient {
	return &McRouterClient{
		host: host,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *McRouterClient) GetRoutes() (Routes, error) {
	req, err := http.NewRequest(http.MethodGet, c.host+"/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var routes Routes
	if err := routes.Parse(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return routes, nil
}

func (c *McRouterClient) RegisterRoute(route Route) error {
	r, err := route.Json()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.host+"/routes", r)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *McRouterClient) DeleteRoute(serverAddress string) error {
	req, err := http.NewRequest(http.MethodDelete, c.host+"/routes/"+serverAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
