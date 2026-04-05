package main

import (
	"path/filepath"
	"testing"
)

func TestLoadAppConfig_ExampleConfig_Parses(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join("..", "..", "configs", "config.example.yaml")
	cfg, err := loadAppConfig(configPath)
	if err != nil {
		t.Fatalf("loadAppConfig(%q) error = %v", configPath, err)
	}

	if cfg == nil {
		t.Fatal("loadAppConfig() = nil, want config")
	}
	if cfg.StateStoreDSN == "" {
		t.Fatal("StateStoreDSN = empty, want example DSN")
	}
	if cfg.Sources["kvm"].Address != "examples/lab/kvm" {
		t.Fatalf("kvm source address = %q, want examples/lab/kvm", cfg.Sources["kvm"].Address)
	}
	if cfg.Plugins["example"] != "grpc://127.0.0.1:50071" {
		t.Fatalf("example plugin address = %q, want grpc://127.0.0.1:50071", cfg.Plugins["example"])
	}
}
