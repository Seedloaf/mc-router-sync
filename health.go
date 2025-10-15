package mcrouterdiscovery

import (
	"context"
	"log"
	"log/slog"
	"net/http"
)

func StartHealthServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down health server...")
		if err := server.Shutdown(context.Background()); err != nil {
			slog.Info("Health server shutdown error: %s", "err", err)
		}
	}()

	slog.Info("Starting health server on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Health server failed: %s", err)
	}
}
