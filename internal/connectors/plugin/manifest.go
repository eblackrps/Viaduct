package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	// MinimumViaductVersion is the oldest supported Viaduct host release when specified.
	MinimumViaductVersion string `json:"minimum_viaduct_version,omitempty"`
	// MaximumViaductVersion is the newest supported Viaduct host release when specified.
	MaximumViaductVersion string `json:"maximum_viaduct_version,omitempty"`
}

// LoadManifest loads a plugin manifest from the executable directory when present.
func LoadManifest(path string) (*Manifest, error) {
	manifestPath, ok := manifestPathForPlugin(path)
	if !ok {
		return nil, nil
	}

	// #nosec G304 -- plugin manifests are loaded from the plugin executable directory resolved by the host.
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
	if _, ok := parseSemanticVersion(manifest.MinimumViaductVersion); strings.TrimSpace(manifest.MinimumViaductVersion) != "" && !ok {
		return fmt.Errorf("plugin manifest: minimum_viaduct_version %q is invalid", manifest.MinimumViaductVersion)
	}
	if _, ok := parseSemanticVersion(manifest.MaximumViaductVersion); strings.TrimSpace(manifest.MaximumViaductVersion) != "" && !ok {
		return fmt.Errorf("plugin manifest: maximum_viaduct_version %q is invalid", manifest.MaximumViaductVersion)
	}
	return nil
}

// ValidateManifestCompatibility validates plugin compatibility markers against the supplied Viaduct host version.
func ValidateManifestCompatibility(manifest *Manifest, hostVersion string) error {
	return manifestSupportsHost(manifest, hostVersion)
}

func manifestSupportsHost(manifest *Manifest, hostVersion string) error {
	if manifest == nil {
		return nil
	}

	hostSemver, ok := parseSemanticVersion(hostVersion)
	if !ok {
		return nil
	}

	if minVersion, ok := parseSemanticVersion(manifest.MinimumViaductVersion); ok && compareSemanticVersions(hostSemver, minVersion) < 0 {
		return fmt.Errorf("plugin manifest: requires Viaduct >= %s", manifest.MinimumViaductVersion)
	}
	if maxVersion, ok := parseSemanticVersion(manifest.MaximumViaductVersion); ok && compareSemanticVersions(hostSemver, maxVersion) > 0 {
		return fmt.Errorf("plugin manifest: requires Viaduct <= %s", manifest.MaximumViaductVersion)
	}
	return nil
}

func manifestPathForPlugin(path string) (string, bool) {
	if strings.TrimSpace(path) == "" || strings.HasPrefix(path, "grpc://") {
		return "", false
	}
	return filepath.Join(filepath.Dir(path), "plugin.json"), true
}

func parseSemanticVersion(version string) ([3]int, bool) {
	trimmed := strings.TrimSpace(version)
	trimmed = strings.TrimPrefix(trimmed, "v")
	if trimmed == "" {
		return [3]int{}, false
	}

	core := trimmed
	if index := strings.IndexAny(core, "-+"); index >= 0 {
		core = core[:index]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}

	var parsed [3]int
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, false
		}
		parsed[index] = value
	}
	return parsed, true
}

func compareSemanticVersions(left, right [3]int) int {
	for index := range left {
		switch {
		case left[index] < right[index]:
			return -1
		case left[index] > right[index]:
			return 1
		}
	}
	return 0
}
