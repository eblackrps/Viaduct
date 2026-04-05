package main

import (
	"encoding/json"
	"fmt"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/spf13/cobra"
)

func newRollbackCommand() *cobra.Command {
	var migrationID string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Revert a migration to its pre-cutover state",
		Long:  "Revert a migration to its pre-cutover state using Viaduct rollback workflows.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadAppConfig(configPath)
			if err != nil {
				return fmt.Errorf("rollback: load config: %w", err)
			}

			stateStore, err := openStateStore(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("rollback: open store: %w", err)
			}
			defer stateStore.Close()

			record, err := stateStore.GetMigration(cmd.Context(), store.DefaultTenantID, migrationID)
			if err != nil {
				return fmt.Errorf("rollback: load migration %s: %w", migrationID, err)
			}

			var state migratepkg.MigrationState
			if err := json.Unmarshal(record.RawJSON, &state); err != nil {
				return fmt.Errorf("rollback: decode migration state: %w", err)
			}

			spec := &migratepkg.MigrationSpec{
				Name: state.SpecName,
				Source: migratepkg.SourceSpec{
					Address:  state.SourceAddress,
					Platform: state.SourcePlatform,
				},
				Target: migratepkg.TargetSpec{
					Address:  state.TargetAddress,
					Platform: state.TargetPlatform,
				},
			}

			sourceConnector, targetConnector, err := connectorsForSpec(cfg, spec)
			if err != nil {
				return fmt.Errorf("rollback: %w", err)
			}

			result, err := migratepkg.NewRollbackManager(stateStore, sourceConnector, targetConnector).Rollback(cmd.Context(), migrationID)
			if err != nil {
				return fmt.Errorf("rollback: %w", err)
			}

			if output == "table" {
				fmt.Fprintf(cmd.OutOrStdout(), "Rollback %s removed %d target VMs and restored %d source VMs\n", migrationID, result.TargetVMsRemoved, result.SourceVMsRestored)
				return nil
			}

			return printStructuredOutput(output, result)
		},
	}

	cmd.Flags().StringVar(&migrationID, "migration", "", "Migration identifier")
	cobra.CheckErr(cmd.MarkFlagRequired("migration"))

	return cmd
}
