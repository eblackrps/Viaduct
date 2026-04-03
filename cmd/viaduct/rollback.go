package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRollbackCommand() *cobra.Command {
	var migrationID string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Revert a migration to its pre-cutover state",
		Long:  "Revert a migration to its pre-cutover state using Viaduct rollback workflows.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Rollback not yet implemented for migration %s\n", migrationID)
			return nil
		},
	}

	cmd.Flags().StringVar(&migrationID, "migration", "", "Migration identifier")
	cobra.CheckErr(cmd.MarkFlagRequired("migration"))

	return cmd
}
