package main

import (
	"log"

	mcroutersync "github.com/Seedloaf/mc-router-sync"
	"github.com/Seedloaf/mc-router-sync/auth"
)

func main() {
	cfg, err := mcroutersync.LoadConfigFromFlags()
	if err != nil {
		log.Fatalf("Invalid configuration: %s", err)
	}

	var authimpl mcroutersync.Auth
	switch cfg.AuthType {
	case mcroutersync.AuthTypeApiKey:
		authimpl = auth.NewApiKeyAuth(cfg.AuthToken)
	default:
		authimpl = auth.NewNoneAuth()
	}

	_ = mcroutersync.NewMcRouterClient(cfg.McRouterHost)

	if authimpl == nil {

	}

}
