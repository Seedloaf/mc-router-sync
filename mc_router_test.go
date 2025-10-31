package mcrouterdiscovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewMcRouterClient(t *testing.T) {
	client := NewMcRouterClient("http://localhost:8080", McRouterClientOpts{})
	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.host != "http://localhost:8080" {
		t.Errorf("expected host to be http://localhost:8080, got %s", client.host)
	}
	if client.client == nil {
		t.Error("expected http client to be non-nil")
	}
}

func TestGetRoutes(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse *GetResponse
		serverStatus   int
		expectError    bool
	}{
		{
			name: "successful get routes",
			serverResponse: &GetResponse{
				"server1.example.com": "backend1:25565",
				"server2.example.com": "backend2:25565",
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name:           "empty routes",
			serverResponse: &GetResponse{},
			serverStatus:   http.StatusOK,
			expectError:    false,
		},
		{
			name:           "server error",
			serverResponse: nil,
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/routes" {
					t.Errorf("expected path /routes, got %s", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Errorf("expected method GET, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK && tt.serverResponse != nil {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewMcRouterClient(server.URL, McRouterClientOpts{})
			routes, err := client.GetRoutes()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(routes) != len(*tt.serverResponse) {
					t.Errorf("expected %d routes, got %d", len(*tt.serverResponse), len(routes))
				}
				for _, route := range routes {
					expectedBackend, exists := (*tt.serverResponse)[route.ServerAddress]
					if !exists {
						t.Errorf("unexpected route with ServerAddress %s", route.ServerAddress)
					}
					if route.Backend != expectedBackend {
						t.Errorf("expected Backend %s for ServerAddress %s, got %s", expectedBackend, route.ServerAddress, route.Backend)
					}
				}
			}
		})
	}
}

func TestRegisterRoute(t *testing.T) {
	tests := []struct {
		name         string
		route        Route
		serverStatus int
		expectError  bool
	}{
		{
			name: "successful registration",
			route: Route{
				ServerAddress: "server1.example.com",
				Backend:       "backend1:25565",
			},
			serverStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "successful registration with 201",
			route: Route{
				ServerAddress: "server2.example.com",
				Backend:       "backend2:25565",
			},
			serverStatus: http.StatusCreated,
			expectError:  false,
		},
		{
			name: "server error",
			route: Route{
				ServerAddress: "server3.example.com",
				Backend:       "backend3:25565",
			},
			serverStatus: http.StatusInternalServerError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRoute Route
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/routes" {
					t.Errorf("expected path /routes, got %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Errorf("expected method POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				if err := json.NewDecoder(r.Body).Decode(&receivedRoute); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := NewMcRouterClient(server.URL, McRouterClientOpts{})
			err := client.RegisterRoute(tt.route)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if receivedRoute.ServerAddress != tt.route.ServerAddress {
					t.Errorf("expected ServerAddress %s, got %s", tt.route.ServerAddress, receivedRoute.ServerAddress)
				}
				if receivedRoute.Backend != tt.route.Backend {
					t.Errorf("expected Backend %s, got %s", tt.route.Backend, receivedRoute.Backend)
				}
			}
		})
	}
}

func TestDeleteRoute(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		serverStatus  int
		expectError   bool
	}{
		{
			name:          "successful deletion",
			serverAddress: "server1.example.com",
			serverStatus:  http.StatusOK,
			expectError:   false,
		},
		{
			name:          "successful deletion with 204",
			serverAddress: "server2.example.com",
			serverStatus:  http.StatusNoContent,
			expectError:   false,
		},
		{
			name:          "not found error",
			serverAddress: "nonexistent.example.com",
			serverStatus:  http.StatusNotFound,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/routes/" + tt.serverAddress
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("expected method DELETE, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := NewMcRouterClient(server.URL, McRouterClientOpts{})
			err := client.DeleteRoute(tt.serverAddress)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
