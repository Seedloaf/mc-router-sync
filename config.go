package mcrouterdiscovery

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-playground/validator"
)

var (
	ErrMissingRequired = errors.New("missing required argument")
)

type Config struct {
	McRouterHost  string `validate:"required"`
	ServerListAPI string `validate:"required"`
	AuthType      string // "apikey", "none"
	AuthToken     string // Bearer token or API key value
	LogLevel      string
	SyncInterval  int // Sync interval in seconds
}

type ParsedConfig struct {
	McRouterHost  string
	ServerListAPI string
	AuthType      AuthType
	AuthToken     string
	LogLevel      slog.Level
	SyncInterval  time.Duration
}

func LoadConfigFromFlags() (*ParsedConfig, error) {
	v := validator.New()

	config := &Config{}

	flag.StringVar(&config.McRouterHost, "mc-router-host", "", "* McRouter API host (e.g. http://localhost:8000)")
	flag.StringVar(&config.ServerListAPI, "server-list-api", "", "* Server list API endpoint (e.g. http://localhost:3000/api/servers)")
	flag.StringVar(&config.AuthType, "auth-type", "none", "Authentication type for the server list API: apikey, none")
	flag.StringVar(&config.LogLevel, "log-level", "info", "The lowest level log you would like (e.g. debug)")
	flag.IntVar(&config.SyncInterval, "sync-interval", 30, "Sync interval in seconds")

	flag.Parse()

	config.AuthToken = resolveApiKeySecrets()

	var validateErrs validator.ValidationErrors
	err := v.Struct(config)
	if err != nil {
		if errors.As(err, &validateErrs) {
			for _, err := range validateErrs {
				return nil, fmt.Errorf("%s is invalid: %s", err.Field(), err.Tag())
			}
		}

		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	authType, err := GetAuthType(config.AuthType)
	if err != nil {
		return nil, fmt.Errorf("invalid auth-type: %s (must be apikey or none)", config.AuthType)
	}

	if authType == AuthTypeApiKey && config.AuthToken == "" {
		return nil, fmt.Errorf("auth-token is required when auth-type is %s", config.AuthType)
	}

	return &ParsedConfig{
		McRouterHost:  config.McRouterHost,
		ServerListAPI: config.ServerListAPI,
		AuthType:      authType,
		AuthToken:     config.AuthToken,
		LogLevel:      resolveLogLevel(config.LogLevel),
		SyncInterval:  time.Duration(config.SyncInterval) * time.Second,
	}, nil
}

func resolveLogLevel(l string) slog.Level {
	switch l {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func resolveApiKeySecrets() string {
	return os.Getenv("API_KEY")
}
