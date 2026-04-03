package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		Long:  "Print Viaduct version, commit hash, and build timestamp information.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "viaduct %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}
