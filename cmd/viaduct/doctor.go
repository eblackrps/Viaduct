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
		Long:  "Inspect the local config, lab fixtures, bundled dashboard assets, and any recorded runtime state used by the default dashboard startup flow.",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := collectDoctorReport(configPath, webDir, host, port)
			if err != nil {
				return fmt.Errorf("doctor: %w", err)
			}

			if output == "table" {
				fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", report.ConfigPath)
				fmt.Fprintf(cmd.OutOrStdout(), "Dashboard: %s\n", report.BaseURL)
				fmt.Fprintf(cmd.OutOrStdout(), "API:   %s\n", report.APIURL)
				fmt.Fprintf(cmd.OutOrStdout(), "Store: %s\n", summarizeDoctorStore(report.Store))
				fmt.Fprintf(cmd.OutOrStdout(), "Auth:  %s\n", summarizeDoctorAuth(report.Auth))
				if report.Runtime.Readiness != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Ready: %s\n", summarizeDoctorRuntime(report.Runtime.Readiness))
				}
				for _, check := range report.Checks {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
				}
				return nil
			}

			return printStructuredOutput(output, report)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to inspect for the local runtime")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface to inspect for the local runtime")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "Path to built dashboard assets; when empty, Viaduct auto-detects packaged or built web assets")
	return cmd
}

func summarizeDoctorStore(report doctorStoreReport) string {
	backend := firstNonEmpty(report.Backend, "unknown")
	persistence := "non-persistent"
	if report.Persistent {
		persistence = "persistent"
	}
	if report.SchemaVersion > 0 {
		return fmt.Sprintf("%s (%s, schema %d)", backend, persistence, report.SchemaVersion)
	}
	return fmt.Sprintf("%s (%s)", backend, persistence)
}

func summarizeDoctorAuth(report doctorAuthReport) string {
	parts := make([]string, 0, 4)
	if report.AdminKeyConfigured {
		parts = append(parts, "admin configured")
	}
	if report.TenantKeyTenants > 0 {
		parts = append(parts, fmt.Sprintf("%d tenant-key tenant(s)", report.TenantKeyTenants))
	}
	if report.ServiceAccountKeys > 0 {
		parts = append(parts, fmt.Sprintf("%d service account key(s)", report.ServiceAccountKeys))
	}
	if len(parts) == 0 {
		return "local-only fallback"
	}
	return strings.Join(parts, ", ")
}

func summarizeDoctorRuntime(report *doctorRuntimeReadiness) string {
	if report == nil {
		return "not inspected"
	}
	if report.Ready {
		return fmt.Sprintf("ready (%s)", firstNonEmpty(report.About.Version, "current build"))
	}
	if report.InspectionErr != "" {
		return fmt.Sprintf("degraded (%s)", report.InspectionErr)
	}
	if issues := readinessIssuesSummary(*report); issues != "" {
		return fmt.Sprintf("degraded (%s)", issues)
	}
	return fmt.Sprintf("degraded (%s, %d)", firstNonEmpty(report.Status, "unknown"), report.HTTPStatus)
}
