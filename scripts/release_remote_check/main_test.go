package main

import (
	"strings"
	"testing"
)

func TestValidateAssets_CurrentReleaseAssetShape_Expected(t *testing.T) {
	assets := []releaseAsset{
		{Name: "SHA256SUMS"},
		{Name: "SHA256SUMS.sig"},
		{Name: "SHA256SUMS.pem"},
		{Name: "viaduct_3.2.1_image.cdx.json"},
		{Name: "viaduct_3.2.1_image.spdx.json"},
		{Name: "viaduct_3.2.1_linux_amd64.tar.gz"},
		{Name: "viaduct_3.2.1_linux_amd64.tar.gz.pem"},
		{Name: "viaduct_3.2.1_linux_amd64.tar.gz.sig"},
		{Name: "viaduct_3.2.1_linux_arm64.tar.gz"},
		{Name: "viaduct_3.2.1_linux_arm64.tar.gz.pem"},
		{Name: "viaduct_3.2.1_linux_arm64.tar.gz.sig"},
		{Name: "viaduct_3.2.1_darwin_arm64.tar.gz"},
		{Name: "viaduct_3.2.1_darwin_arm64.tar.gz.pem"},
		{Name: "viaduct_3.2.1_darwin_arm64.tar.gz.sig"},
		{Name: "viaduct_3.2.1_windows_amd64.tar.gz"},
		{Name: "viaduct_3.2.1_windows_amd64.tar.gz.pem"},
		{Name: "viaduct_3.2.1_windows_amd64.tar.gz.sig"},
	}

	if failures := validateAssets(assets); len(failures) != 0 {
		t.Fatalf("validateAssets() failures = %#v, want none", failures)
	}
}

func TestValidateAssets_MissingImageSBOMFails_Expected(t *testing.T) {
	failures := validateAssets([]releaseAsset{
		{Name: "SHA256SUMS"},
		{Name: "SHA256SUMS.sig"},
		{Name: "SHA256SUMS.pem"},
	})

	if !containsFailure(failures, "missing SBOM release asset") {
		t.Fatalf("validateAssets() failures = %#v, want SBOM failure", failures)
	}
}

func containsFailure(failures []string, needle string) bool {
	for _, failure := range failures {
		if strings.Contains(failure, needle) {
			return true
		}
	}
	return false
}
