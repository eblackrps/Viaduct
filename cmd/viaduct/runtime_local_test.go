package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveLocalRuntimePaths_ConfigDirectory_Expected(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config", "viaduct.yaml")
	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		t.Fatalf("resolveLocalRuntimePaths() error = %v", err)
	}

	wantRuntimeDir := filepath.Join(filepath.Dir(paths.ConfigPath), "runtime")
	if paths.RuntimeDir != wantRuntimeDir {
		t.Fatalf("RuntimeDir = %q, want %q", paths.RuntimeDir, wantRuntimeDir)
	}
	if filepath.Base(paths.StatePath) != localRuntimeStateFile {
		t.Fatalf("StatePath = %q, want file %q", paths.StatePath, localRuntimeStateFile)
	}
}

func TestEnsureLocalLabConfig_MissingConfig_WritesFixtureAddress_Expected(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "generated", "config.yaml")
	expanded, created, err := ensureLocalLabConfig(configPath)
	if err != nil {
		t.Fatalf("ensureLocalLabConfig() error = %v", err)
	}
	if !created {
		t.Fatal("created = false, want true")
	}

	payload, err := os.ReadFile(expanded)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", expanded, err)
	}
	if len(payload) == 0 {
		t.Fatal("generated config is empty")
	}
	if string(payload) == "" || !strings.Contains(filepath.ToSlash(string(payload)), "examples/lab/kvm") {
		t.Fatalf("generated config missing lab fixture reference:\n%s", string(payload))
	}
}

func TestReadWriteLocalRuntimeState_RoundTrip_Expected(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		t.Fatalf("resolveLocalRuntimePaths() error = %v", err)
	}

	now := time.Date(2026, time.April, 11, 23, 10, 0, 0, time.UTC)
	want := localRuntimeState{
		PID:        4242,
		Detached:   true,
		Host:       "127.0.0.1",
		Port:       18080,
		BaseURL:    "http://127.0.0.1:18080",
		APIURL:     "http://127.0.0.1:18080/api/v1/",
		ConfigPath: paths.ConfigPath,
		WebDir:     filepath.Join(t.TempDir(), "web"),
		LogPath:    paths.LogPath,
		Mode:       "local-lab",
		Version:    "v2.0.0",
		Commit:     "deadbee",
		StartedAt:  now,
	}

	if err := writeLocalRuntimeState(paths, want); err != nil {
		t.Fatalf("writeLocalRuntimeState() error = %v", err)
	}

	got, err := readLocalRuntimeState(paths)
	if err != nil {
		t.Fatalf("readLocalRuntimeState() error = %v", err)
	}
	if got == nil {
		t.Fatal("readLocalRuntimeState() = nil, want state")
	}
	if got.PID != want.PID || got.BaseURL != want.BaseURL || !got.StartedAt.Equal(now) {
		t.Fatalf("runtime state round-trip mismatch: got %#v want %#v", got, want)
	}
}

func TestStopCommand_StaleRuntimeStateClearedWithoutKill_Expected(t *testing.T) {
	root := t.TempDir()
	previousConfigPath := configPath
	configPath = filepath.Join(root, "config.yaml")
	defer func() {
		configPath = previousConfigPath
	}()

	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		t.Fatalf("resolveLocalRuntimePaths() error = %v", err)
	}
	if err := writeLocalRuntimeState(paths, localRuntimeState{
		PID:        0,
		Detached:   true,
		BaseURL:    "http://127.0.0.1:1",
		APIURL:     "http://127.0.0.1:1/api/v1/",
		ConfigPath: paths.ConfigPath,
		Version:    "dev",
		Commit:     "none",
		StartedAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("writeLocalRuntimeState() error = %v", err)
	}

	var output bytes.Buffer
	cmd := newStopCommand()
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stop command error = %v", err)
	}

	state, err := readLocalRuntimeState(paths)
	if err != nil {
		t.Fatalf("readLocalRuntimeState() error = %v", err)
	}
	if state != nil {
		t.Fatalf("runtime state = %#v, want cleared", state)
	}
	if !strings.Contains(output.String(), "Removed stale local Viaduct runtime record") {
		t.Fatalf("output = %q, want stale runtime message", output.String())
	}
}

func TestLocalBaseURL_UnspecifiedHost_UsesLoopback_Expected(t *testing.T) {
	t.Parallel()

	if got := localBaseURL("", 8080); got != "http://127.0.0.1:8080" {
		t.Fatalf("localBaseURL(\"\") = %q, want loopback URL", got)
	}
	if got := localBaseURL("0.0.0.0", 8080); got != "http://127.0.0.1:8080" {
		t.Fatalf("localBaseURL(\"0.0.0.0\") = %q, want loopback URL", got)
	}
}

func TestCollectDoctorReport_MissingConfigReportsLocalLabGuidance_Expected(t *testing.T) {
	t.Parallel()

	webDir := builtDashboardDir(t)
	configPath := filepath.Join(t.TempDir(), "viaduct", "config.yaml")

	report, err := collectDoctorReport(configPath, webDir, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("collectDoctorReport() error = %v", err)
	}
	if report.WebDir != webDir {
		t.Fatalf("WebDir = %q, want %q", report.WebDir, webDir)
	}
	if report.LabDir == "" || !strings.Contains(filepath.ToSlash(report.LabDir), "examples/lab/kvm") {
		t.Fatalf("LabDir = %q, want bundled lab fixtures", report.LabDir)
	}
	if len(report.Checks) < 4 {
		t.Fatalf("Checks = %d, want at least 4", len(report.Checks))
	}
	if report.Store.Backend != "memory" {
		t.Fatalf("Store.Backend = %q, want memory", report.Store.Backend)
	}
	if !report.Auth.LocalOnlyFallbackMode {
		t.Fatal("Auth.LocalOnlyFallbackMode = false, want true")
	}
	if report.Runtime.Recorded {
		t.Fatal("Runtime.Recorded = true, want false")
	}
}

func TestCollectDoctorReport_ParsedConfigIncludesStoreAndAuthSummaries_Expected(t *testing.T) {
	webDir := builtDashboardDir(t)
	root := t.TempDir()
	configPath := filepath.Join(root, "viaduct", "config.yaml")
	t.Setenv("VIADUCT_ADMIN_KEY", "doctor-admin-key")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte("insecure: false\nsources:\n  kvm:\n    address: examples/lab/kvm\ncredentials:\n  lab-kvm:\n    username: operator\nplugins:\n  sample: ./plugin\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	report, err := collectDoctorReport(configPath, webDir, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("collectDoctorReport() error = %v", err)
	}

	if !report.Config.Exists || !report.Config.Valid {
		t.Fatalf("Config summary = %#v, want existing valid config", report.Config)
	}
	if report.Config.SourceCount != 1 || report.Config.CredentialCount != 1 || report.Config.PluginCount != 1 {
		t.Fatalf("Config summary counts = %#v, want one source, credential, and plugin", report.Config)
	}
	if report.Store.Backend != "memory" {
		t.Fatalf("Store.Backend = %q, want memory", report.Store.Backend)
	}
	if !report.Auth.AdminKeyConfigured {
		t.Fatal("Auth.AdminKeyConfigured = false, want true")
	}
	if !report.Auth.SharedAccessReady {
		t.Fatal("Auth.SharedAccessReady = false, want true")
	}
}

func TestCollectDoctorReport_RecordedRuntimeIncludesReadinessAndAbout_Expected(t *testing.T) {
	t.Parallel()

	webDir := builtDashboardDir(t)
	configPath := filepath.Join(t.TempDir(), "viaduct", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte("insecure: false\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	runtimeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/health":
			fmt.Fprint(w, `{"status":"ok"}`)
		case "/readyz":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"ready"}`)
		case "/api/v1/about":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"version":"v3.2.1","store_backend":"memory","persistent_store":false,"local_operator_session_enabled":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer runtimeServer.Close()

	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		t.Fatalf("resolveLocalRuntimePaths() error = %v", err)
	}
	state := localRuntimeState{
		PID:        4242,
		Detached:   true,
		Host:       "127.0.0.1",
		Port:       8080,
		BaseURL:    runtimeServer.URL,
		APIURL:     runtimeServer.URL + "/api/v1/",
		ConfigPath: paths.ConfigPath,
		WebDir:     webDir,
		LogPath:    paths.LogPath,
		Mode:       "local-lab",
		Version:    "v3.2.1",
		Commit:     "deadbee",
		StartedAt:  time.Now().UTC(),
	}
	if err := writeLocalRuntimeState(paths, state); err != nil {
		t.Fatalf("writeLocalRuntimeState() error = %v", err)
	}

	report, err := collectDoctorReport(configPath, webDir, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("collectDoctorReport() error = %v", err)
	}

	if !report.Runtime.Recorded || !report.Runtime.Reachable {
		t.Fatalf("Runtime summary = %#v, want recorded reachable runtime", report.Runtime)
	}
	if report.Runtime.Readiness == nil || !report.Runtime.Readiness.Ready {
		t.Fatalf("Runtime readiness = %#v, want ready runtime", report.Runtime.Readiness)
	}
	if report.Runtime.Readiness.About.Version != "v3.2.1" {
		t.Fatalf("Runtime about version = %q, want v3.2.1", report.Runtime.Readiness.About.Version)
	}
	if !report.Runtime.Readiness.About.LocalOperatorSession {
		t.Fatal("Runtime about local operator session = false, want true")
	}
}

func TestCollectDoctorReport_RecordedRuntimeDegradedReadinessIncludesIssues_Expected(t *testing.T) {
	t.Parallel()

	webDir := builtDashboardDir(t)
	configPath := filepath.Join(t.TempDir(), "viaduct", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte("insecure: false\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	runtimeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/health":
			fmt.Fprint(w, `{"status":"ok"}`)
		case "/readyz":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status":"not_ready","policies_loaded":false,"issues":["no lifecycle policies are loaded from configs/policies","vmware connector circuit is open for vcsa.lab.local; retry after 60s"],"circuit_breakers":[{"state":"open"}]}`)
		case "/api/v1/about":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"version":"v3.2.1","store_backend":"memory","persistent_store":false,"local_operator_session_enabled":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer runtimeServer.Close()

	paths, err := resolveLocalRuntimePaths(configPath)
	if err != nil {
		t.Fatalf("resolveLocalRuntimePaths() error = %v", err)
	}
	state := localRuntimeState{
		PID:        5252,
		Detached:   true,
		Host:       "127.0.0.1",
		Port:       8080,
		BaseURL:    runtimeServer.URL,
		APIURL:     runtimeServer.URL + "/api/v1/",
		ConfigPath: paths.ConfigPath,
		WebDir:     webDir,
		LogPath:    paths.LogPath,
		Mode:       "local-lab",
		Version:    "v3.2.1",
		Commit:     "deadbee",
		StartedAt:  time.Now().UTC(),
	}
	if err := writeLocalRuntimeState(paths, state); err != nil {
		t.Fatalf("writeLocalRuntimeState() error = %v", err)
	}

	report, err := collectDoctorReport(configPath, webDir, "127.0.0.1", 8080)
	if err != nil {
		t.Fatalf("collectDoctorReport() error = %v", err)
	}

	if report.Runtime.Readiness == nil {
		t.Fatal("Runtime.Readiness = nil, want degraded readiness details")
	}
	if report.Runtime.Readiness.Ready {
		t.Fatalf("Runtime.Readiness = %#v, want degraded readiness", report.Runtime.Readiness)
	}
	if report.Runtime.Readiness.OpenCircuitCount != 1 {
		t.Fatalf("Runtime.Readiness.OpenCircuitCount = %d, want 1", report.Runtime.Readiness.OpenCircuitCount)
	}
	if report.Runtime.Readiness.PoliciesLoaded {
		t.Fatal("Runtime.Readiness.PoliciesLoaded = true, want false")
	}
	if len(report.Runtime.Readiness.Issues) != 2 {
		t.Fatalf("Runtime.Readiness.Issues = %#v, want two issues", report.Runtime.Readiness.Issues)
	}
	if !strings.Contains(report.Runtime.Message, "no lifecycle policies are loaded from configs/policies") {
		t.Fatalf("Runtime.Message = %q, want policy guidance", report.Runtime.Message)
	}
}

func builtDashboardDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<!doctype html><html><body><div id=\"root\"></div></body></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.html) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('viaduct');"), 0o644); err != nil {
		t.Fatalf("WriteFile(app.js) error = %v", err)
	}
	return root
}
