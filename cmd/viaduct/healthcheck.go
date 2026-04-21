package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newHealthcheckCommand() *cobra.Command {
	var (
		url     string
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:    "healthcheck",
		Short:  "Probe a Viaduct HTTP health endpoint",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			if err := healthCheck(ctx, url); err != nil {
				return fmt.Errorf("healthcheck: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "http://127.0.0.1:8080/healthz", "Health endpoint URL to probe")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "Maximum time to wait for a healthy response")
	return cmd
}
