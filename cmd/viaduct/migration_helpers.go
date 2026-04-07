package main

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/connectorcatalog"
	"github.com/eblackrps/viaduct/internal/connectors"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
)

func connectorsForSpec(cfg *appConfig, catalog *connectorcatalog.Catalog, spec *migratepkg.MigrationSpec) (sourceConnector connectors.Connector, targetConnector connectors.Connector, err error) {
	sourceConfig := resolveMigrationConnectorConfig(spec.Source.Address, string(spec.Source.Platform), spec.Source.CredentialRef, cfg)
	targetConfig := resolveMigrationConnectorConfig(spec.Target.Address, string(spec.Target.Platform), spec.Target.CredentialRef, cfg)

	sourceConnector, err = catalog.Build(spec.Source.Platform, sourceConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build source connector: %w", err)
	}

	targetConnector, err = catalog.Build(spec.Target.Platform, targetConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build target connector: %w", err)
	}

	return sourceConnector, targetConnector, nil
}
