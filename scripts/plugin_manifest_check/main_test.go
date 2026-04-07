package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifestFile_CompatibleManifest_Expected(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifestPath := filepath.Join(root, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "name": "Example Plugin",
  "platform": "example",
  "version": "1.0.0",
  "protocol_version": "v1",
  "minimum_viaduct_version": "v1.2.0"
}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifest, err := validateManifestFile(manifestPath, "v1.3.0")
	if err != nil {
		t.Fatalf("validateManifestFile() error = %v", err)
	}
	if manifest.Platform != "example" {
		t.Fatalf("manifest.Platform = %q, want example", manifest.Platform)
	}
}

func TestValidateManifestFile_IncompatibleHost_ReturnsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifestPath := filepath.Join(root, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "name": "Example Plugin",
  "platform": "example",
  "version": "1.0.0",
  "protocol_version": "v1",
  "minimum_viaduct_version": "v2.0.0"
}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := validateManifestFile(manifestPath, "v1.3.0"); err == nil {
		t.Fatal("validateManifestFile() error = nil, want compatibility failure")
	}
}
