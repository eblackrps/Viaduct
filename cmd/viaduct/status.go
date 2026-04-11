package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var source string
	var runtimeStatus bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current inventory and migration state",
		Long:  "Show current inventory and migration state, optionally filtered by source endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtimeStatus {
				return runRuntimeStatus(cmd)
			}

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
	cmd.Flags().BoolVar(&runtimeStatus, "runtime", false, "Show the recorded local WebUI runtime state instead of snapshot and migration history")

	return cmd
}

func runRuntimeStatus(cmd *cobra.Command) error {
	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	state, err := readLocalRuntimeState(paths)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	report := localRuntimeStatusReport{
		Recorded:  state != nil,
		Reachable: runtimeReachable(state, 2*time.Second),
		State:     state,
	}
	switch {
	case state == nil:
		report.Message = "No recorded local Viaduct runtime is active."
	case report.Reachable:
		report.Message = fmt.Sprintf("Viaduct is reachable at %s.", state.BaseURL)
	default:
		report.Message = fmt.Sprintf("Runtime state exists for pid %d, but %s is not responding.", state.PID, state.BaseURL)
	}

	if output == "table" {
		fmt.Fprintln(cmd.OutOrStdout(), report.Message)
		if state != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Mode: %s\n", state.Mode)
			fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", state.ConfigPath)
			fmt.Fprintf(cmd.OutOrStdout(), "WebUI: %s\n", state.BaseURL)
			fmt.Fprintf(cmd.OutOrStdout(), "API:   %s\n", state.APIURL)
			fmt.Fprintf(cmd.OutOrStdout(), "PID: %d\n", state.PID)
			fmt.Fprintf(cmd.OutOrStdout(), "Detached: %s\n", strings.ToLower(fmt.Sprintf("%t", state.Detached)))
			if !state.StartedAt.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "Started: %s\n", state.StartedAt.Format(time.RFC3339))
			}
			if strings.TrimSpace(state.LogPath) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Log: %s\n", state.LogPath)
			}
		}
		return nil
	}

	return printStructuredOutput(output, report)
}
