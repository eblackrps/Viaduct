package main

import (
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
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
	if cfg.Credentials["source/vcenter"].Username == "" {
		t.Fatal("source/vcenter credential username = empty, want populated example credential")
	}
	if cfg.Plugins["example"] != "grpc://127.0.0.1:50071" {
		t.Fatalf("example plugin address = %q, want grpc://127.0.0.1:50071", cfg.Plugins["example"])
	}
}

func TestResolveMigrationConnectorConfig_CredentialRefApplied_Expected(t *testing.T) {
	t.Parallel()

	cfg := &appConfig{
		Sources: map[string]connectors.Config{
			"vmware": {Insecure: true},
		},
		Credentials: map[string]connectors.Config{
			"source/vcenter": {
				Username: "administrator@vsphere.local",
				Password: "secret",
				Port:     443,
			},
		},
	}

	config := resolveMigrationConnectorConfig("https://vcsa.lab.local", "vmware", "source/vcenter", cfg)
	if config.Address != "https://vcsa.lab.local" {
		t.Fatalf("Address = %q, want https://vcsa.lab.local", config.Address)
	}
	if config.Username != "administrator@vsphere.local" || config.Password != "secret" {
		t.Fatalf("credentials = %#v, want credential-ref values", config)
	}
	if !config.Insecure || config.Port != 443 {
		t.Fatalf("transport settings = %#v, want insecure source config and port 443", config)
	}
}
