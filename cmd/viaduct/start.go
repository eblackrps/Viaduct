package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newStartCommand() *cobra.Command {
	var port int
	var host string
	var webDir string
	var openBrowserWhenReady bool
	var detach bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the local Viaduct operator runtime",
		Long:  "Start the Viaduct API and bundled dashboard together for the default WebUI-first local experience.",
		RunE: func(cmd *cobra.Command, args []string) error {
			startContext, err := prepareLocalRuntime(configPath, webDir, host, port, detach)
			if err != nil {
				return fmt.Errorf("start: %w", err)
			}

			if detach {
				return startDetached(cmd, *startContext)
			}
			return startForeground(cmd, *startContext, openBrowserWhenReady)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Port to bind the local operator runtime to")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface for the local operator runtime")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "Path to built dashboard assets; when empty, Viaduct auto-detects packaged or built web assets")
	cmd.Flags().BoolVar(&openBrowserWhenReady, "open-browser", true, "Open the Viaduct WebUI in the default browser when the runtime becomes healthy")
	cmd.Flags().BoolVar(&detach, "detach", false, "Start the runtime in the background and return once the local health check passes")
	return cmd
}

func startForeground(cmd *cobra.Command, startContext localStartContext, openBrowserWhenReady bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := writeLocalRuntimeState(startContext.paths, startContext.state); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	defer func() {
		_ = clearLocalRuntimeState(startContext.paths)
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServeAPI(ctx, serveAPIOptions{
			ConfigPath: startContext.state.ConfigPath,
			Port:       startContext.state.Port,
			WebDir:     startContext.state.WebDir,
			Host:       startContext.state.Host,
		})
	}()

	if err := waitForRuntimeHealth(startContext.state.BaseURL, 20*time.Second); err != nil {
		stop()
		serveErr := <-errCh
		if serveErr != nil {
			return serveErr
		}
		return fmt.Errorf("start: %w", err)
	}

	printStartSummary(cmd, startContext)
	if openBrowserWhenReady && isLikelyInteractiveSession() {
		if err := openBrowser(startContext.state.BaseURL); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Viaduct started, but the browser could not be opened automatically: %v\n", err)
		}
	}

	if err := <-errCh; err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Viaduct stopped. WebUI %s is no longer available.\n", startContext.state.BaseURL)
	return nil
}

func startDetached(cmd *cobra.Command, startContext localStartContext) error {
	if err := os.MkdirAll(startContext.paths.RuntimeDir, 0o750); err != nil {
		return fmt.Errorf("start: create runtime directory: %w", err)
	}

	logFile, err := os.OpenFile(startContext.paths.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("start: open runtime log: %w", err)
	}
	defer logFile.Close()

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("start: resolve current executable: %w", err)
	}

	args := []string{
		"serve-api",
		"--config", startContext.state.ConfigPath,
		"--port", strconv.Itoa(startContext.state.Port),
	}
	if startContext.state.Host != "" {
		args = append(args, "--host", startContext.state.Host)
	}
	if startContext.state.WebDir != "" {
		args = append(args, "--web-dir", startContext.state.WebDir)
	}

	// #nosec G204 -- the child process re-executes the current trusted Viaduct binary with explicit arguments.
	child := exec.Command(executable, args...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.Stdin = nil
	child.Env = os.Environ()

	if err := child.Start(); err != nil {
		return fmt.Errorf("start: launch background runtime: %w", err)
	}

	startContext.state.Detached = true
	startContext.state.PID = child.Process.Pid
	if err := writeLocalRuntimeState(startContext.paths, startContext.state); err != nil {
		_ = child.Process.Kill()
		return fmt.Errorf("start: %w", err)
	}

	if err := waitForRuntimeHealth(startContext.state.BaseURL, 20*time.Second); err != nil {
		_ = child.Process.Kill()
		_ = clearLocalRuntimeState(startContext.paths)
		return fmt.Errorf("start: %w", err)
	}

	printStartSummary(cmd, startContext)
	fmt.Fprintf(cmd.OutOrStdout(), "Background log: %s\n", startContext.paths.LogPath)
	return nil
}

func printStartSummary(cmd *cobra.Command, startContext localStartContext) {
	if startContext.configCreated {
		fmt.Fprintf(cmd.OutOrStdout(), "Generated local lab config at %s.\n", startContext.state.ConfigPath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Viaduct %s is running.\n", startContext.state.Version)
	fmt.Fprintf(cmd.OutOrStdout(), "WebUI: %s\n", startContext.state.BaseURL)
	fmt.Fprintf(cmd.OutOrStdout(), "API:   %s\n", startContext.state.APIURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", startContext.state.ConfigPath)
	if startContext.state.Detached {
		fmt.Fprintf(cmd.OutOrStdout(), "Mode: background local runtime (pid %d)\n", startContext.state.PID)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Mode: foreground local runtime (press Ctrl+C to stop)\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Use `viaduct status --runtime` for health and `viaduct doctor` for local environment checks.\n")
}
