package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/eblackrps/viaduct/internal/store"
)

func openStateStore(ctx context.Context, cfg *appConfig) (store.Store, error) {
	if dsn := configuredStateStoreDSN(cfg); dsn != "" {
		postgresStore, err := store.NewPostgresStore(ctx, dsn)
		if err != nil {
			return nil, fmt.Errorf("open postgres store: %w", err)
		}
		return postgresStore, nil
	}

	return store.NewMemoryStore(), nil
}

func configuredStateStoreDSN(cfg *appConfig) string {
	if envDSN := strings.TrimSpace(os.Getenv("VIADUCT_STATE_STORE_DSN")); envDSN != "" {
		return envDSN
	}
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.StateStoreDSN)
}
