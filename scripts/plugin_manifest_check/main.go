package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	pluginhost "github.com/eblackrps/viaduct/internal/connectors/plugin"
)

func main() {
	manifestPath := flag.String("manifest", "", "Path to plugin.json")
	hostVersion := flag.String("host-version", "", "Viaduct host version to validate against")
	flag.Parse()

	if strings.TrimSpace(*manifestPath) == "" {
		fail("manifest path is required")
	}

	manifest, err := validateManifestFile(*manifestPath, *hostVersion)
	if err != nil {
		fail(err.Error())
	}

	fmt.Printf("plugin manifest ok: %s (%s) protocol=%s host=%s\n", manifest.Name, manifest.Version, manifest.ProtocolVersion, normalizedHostVersion(*hostVersion))
}

func validateManifestFile(path, hostVersion string) (*pluginhost.Manifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}

	var manifest pluginhost.Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest %s: %w", path, err)
	}
	if err := pluginhost.ValidateManifest(&manifest); err != nil {
		return nil, err
	}
	if err := pluginhost.ValidateManifestCompatibility(&manifest, normalizedHostVersion(hostVersion)); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func normalizedHostVersion(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "dev"
	}
	return trimmed
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
