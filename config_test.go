package mcrouterdiscovery

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestLoadConfigFromFlags(t *testing.T) {
	oldArgs := os.Args
	oldEnv := os.Getenv("API_KEY")
	defer func() {
		os.Args = oldArgs
		if oldEnv != "" {
			os.Setenv("API_KEY", oldEnv)
		} else {
			os.Unsetenv("API_KEY")
		}
	}()

	tests := []struct {
		name        string
		args        []string
		envAPIKey   string
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *ParsedConfig)
	}{
		{
			name:        "missing mc-router-host",
			args:        []string{"cmd", "-server-list-api=http://api.example.com"},
			expectError: true,
			errorMsg:    "McRouterHost is invalid: required",
		},
		{
			name:        "missing server-list-api",
			args:        []string{"cmd", "-mc-router-host=http://localhost:8080"},
			expectError: true,
			errorMsg:    "ServerListAPI is invalid: required",
		},
		{
			name: "valid config with no auth",
			args: []string{"cmd", "-mc-router-host=http://localhost:8080", "-server-list-api=http://api.example.com"},
			validate: func(t *testing.T, c *ParsedConfig) {
				if c.McRouterHost != "http://localhost:8080" {
					t.Errorf("expected McRouterHost to be http://localhost:8080, got %s", c.McRouterHost)
				}
				if c.ServerListAPI != "http://api.example.com" {
					t.Errorf("expected ServerListAPI to be http://api.example.com, got %s", c.ServerListAPI)
				}
				if c.AuthType != AuthTypeNone {
					t.Errorf("expected AuthType to be none, got %s", c.AuthType)
				}
				if c.SyncInterval != 30*time.Second {
					t.Errorf("expected SyncInterval to be 30s, got %s", c.SyncInterval)
				}
			},
		},
		{
			name:        "apikey auth without token",
			args:        []string{"cmd", "-mc-router-host=http://localhost:8080", "-server-list-api=http://api.example.com", "-auth-type=apikey"},
			expectError: true,
			errorMsg:    "auth-token is required when auth-type is apikey",
		},
		{
			name:      "valid apikey auth",
			args:      []string{"cmd", "-mc-router-host=http://localhost:8080", "-server-list-api=http://api.example.com", "-auth-type=apikey"},
			envAPIKey: "secret123",
			validate: func(t *testing.T, c *ParsedConfig) {
				if c.AuthType != AuthTypeApiKey {
					t.Errorf("expected AuthType to be apikey, got %s", c.AuthType)
				}
				if c.AuthToken != "secret123" {
					t.Errorf("expected AuthToken to be secret123, got %s", c.AuthToken)
				}
			},
		},
		{
			name:        "invalid auth type",
			args:        []string{"cmd", "-mc-router-host=http://localhost:8080", "-server-list-api=http://api.example.com", "-auth-type=invalid"},
			expectError: true,
			errorMsg:    "invalid auth-type: invalid (must be apikey or none)",
		},
		{
			name: "custom sync interval",
			args: []string{"cmd", "-mc-router-host=http://localhost:8080", "-server-list-api=http://api.example.com", "-sync-interval=60"},
			validate: func(t *testing.T, c *ParsedConfig) {
				if c.SyncInterval != 60*time.Second {
					t.Errorf("expected SyncInterval to be 60s, got %s", c.SyncInterval)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			if tt.envAPIKey != "" {
				os.Setenv("API_KEY", tt.envAPIKey)
			} else {
				os.Unsetenv("API_KEY")
			}

			os.Args = tt.args

			config, err := LoadConfigFromFlags()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}
