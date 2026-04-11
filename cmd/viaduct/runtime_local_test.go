package main

import (
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
	if report.Runtime.Recorded {
		t.Fatal("Runtime.Recorded = true, want false")
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
