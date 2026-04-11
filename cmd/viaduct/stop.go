package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the recorded local Viaduct runtime",
		Long:  "Stop the local Viaduct runtime started through `viaduct start` and remove the recorded runtime state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := resolveLocalRuntimePaths(configPath)
			if err != nil {
				return fmt.Errorf("stop: %w", err)
			}

			state, err := readLocalRuntimeState(paths)
			if err != nil {
				return fmt.Errorf("stop: %w", err)
			}
			if state == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "No recorded local Viaduct runtime is active.")
				return nil
			}

			if err := killRuntimeProcess(state.PID); err != nil && runtimeReachable(state, 2*time.Second) {
				return fmt.Errorf("stop: %w", err)
			}

			deadline := time.Now().Add(10 * time.Second)
			for runtimeReachable(state, 750*time.Millisecond) && time.Now().Before(deadline) {
				time.Sleep(250 * time.Millisecond)
			}
			if runtimeReachable(state, 750*time.Millisecond) {
				return fmt.Errorf("stop: runtime %s is still responding after the stop request", state.BaseURL)
			}

			if err := clearLocalRuntimeState(paths); err != nil {
				return fmt.Errorf("stop: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Stopped Viaduct at %s.\n", state.BaseURL)
			return nil
		},
	}
}
