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

type serveAPIOptions struct {
	ConfigPath string
	Port       int
	WebDir     string
	Host       string
}

func newServeAPICommand() *cobra.Command {
	var port int
	var webDir string
	var host string

	cmd := &cobra.Command{
		Use:    "serve-api",
		Short:  "Start the Viaduct API server and serve built dashboard assets when available",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runServeAPI(ctx, serveAPIOptions{
				ConfigPath: configPath,
				Port:       port,
				WebDir:     webDir,
				Host:       host,
			})
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to bind the Viaduct API server to")
	cmd.Flags().StringVar(&host, "host", "", "Host interface to bind; leave empty to listen on all interfaces")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "Path to built dashboard assets; when empty, Viaduct auto-detects packaged or built web assets")
	return cmd
}

func runServeAPI(ctx context.Context, options serveAPIOptions) error {
	cfg, err := loadAppConfig(options.ConfigPath)
	if err != nil {
		return fmt.Errorf("serve-api: load config: %w", err)
	}
	catalog, err := openConnectorCatalog(cfg)
	if err != nil {
		return fmt.Errorf("serve-api: %w", err)
	}
	defer catalog.Close()

	stateStore, err := openStateStore(ctx, cfg)
	if err != nil {
		return fmt.Errorf("serve-api: open store: %w", err)
	}
	defer stateStore.Close()

	server := viaductapi.NewServer(discovery.NewEngine(), stateStore, options.Port, catalog)
	server.SetBuildInfo(version, commit, date)
	server.SetBindHost(options.Host)
	server.SetDashboardDir(options.WebDir)
	server.SetConnectorConfigResolver(func(platform models.Platform, address, credentialRef string) connectors.Config {
		return resolveMigrationConnectorConfig(address, string(platform), credentialRef, cfg)
	})
	return server.Start(ctx)
}
