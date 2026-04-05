package main

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	var (
		source   string
		kind     string
		insecure bool
		username string
		password string
		save     bool
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover VM inventory from a hypervisor",
		Long:  "Discover VM inventory from a source hypervisor and normalize it into the Viaduct schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(configPath)
			if err != nil {
				return fmt.Errorf("discover: load config: %w", err)
			}

			catalog, err := openConnectorCatalog(cfg)
			if err != nil {
				return fmt.Errorf("discover: %w", err)
			}
			defer catalog.Close()

			connectorConfig := resolveConnectorConfig(source, kind, username, password, insecure, cfg)
			connector, err := catalog.Build(models.Platform(kind), connectorConfig)
			if err != nil {
				return fmt.Errorf("discover: %w", err)
			}

			engine := discovery.NewEngine()
			engine.AddSource(source, connector)

			result, err := engine.RunAll(cmd.Context())
			if err != nil {
				return fmt.Errorf("discover: %w", err)
			}

			if save {
				stateStore, err := openStateStore(cmd.Context(), cfg)
				if err != nil {
					return fmt.Errorf("discover: open store: %w", err)
				}
				defer stateStore.Close()

				for _, sourceResult := range result.Sources {
					if _, err := stateStore.SaveDiscovery(cmd.Context(), store.DefaultTenantID, &sourceResult); err != nil {
						return fmt.Errorf("discover: save snapshot: %w", err)
					}
				}
			}

			if err := PrintResult(output, result, verbose); err != nil {
				return fmt.Errorf("discover: format result: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "", "Source hypervisor endpoint")
	cmd.Flags().StringVarP(&kind, "type", "t", "", "Source platform type")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification for self-signed certificates")
	cmd.Flags().StringVarP(&username, "username", "u", "", "Source platform username")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Source platform password or token secret")
	cmd.Flags().BoolVar(&save, "save", false, "Persist discovered inventory to the configured state store")

	cobra.CheckErr(cmd.MarkFlagRequired("source"))
	cobra.CheckErr(cmd.MarkFlagRequired("type"))

	return cmd
}
