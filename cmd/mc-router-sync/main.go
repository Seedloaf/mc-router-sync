package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	mcroutersync "github.com/Seedloaf/mc-router-discovery"
	"github.com/Seedloaf/mc-router-discovery/auth"
)

func main() {
	cfg, err := mcroutersync.LoadConfigFromFlags()
	if err != nil {
		log.Fatalf("Invalid configuration: %s", err)
	}

	configureLogger(cfg.LogLevel)

	var authimpl mcroutersync.Auth
	switch cfg.AuthType {
	case mcroutersync.AuthTypeApiKey:
		authimpl = auth.NewApiKeyAuth(cfg.AuthToken)
	default:
		authimpl = auth.NewNoneAuth()
	}

	sl := mcroutersync.NewServerListClient(cfg.ServerListAPI, authimpl)
	mr := mcroutersync.NewMcRouterClient(cfg.McRouterHost)
	reconciler := mcroutersync.NewReconciler(sl, mr, cfg.SyncInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go mcroutersync.StartHealthServer(ctx)
	reconciler.Start(ctx)
}

func configureLogger(l slog.Level) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	}))
	slog.SetDefault(logger)
}
