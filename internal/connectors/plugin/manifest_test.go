package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_ValidManifest_LoadsExpectedMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginPath := filepath.Join(root, "viaduct-example-plugin.exe")
	if err := os.WriteFile(pluginPath, []byte("binary"), 0o644); err != nil {
		t.Fatalf("WriteFile(plugin) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(`{"name":"Example Plugin","platform":"example","version":"1.2.3","protocol_version":"v1"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(plugin.json) error = %v", err)
	}

	manifest, err := LoadManifest(pluginPath)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if manifest == nil {
		t.Fatal("LoadManifest() = nil, want manifest")
	}
	if manifest.Name != "Example Plugin" || manifest.Platform != "example" || manifest.Version != "1.2.3" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
}

func TestValidateManifest_UnsupportedProtocol_ReturnsError(t *testing.T) {
	t.Parallel()

	err := ValidateManifest(&Manifest{
		Name:            "Example Plugin",
		Platform:        "example",
		Version:         "1.0.0",
		ProtocolVersion: "v2",
	})
	if err == nil {
		t.Fatal("ValidateManifest() error = nil, want unsupported protocol error")
	}
}
