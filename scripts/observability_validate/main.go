package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type validationOptions struct {
	ViaductURL        string
	GrafanaURL        string
	TempoURL          string
	Timeout           time.Duration
	GrafanaUser       string
	GrafanaPassword   string
	ServiceAccountKey string
	TenantKey         string
}

func main() {
	options := validationOptions{}
	flag.StringVar(&options.ViaductURL, "viaduct-url", "http://127.0.0.1:8080", "Base URL for the Viaduct API")
	flag.StringVar(&options.GrafanaURL, "grafana-url", "http://127.0.0.1:3000", "Base URL for Grafana")
	flag.StringVar(&options.TempoURL, "tempo-url", "http://127.0.0.1:3200", "Base URL for Tempo")
	flag.DurationVar(&options.Timeout, "timeout", 45*time.Second, "How long to wait for the trace to arrive in Tempo")
	flag.StringVar(&options.GrafanaUser, "grafana-user", "admin", "Grafana admin username for datasource validation")
	flag.StringVar(&options.GrafanaPassword, "grafana-password", "admin", "Grafana admin password for datasource validation")
	flag.StringVar(&options.ServiceAccountKey, "service-account-key", "", "Optional Viaduct service account key for exercising a tenant-scoped report route")
	flag.StringVar(&options.TenantKey, "tenant-key", "", "Optional Viaduct tenant key for exercising a tenant-scoped report route")
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}

	if err := expectStatus(client, strings.TrimRight(options.TempoURL, "/")+"/ready", http.StatusOK); err != nil {
		exitWithError(fmt.Errorf("tempo readiness check failed: %w", err))
	}
	if err := expectGrafanaHealth(client, options); err != nil {
		exitWithError(fmt.Errorf("grafana health check failed: %w", err))
	}
	if err := expectGrafanaTempoDatasource(client, options); err != nil {
		exitWithError(fmt.Errorf("grafana datasource check failed: %w", err))
	}

	traceID, exercisedPath, err := exerciseViaduct(client, options)
	if err != nil {
		exitWithError(fmt.Errorf("viaduct telemetry exercise failed: %w", err))
	}

	if err := waitForTrace(client, options, traceID); err != nil {
		exitWithError(err)
	}

	fmt.Printf("observability validation passed\n")
	fmt.Printf("grafana=%s\n", strings.TrimRight(options.GrafanaURL, "/"))
	fmt.Printf("tempo=%s\n", strings.TrimRight(options.TempoURL, "/"))
	fmt.Printf("viaduct=%s%s\n", strings.TrimRight(options.ViaductURL, "/"), exercisedPath)
	fmt.Printf("trace_id=%s\n", traceID)
}

func expectStatus(client *http.Client, target string, expected int) error {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		return fmt.Errorf("%s returned %d, want %d", target, resp.StatusCode, expected)
	}
	return nil
}

func expectGrafanaHealth(client *http.Client, options validationOptions) error {
	target := strings.TrimRight(options.GrafanaURL, "/") + "/api/health"
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(options.GrafanaUser, options.GrafanaPassword)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d, want %d", target, resp.StatusCode, http.StatusOK)
	}
	return nil
}

func expectGrafanaTempoDatasource(client *http.Client, options validationOptions) error {
	target := strings.TrimRight(options.GrafanaURL, "/") + "/api/datasources/name/Tempo"
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(options.GrafanaUser, options.GrafanaPassword)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d, want %d", target, resp.StatusCode, http.StatusOK)
	}
	return nil
}

func exerciseViaduct(client *http.Client, options validationOptions) (string, string, error) {
	path := "/api/v1/about"
	headerName := ""
	headerValue := ""
	switch {
	case strings.TrimSpace(options.ServiceAccountKey) != "":
		path = "/api/v1/reports/summary?format=json"
		headerName = "X-Service-Account-Key"
		headerValue = strings.TrimSpace(options.ServiceAccountKey)
	case strings.TrimSpace(options.TenantKey) != "":
		path = "/api/v1/reports/summary?format=json"
		headerName = "X-API-Key"
		headerValue = strings.TrimSpace(options.TenantKey)
	}

	target := strings.TrimRight(options.ViaductURL, "/") + path
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", "", err
	}
	if headerName != "" {
		req.Header.Set(headerName, headerValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", path, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", path, fmt.Errorf("%s returned %d, want %d", target, resp.StatusCode, http.StatusOK)
	}

	traceID := strings.TrimSpace(resp.Header.Get("X-Trace-ID"))
	if traceID == "" {
		return "", path, fmt.Errorf("%s did not return X-Trace-ID; ensure VIADUCT_OTEL_ENABLED=true", target)
	}
	return traceID, path, nil
}

func waitForTrace(client *http.Client, options validationOptions, traceID string) error {
	target := strings.TrimRight(options.TempoURL, "/") + "/api/traces/" + traceID
	deadline := time.Now().Add(options.Timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				return fmt.Errorf("decode trace response: %w", err)
			}
			if len(payload) == 0 {
				return fmt.Errorf("tempo returned an empty trace payload for %s", traceID)
			}
			return nil
		}
		resp.Body.Close()

		lastErr = fmt.Errorf("%s returned %d", target, resp.StatusCode)
		time.Sleep(1 * time.Second)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("trace %s did not arrive in tempo before timeout", traceID)
	}
	return fmt.Errorf("tempo trace lookup failed: %w", lastErr)
}

func exitWithError(err error) {
	fmt.Printf("observability validation failed: %v\n", err)
	os.Exit(1)
}
