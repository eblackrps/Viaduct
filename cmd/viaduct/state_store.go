package main

import (
	"context"
	"fmt"

	"github.com/eblackrps/viaduct/internal/store"
)

func openStateStore(ctx context.Context, cfg *appConfig) (store.Store, error) {
	if cfg != nil && cfg.StateStoreDSN != "" {
		postgresStore, err := store.NewPostgresStore(ctx, cfg.StateStoreDSN)
		if err != nil {
			return nil, fmt.Errorf("open postgres store: %w", err)
		}
		return postgresStore, nil
	}

	return store.NewMemoryStore(), nil
}
