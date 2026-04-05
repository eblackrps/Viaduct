package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ProtocolVersion identifies the plugin RPC contract supported by Viaduct.
	ProtocolVersion = "v1"
)

// Manifest describes compatibility and identity metadata for a connector plugin.
type Manifest struct {
	// Name is the human-readable plugin name.
	Name string `json:"name"`
	// Platform is the platform identifier implemented by the plugin.
	Platform string `json:"platform"`
	// Version is the plugin version.
	Version string `json:"version"`
	// ProtocolVersion is the plugin RPC protocol version supported by the plugin.
	ProtocolVersion string `json:"protocol_version"`
}

// LoadManifest loads a plugin manifest from the executable directory when present.
func LoadManifest(path string) (*Manifest, error) {
	manifestPath, ok := manifestPathForPlugin(path)
	if !ok {
		return nil, nil
	}

	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load plugin manifest %s: %w", manifestPath, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, fmt.Errorf("load plugin manifest %s: %w", manifestPath, err)
	}
	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ValidateManifest validates required plugin manifest fields and compatibility markers.
func ValidateManifest(manifest *Manifest) error {
	if manifest == nil {
		return nil
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("plugin manifest: name is required")
	}
	if strings.TrimSpace(manifest.Platform) == "" {
		return fmt.Errorf("plugin manifest: platform is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("plugin manifest: version is required")
	}
	if strings.TrimSpace(manifest.ProtocolVersion) != ProtocolVersion {
		return fmt.Errorf("plugin manifest: unsupported protocol version %q", manifest.ProtocolVersion)
	}
	return nil
}

func manifestPathForPlugin(path string) (string, bool) {
	if strings.TrimSpace(path) == "" || strings.HasPrefix(path, "grpc://") {
		return "", false
	}
	return filepath.Join(filepath.Dir(path), "plugin.json"), true
}
