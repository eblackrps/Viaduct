package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	viaductapi "github.com/eblackrps/viaduct/internal/api"
	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/spf13/cobra"
)

type serveAPIOptions struct {
	ConfigPath                 string
	Port                       int
	WebDir                     string
	Host                       string
	LocalRuntime               bool
	AllowUnauthenticatedRemote bool
}

func newServeAPICommand() *cobra.Command {
	var port int
	var webDir string
	var host string
	var localRuntime bool
	var allowUnauthenticatedRemote bool

	cmd := &cobra.Command{
		Use:    "serve-api",
		Short:  "Start the Viaduct API server and serve built dashboard assets when available",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runServeAPI(ctx, serveAPIOptions{
				ConfigPath:                 configPath,
				Port:                       port,
				WebDir:                     webDir,
				Host:                       host,
				LocalRuntime:               localRuntime,
				AllowUnauthenticatedRemote: allowUnauthenticatedRemote,
			})
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to bind the Viaduct API server to")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface to bind; defaults to loopback for safe local operation")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "Path to built dashboard assets; when empty, Viaduct auto-detects packaged or built web assets")
	cmd.Flags().BoolVar(&localRuntime, "local-runtime", false, "Enable the local-runtime operator bootstrap affordances")
	cmd.Flags().BoolVar(&allowUnauthenticatedRemote, "allow-unauthenticated-remote", false, "Dangerous: allow a non-loopback bind even when no admin, tenant, or service-account credentials are configured")
	_ = cmd.Flags().MarkHidden("local-runtime")
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

	server, err := viaductapi.NewServer(discovery.NewEngine(), stateStore, options.Port, catalog)
	if err != nil {
		return fmt.Errorf("serve-api: create server: %w", err)
	}
	server.SetBuildInfo(version, commit, date)
	server.SetBindHost(options.Host)
	server.SetDashboardDir(options.WebDir)
	server.SetLocalRuntimeMode(options.LocalRuntime)
	server.SetAllowUnauthenticatedRemote(options.AllowUnauthenticatedRemote || strings.EqualFold(strings.TrimSpace(os.Getenv("VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE")), "true"))
	server.SetConnectorConfigResolver(func(platform models.Platform, address, credentialRef string) connectors.Config {
		return resolveMigrationConnectorConfig(address, string(platform), credentialRef, cfg)
	})
	return server.Start(ctx)
}
