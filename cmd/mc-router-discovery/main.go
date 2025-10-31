package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	mcrouterdiscovery "github.com/Seedloaf/mc-router-discovery"
	"github.com/Seedloaf/mc-router-discovery/auth"
)

func main() {
	cfg, err := mcrouterdiscovery.LoadConfigFromFlags()
	if err != nil {
		log.Fatalf("Invalid configuration: %s", err)
	}

	configureLogger(cfg.LogLevel)

	var authimpl mcrouterdiscovery.Auth
	switch cfg.AuthType {
	case mcrouterdiscovery.AuthTypeApiKey:
		authimpl = auth.NewApiKeyAuth(cfg.AuthToken)
	default:
		authimpl = auth.NewNoneAuth()
	}

	sl := mcrouterdiscovery.NewServerListClient(cfg.ServerListAPI, authimpl)
	mr := mcrouterdiscovery.NewMcRouterClient(cfg.McRouterHost, mcrouterdiscovery.McRouterClientOpts{Auth: authimpl})
	reconciler := mcrouterdiscovery.NewReconciler(sl, mr, cfg.SyncInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go mcrouterdiscovery.StartHealthServer(ctx)
	reconciler.Start(ctx)
}

func configureLogger(l slog.Level) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	}))
	slog.SetDefault(logger)
}
