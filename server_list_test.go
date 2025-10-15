package mcrouterdiscovery

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAuth struct {
	shouldError bool
	headerKey   string
	headerValue string
}

func (m *mockAuth) AuthenticateRequest(req *http.Request) error {
	if m.shouldError {
		return errors.New("auth error")
	}
	if m.headerKey != "" {
		req.Header.Set(m.headerKey, m.headerValue)
	}
	return nil
}

func TestNewServerListClient(t *testing.T) {
	auth := &mockAuth{}
	client := NewServerListClient("http://api.example.com/servers", auth)

	if client == nil {
		t.Fatal("expected client to be non-nil")
	}
	if client.endpoint != "http://api.example.com/servers" {
		t.Errorf("expected endpoint to be http://api.example.com/servers, got %s", client.endpoint)
	}
	if client.client == nil {
		t.Error("expected http client to be non-nil")
	}
	if client.auth == nil {
		t.Error("expected auth to be non-nil")
	}
}

func TestGetServers(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse []Route
		serverStatus   int
		auth           Auth
		expectError    bool
		authHeaderKey  string
		authHeaderVal  string
	}{
		{
			name: "successful fetch with no auth",
			serverResponse: []Route{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
				{ServerAddress: "server2.example.com", Backend: "backend2:25565"},
			},
			serverStatus: http.StatusOK,
			auth:         &mockAuth{},
			expectError:  false,
		},
		{
			name: "successful fetch with api key auth",
			serverResponse: []Route{
				{ServerAddress: "server1.example.com", Backend: "backend1:25565"},
			},
			serverStatus: http.StatusOK,
			auth: &mockAuth{
				headerKey:   "X-API-Key",
				headerValue: "test-api-key",
			},
			expectError:   false,
			authHeaderKey: "X-API-Key",
			authHeaderVal: "test-api-key",
		},
		{
			name:           "empty server list",
			serverResponse: []Route{},
			serverStatus:   http.StatusOK,
			auth:           &mockAuth{},
			expectError:    false,
		},
		{
			name:           "server error",
			serverResponse: nil,
			serverStatus:   http.StatusInternalServerError,
			auth:           &mockAuth{},
			expectError:    true,
		},
		{
			name:           "authentication error",
			serverResponse: []Route{},
			serverStatus:   http.StatusOK,
			auth:           &mockAuth{shouldError: true},
			expectError:    true,
		},
		{
			name:           "unauthorized response",
			serverResponse: nil,
			serverStatus:   http.StatusUnauthorized,
			auth:           &mockAuth{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected method GET, got %s", r.Method)
				}

				if tt.authHeaderKey != "" {
					if r.Header.Get(tt.authHeaderKey) != tt.authHeaderVal {
						t.Errorf("expected auth header %s: %s, got %s", tt.authHeaderKey, tt.authHeaderVal, r.Header.Get(tt.authHeaderKey))
					}
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK && tt.serverResponse != nil {
					json.NewEncoder(w).Encode(tt.serverResponse)
				}
			}))
			defer server.Close()

			client := NewServerListClient(server.URL, tt.auth)
			routes, err := client.GetServers()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(routes) != len(tt.serverResponse) {
					t.Errorf("expected %d routes, got %d", len(tt.serverResponse), len(routes))
				}
				for i, route := range routes {
					if route.ServerAddress != tt.serverResponse[i].ServerAddress {
						t.Errorf("expected ServerAddress %s, got %s", tt.serverResponse[i].ServerAddress, route.ServerAddress)
					}
					if route.Backend != tt.serverResponse[i].Backend {
						t.Errorf("expected Backend %s, got %s", tt.serverResponse[i].Backend, route.Backend)
					}
				}
			}
		})
	}
}
