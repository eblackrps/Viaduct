package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current inventory and migration state",
		Long:  "Show current inventory and migration state, optionally filtered by source endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Status not yet implemented")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status not yet implemented for source %s\n", source)
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Optional source hypervisor endpoint")

	return cmd
}
