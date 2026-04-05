package main

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/connectorcatalog"
)

func openConnectorCatalog(cfg *appConfig) (*connectorcatalog.Catalog, error) {
	var pluginPaths map[string]string
	if cfg != nil {
		pluginPaths = cfg.Plugins
	}

	catalog, err := connectorcatalog.New(pluginPaths)
	if err != nil {
		return nil, fmt.Errorf("open connector catalog: %w", err)
	}
	return catalog, nil
}
