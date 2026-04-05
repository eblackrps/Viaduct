package main

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/store"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current inventory and migration state",
		Long:  "Show current inventory and migration state, optionally filtered by source endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(configPath)
			if err != nil {
				return fmt.Errorf("status: load config: %w", err)
			}

			stateStore, err := openStateStore(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("status: open store: %w", err)
			}
			defer stateStore.Close()

			snapshots, err := stateStore.ListSnapshots(cmd.Context(), store.DefaultTenantID, "", 20)
			if err != nil {
				return fmt.Errorf("status: list snapshots: %w", err)
			}

			if source != "" {
				filtered := snapshots[:0]
				for _, snapshot := range snapshots {
					if snapshot.Source == source {
						filtered = append(filtered, snapshot)
					}
				}
				snapshots = filtered
			}

			migrations, err := stateStore.ListMigrations(cmd.Context(), store.DefaultTenantID, 20)
			if err != nil {
				return fmt.Errorf("status: list migrations: %w", err)
			}

			payload := struct {
				Snapshots  interface{} `json:"snapshots" yaml:"snapshots"`
				Migrations interface{} `json:"migrations" yaml:"migrations"`
			}{
				Snapshots:  snapshots,
				Migrations: migrations,
			}

			if output == "table" {
				fmt.Fprintf(cmd.OutOrStdout(), "Snapshots: %d\nMigrations: %d\n", len(snapshots), len(migrations))
				return nil
			}

			return printStructuredOutput(output, payload)
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Optional source hypervisor endpoint")

	return cmd
}
