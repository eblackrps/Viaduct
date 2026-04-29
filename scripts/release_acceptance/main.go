package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	adminKey             = "viaduct-acceptance-admin"  // #nosec G101 -- disposable release-acceptance credential scoped to a temporary local container.
	tenantKey            = "viaduct-acceptance-tenant" // #nosec G101 -- disposable release-acceptance credential scoped to a temporary local container.
	postgresImage        = "postgres:16-alpine"
	postgresNetworkAlias = "postgres"
)

func main() {
	var (
		image      string
		identity   string
		issuer     string
		port       int
		skipCosign bool
		keep       bool
	)

	flag.StringVar(&image, "image", "", "Published Viaduct image reference to validate, for example ghcr.io/eblackrps/viaduct:3.3.0")
	flag.StringVar(&identity, "certificate-identity", "", "Expected cosign certificate identity")
	flag.StringVar(&issuer, "certificate-oidc-issuer", "https://token.actions.githubusercontent.com", "Expected cosign OIDC issuer")
	flag.IntVar(&port, "port", 18080, "Loopback host port for the acceptance container")
	flag.BoolVar(&skipCosign, "skip-cosign", false, "Skip cosign verification when validating an unsigned local image")
	flag.BoolVar(&keep, "keep", false, "Keep Docker resources for debugging")
	flag.Parse()

	if strings.TrimSpace(image) == "" {
		failf("-image is required")
	}
	if !skipCosign && strings.TrimSpace(identity) == "" {
		failf("-certificate-identity is required unless -skip-cosign is set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	if err := run(ctx, keep, acceptanceConfig{
		Image:      strings.TrimSpace(image),
		Identity:   strings.TrimSpace(identity),
		Issuer:     strings.TrimSpace(issuer),
		Port:       port,
		SkipCosign: skipCosign,
	}); err != nil {
		failf("release acceptance failed: %v", err)
	}
	fmt.Printf("release acceptance passed for %s\n", image)
}

type acceptanceConfig struct {
	Image      string
	Identity   string
	Issuer     string
	Port       int
	SkipCosign bool
}

func run(ctx context.Context, keep bool, cfg acceptanceConfig) error {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	network := "viaduct-acceptance-" + suffix
	postgres := "viaduct-acceptance-postgres-" + suffix
	app := "viaduct-acceptance-app-" + suffix

	cleanup := func() {
		_ = command(context.Background(), "docker", "rm", "-f", app).Run()
		_ = command(context.Background(), "docker", "rm", "-f", postgres).Run()
		_ = command(context.Background(), "docker", "network", "rm", network).Run()
	}
	if !keep {
		defer cleanup()
	}

	if err := runLogged(ctx, "docker", "pull", cfg.Image); err != nil {
		return err
	}
	if !cfg.SkipCosign {
		if err := runLogged(
			ctx,
			"cosign",
			"verify",
			"--certificate-identity",
			cfg.Identity,
			"--certificate-oidc-issuer",
			cfg.Issuer,
			cfg.Image,
		); err != nil {
			return err
		}
	}

	if err := runLogged(ctx, "docker", "network", "create", network); err != nil {
		return err
	}
	if err := runLogged(
		ctx,
		"docker",
		"run",
		"-d",
		"--name",
		postgres,
		"--network",
		network,
		"--network-alias",
		postgresNetworkAlias,
		"-e",
		"POSTGRES_DB=viaduct",
		"-e",
		"POSTGRES_USER=viaduct",
		"-e",
		"POSTGRES_PASSWORD=viaduct-acceptance",
		postgresImage,
	); err != nil {
		return err
	}
	if err := waitForPostgres(ctx, postgres); err != nil {
		return err
	}

	adminHash := sha256.Sum256([]byte(adminKey))
	// #nosec G101 -- this DSN is a disposable credential for a temporary local PostgreSQL container created by this release-acceptance script.
	dsn := fmt.Sprintf("postgres://viaduct:viaduct-acceptance@%s:5432/viaduct?sslmode=disable", postgresNetworkAlias)
	if err := runLogged(
		ctx,
		"docker",
		"run",
		"-d",
		"--name",
		app,
		"--network",
		network,
		"-p",
		fmt.Sprintf("127.0.0.1:%d:8080", cfg.Port),
		"-e",
		"VIADUCT_ENVIRONMENT=production",
		"-e",
		"VIADUCT_ADMIN_KEY=sha256:"+hex.EncodeToString(adminHash[:]),
		"-e",
		"VIADUCT_STATE_STORE_DSN="+dsn,
		cfg.Image,
		"serve-api",
		"--host",
		"0.0.0.0",
		"--port",
		"8080",
	); err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
	if err := waitForHTTP(ctx, baseURL+"/readyz", http.StatusOK); err != nil {
		logs, _ := output(context.Background(), "docker", "logs", app)
		return fmt.Errorf("%w\ncontainer logs:\n%s", err, logs)
	}
	if err := expectJSONField(ctx, baseURL+"/api/v1/about", "name", "Viaduct"); err != nil {
		return err
	}
	if err := createTenant(ctx, baseURL); err != nil {
		return err
	}
	if err := currentTenant(ctx, baseURL); err != nil {
		return err
	}
	return nil
}

func waitForPostgres(ctx context.Context, container string) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(90 * time.Second)
	}
	for time.Now().Before(deadline) {
		err := command(ctx, "docker", "exec", container, "pg_isready", "-U", "viaduct", "-d", "viaduct").Run()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("postgres container did not become ready")
}

func waitForHTTP(ctx context.Context, url string, want int) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(90 * time.Second)
	}
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == want {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("%s did not return %d before timeout", url, want)
}

func expectJSONField(ctx context.Context, url, field, want string) error {
	response, err := doJSON(ctx, http.MethodGet, url, "", nil)
	if err != nil {
		return err
	}
	got, _ := response[field].(string)
	if got != want {
		return fmt.Errorf("%s field %s = %q, want %q", url, field, got, want)
	}
	return nil
}

func createTenant(ctx context.Context, baseURL string) error {
	payload := `{"id":"acceptance","name":"Acceptance","api_key":"` + tenantKey + `"}`
	headers := map[string]string{"X-Admin-Key": adminKey}
	response, err := doJSON(ctx, http.MethodPost, baseURL+"/api/v1/admin/tenants", payload, headers)
	if err != nil {
		return err
	}
	if response["id"] != "acceptance" {
		return fmt.Errorf("create tenant id = %#v, want acceptance", response["id"])
	}
	return nil
}

func currentTenant(ctx context.Context, baseURL string) error {
	headers := map[string]string{"X-API-Key": tenantKey}
	response, err := doJSON(ctx, http.MethodGet, baseURL+"/api/v1/tenants/current", "", headers)
	if err != nil {
		return err
	}
	if response["tenant_id"] != "acceptance" {
		return fmt.Errorf("current tenant = %#v, want acceptance", response["tenant_id"])
	}
	return nil
}

func doJSON(ctx context.Context, method, url, payload string, headers map[string]string) (map[string]any, error) {
	var body io.Reader
	if strings.TrimSpace(payload) != "" {
		body = strings.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	payloadBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned %d: %s", method, url, response.StatusCode, strings.TrimSpace(string(payloadBytes)))
	}
	var decoded map[string]any
	if err := json.Unmarshal(payloadBytes, &decoded); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", url, err)
	}
	return decoded, nil
}

func runLogged(ctx context.Context, name string, args ...string) error {
	fmt.Printf("+ %s %s\n", name, strings.Join(args, " "))
	return command(ctx, name, args...).Run()
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	var buffer bytes.Buffer
	cmd := command(ctx, name, args...)
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	err := cmd.Run()
	return buffer.String(), err
}

func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
