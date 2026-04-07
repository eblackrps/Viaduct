package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	viaductapi "github.com/eblackrps/viaduct/internal/api"
	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/spf13/cobra"
)

func newServeAPICommand() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:    "serve-api",
		Short:  "Start the Viaduct REST API server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(configPath)
			if err != nil {
				return fmt.Errorf("serve-api: load config: %w", err)
			}
			catalog, err := openConnectorCatalog(cfg)
			if err != nil {
				return fmt.Errorf("serve-api: %w", err)
			}
			defer catalog.Close()

			stateStore, err := openStateStore(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("serve-api: open store: %w", err)
			}
			defer stateStore.Close()

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			server := viaductapi.NewServer(discovery.NewEngine(), stateStore, port, catalog)
			server.SetBuildInfo(version, commit, date)
			server.SetConnectorConfigResolver(func(platform models.Platform, address, credentialRef string) connectors.Config {
				return resolveMigrationConnectorConfig(address, string(platform), credentialRef, cfg)
			})
			return server.Start(ctx)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to bind the Viaduct API server to")
	return cmd
}
