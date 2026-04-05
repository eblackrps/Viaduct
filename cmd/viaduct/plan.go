package main

import (
	"fmt"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	var spec string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate a migration plan from a YAML spec",
		Long:  "Generate a migration plan from a declarative YAML specification.",
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := migratepkg.ParseSpec(spec)
			if err != nil {
				return fmt.Errorf("plan: %w", err)
			}

			if output == "table" {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Plan %s\n  Source: %s (%s)\n  Target: %s (%s)\n  Selectors: %d\n  Parallel: %d\n",
					parsed.Name,
					parsed.Source.Address,
					parsed.Source.Platform,
					parsed.Target.Address,
					parsed.Target.Platform,
					len(parsed.Workloads),
					parsed.Options.Parallel,
				)
				return nil
			}

			return printStructuredOutput(output, parsed)
		},
	}

	cmd.Flags().StringVar(&spec, "spec", "", "Path to the migration spec")
	cobra.CheckErr(cmd.MarkFlagRequired("spec"))

	return cmd
}
