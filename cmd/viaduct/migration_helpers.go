package main

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/connectors"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
)

func connectorsForSpec(cfg *appConfig, spec *migratepkg.MigrationSpec) (connectors.Connector, connectors.Connector, error) {
	sourceConfig := resolveConnectorConfig(spec.Source.Address, string(spec.Source.Platform), "", "", false, cfg)
	targetConfig := resolveConnectorConfig(spec.Target.Address, string(spec.Target.Platform), "", "", false, cfg)

	sourceConnector, err := buildConnector(string(spec.Source.Platform), sourceConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build source connector: %w", err)
	}

	targetConnector, err := buildConnector(string(spec.Target.Platform), targetConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build target connector: %w", err)
	}

	return sourceConnector, targetConnector, nil
}
