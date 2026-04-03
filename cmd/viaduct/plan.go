package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	var spec string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate a migration plan from a YAML spec",
		Long:  "Generate a migration plan from a declarative YAML specification.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Plan not yet implemented for spec %s\n", spec)
			return nil
		},
	}

	cmd.Flags().StringVar(&spec, "spec", "", "Path to the migration spec")
	cobra.CheckErr(cmd.MarkFlagRequired("spec"))

	return cmd
}
