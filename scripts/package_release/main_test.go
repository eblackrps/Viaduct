package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPackageRelease_CreatesBundleAndArchive(t *testing.T) {
	workspace := t.TempDir()

	mustWriteFile(t, filepath.Join(workspace, "README.md"), "# Viaduct\n")
	mustWriteFile(t, filepath.Join(workspace, "LICENSE"), "Apache License\n")
	mustWriteFile(t, filepath.Join(workspace, "CHANGELOG.md"), "# Changelog\n")
	mustWriteFile(t, filepath.Join(workspace, "CODE_OF_CONDUCT.md"), "# Code of Conduct\n")
	mustWriteFile(t, filepath.Join(workspace, "CONTRIBUTING.md"), "# Contributing\n")
	mustWriteFile(t, filepath.Join(workspace, "INSTALL.md"), "# Installation\n")
	mustWriteFile(t, filepath.Join(workspace, "QUICKSTART.md"), "# Quickstart\n")
	mustWriteFile(t, filepath.Join(workspace, "RELEASE.md"), "# Release\n")
	mustWriteFile(t, filepath.Join(workspace, "SECURITY.md"), "# Security\n")
	mustWriteFile(t, filepath.Join(workspace, "SUPPORT.md"), "# Support\n")
	mustWriteFile(t, filepath.Join(workspace, "UPGRADE.md"), "# Upgrade\n")
	mustWriteFile(t, filepath.Join(workspace, ".env.example"), "VIADUCT_ADMIN_KEY=\n")
	mustWriteFile(t, filepath.Join(workspace, "scripts", "install.sh"), "#!/usr/bin/env sh\n")
	mustWriteFile(t, filepath.Join(workspace, "scripts", "install.ps1"), "Write-Host 'ok'\n")
	mustWriteFile(t, filepath.Join(workspace, "docs", "guide.md"), "guide\n")
	mustWriteFile(t, filepath.Join(workspace, "configs", "config.example.yaml"), "sources: {}\n")
	mustWriteFile(t, filepath.Join(workspace, "examples", "lab", "README.md"), "lab\n")
	mustWriteFile(t, filepath.Join(workspace, "examples", "plugin-example", "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(workspace, "bin", "viaduct"), "binary\n")
	mustWriteFile(t, filepath.Join(workspace, "web", "dist", "index.html"), "<html></html>\n")

	options := releaseOptions{
		Workspace: workspace,
		Version:   "v1.0.0-rc1",
		Commit:    "abc1234",
		Date:      "2026-04-04T12:00:00Z",
		Binary:    filepath.Join("bin", "viaduct"),
		WebDir:    filepath.Join("web", "dist"),
		OutputDir: "dist",
	}

	if err := packageRelease(options); err != nil {
		t.Fatalf("packageRelease() error = %v", err)
	}

	packageName := "viaduct_v1.0.0-rc1_" + runtime.GOOS + "_" + runtime.GOARCH
	bundleDir := filepath.Join(workspace, "dist", packageName)
	manifestPath := filepath.Join(bundleDir, "release-manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}

	var manifest releaseManifest
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("Unmarshal(manifest) error = %v", err)
	}
	if manifest.Version != "v1.0.0-rc1" || manifest.Commit != "abc1234" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "INSTALL.md")); err != nil {
		t.Fatalf("install guide missing from bundle: %v", err)
	}
	moduleMarker, err := os.ReadFile(filepath.Join(bundleDir, "go.mod"))
	if err != nil {
		t.Fatalf("bundle module marker missing: %v", err)
	}
	if string(moduleMarker) != "module github.com/eblackrps/viaduct-release-bundle\n\ngo 1.24\n" {
		t.Fatalf("unexpected bundle module marker contents: %q", string(moduleMarker))
	}
	exampleModuleMarker, err := os.ReadFile(filepath.Join(bundleDir, "examples", "plugin-example", "go.mod"))
	if err != nil {
		t.Fatalf("example module marker missing: %v", err)
	}
	if string(exampleModuleMarker) != "module github.com/eblackrps/viaduct-release-bundle/examples/plugin-example\n\ngo 1.24\n" {
		t.Fatalf("unexpected example module marker contents: %q", string(exampleModuleMarker))
	}

	if _, err := os.Stat(filepath.Join(workspace, "dist", packageName+".zip")); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestPackageRelease_MissingBinary_ReturnsError(t *testing.T) {
	workspace := t.TempDir()
	mustWriteFile(t, filepath.Join(workspace, "README.md"), "# Viaduct\n")
	mustWriteFile(t, filepath.Join(workspace, "LICENSE"), "Apache License\n")
	mustWriteFile(t, filepath.Join(workspace, "CHANGELOG.md"), "# Changelog\n")
	mustWriteFile(t, filepath.Join(workspace, "CODE_OF_CONDUCT.md"), "# Code of Conduct\n")
	mustWriteFile(t, filepath.Join(workspace, "CONTRIBUTING.md"), "# Contributing\n")
	mustWriteFile(t, filepath.Join(workspace, "INSTALL.md"), "# Installation\n")
	mustWriteFile(t, filepath.Join(workspace, "QUICKSTART.md"), "# Quickstart\n")
	mustWriteFile(t, filepath.Join(workspace, "RELEASE.md"), "# Release\n")
	mustWriteFile(t, filepath.Join(workspace, "SECURITY.md"), "# Security\n")
	mustWriteFile(t, filepath.Join(workspace, "SUPPORT.md"), "# Support\n")
	mustWriteFile(t, filepath.Join(workspace, "UPGRADE.md"), "# Upgrade\n")
	mustWriteFile(t, filepath.Join(workspace, ".env.example"), "VIADUCT_ADMIN_KEY=\n")
	mustWriteFile(t, filepath.Join(workspace, "scripts", "install.sh"), "#!/usr/bin/env sh\n")
	mustWriteFile(t, filepath.Join(workspace, "scripts", "install.ps1"), "Write-Host 'ok'\n")
	mustWriteFile(t, filepath.Join(workspace, "docs", "guide.md"), "guide\n")
	mustWriteFile(t, filepath.Join(workspace, "configs", "config.example.yaml"), "sources: {}\n")
	mustWriteFile(t, filepath.Join(workspace, "examples", "lab", "README.md"), "lab\n")
	mustWriteFile(t, filepath.Join(workspace, "examples", "plugin-example", "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(workspace, "web", "dist", "index.html"), "<html></html>\n")

	err := packageRelease(releaseOptions{
		Workspace: workspace,
		Version:   "dev",
		Binary:    filepath.Join("bin", "viaduct"),
		WebDir:    filepath.Join("web", "dist"),
		OutputDir: "dist",
	})
	if err == nil {
		t.Fatal("packageRelease() error = nil, want missing binary error")
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
