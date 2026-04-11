package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	verbose    bool
	output     string
	configPath string

	rootCmd = &cobra.Command{
		Use:   "viaduct",
		Short: "Hypervisor-agnostic workload migration and lifecycle management",
		Long: "Viaduct is a hypervisor-agnostic workload migration and lifecycle management " +
			"platform for discovery, planning, migration, and operations.",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			switch output {
			case "table", "json", "yaml":
				return nil
			default:
				return fmt.Errorf("invalid output format %q: must be table, json, or yaml", output)
			}
		},
	}
)

func init() {
	_ = os.Setenv("VIADUCT_VERSION", version)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "~/.viaduct/config.yaml", "Path to config file")

	rootCmd.AddCommand(
		newDiscoverCommand(),
		newDoctorCommand(),
		newPlanCommand(),
		newStartCommand(),
		newMigrateCommand(),
		newStatusCommand(),
		newStopCommand(),
		newRollbackCommand(),
		newVersionCommand(),
		newServeAPICommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
