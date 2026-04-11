package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var port int
	var host string
	var webDir string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Inspect the local Viaduct startup environment",
		Long:  "Inspect the local config, lab fixtures, bundled dashboard assets, and any recorded runtime state used by the default WebUI-first startup flow.",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := collectDoctorReport(configPath, webDir, host, port)
			if err != nil {
				return fmt.Errorf("doctor: %w", err)
			}

			if output == "table" {
				fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", report.ConfigPath)
				fmt.Fprintf(cmd.OutOrStdout(), "WebUI: %s\n", report.BaseURL)
				fmt.Fprintf(cmd.OutOrStdout(), "API:   %s\n", report.APIURL)
				for _, check := range report.Checks {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
				}
				return nil
			}

			return printStructuredOutput(output, report)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to inspect for the local operator runtime")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface to inspect for the local operator runtime")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "Path to built dashboard assets; when empty, Viaduct auto-detects packaged or built web assets")
	return cmd
}
