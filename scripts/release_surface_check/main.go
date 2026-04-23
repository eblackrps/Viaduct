package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type dashboardPackage struct {
	Version string `json:"version"`
}

type surfaceExpectation struct {
	Path    string
	Needles []string
}

func main() {
	version, err := currentVersion()
	if err != nil {
		failf("resolve current version: %v", err)
	}

	releaseTag := "v" + version
	imageTag := "ghcr.io/eblackrps/viaduct:" + version
	mirrorTag := "docker.io/emb079/viaduct:" + version
	releaseNotePath := filepath.ToSlash(filepath.Join("docs", "releases", releaseTag+".md"))

	expectations := []surfaceExpectation{
		{
			Path: "README.md",
			Needles: []string{
				releaseTag,
				imageTag,
				"docs/releases/current.md",
			},
		},
		{
			Path: "INSTALL.md",
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
				"docs/releases/current.md",
			},
		},
		{
			Path: "QUICKSTART.md",
			Needles: []string{
				releaseTag,
				"docs/releases/current.md",
			},
		},
		{
			Path: filepath.Join("docs", "README.md"),
			Needles: []string{
				releaseTag,
				"releases/current.md",
			},
		},
		{
			Path: filepath.Join("docs", "getting-started", "quickstart.md"),
			Needles: []string{
				releaseTag,
				"../releases/current.md",
			},
		},
		{
			Path: filepath.Join("docs", "releases", "README.md"),
			Needles: []string{
				releaseTag + " release notes",
				"current.md",
			},
		},
		{
			Path: filepath.Join("docs", "releases", "current.md"),
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
				releaseNotePath,
			},
		},
		{
			Path: "CHANGELOG.md",
			Needles: []string{
				fmt.Sprintf("## [%s]", version),
			},
		},
		{
			Path: filepath.Join("deploy", "docker-compose.prod.yml"),
			Needles: []string{
				"image: " + imageTag,
			},
		},
		{
			Path: filepath.Join("deploy", "helm", "viaduct", "Chart.yaml"),
			Needles: []string{
				fmt.Sprintf("appVersion: %q", version),
			},
		},
		{
			Path: filepath.Join("deploy", "helm", "viaduct", "values.yaml"),
			Needles: []string{
				fmt.Sprintf("tag: %q", version),
			},
		},
		{
			Path: filepath.Join("site", "index.html"),
			Needles: []string{
				releaseTag,
				imageTag,
				releaseTag + " release notes",
			},
		},
	}

	failures := make([]string, 0)
	for _, expectation := range expectations {
		if missing := missingNeedles(expectation); len(missing) > 0 {
			failures = append(
				failures,
				fmt.Sprintf("%s is missing %s", filepath.ToSlash(expectation.Path), strings.Join(missing, ", ")),
			)
		}
	}

	if _, err := os.Stat(filepath.Join("docs", "releases", releaseTag+".md")); err != nil {
		failures = append(failures, fmt.Sprintf("%s is missing", releaseNotePath))
	}

	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "release surface drift detected:")
		for _, failure := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", failure)
		}
		os.Exit(1)
	}

	fmt.Printf("release surfaces match %s\n", releaseTag)
}

func currentVersion() (string, error) {
	payload, err := os.ReadFile(filepath.Join("web", "package.json"))
	if err != nil {
		return "", fmt.Errorf("read web/package.json: %w", err)
	}

	var pkg dashboardPackage
	if err := json.Unmarshal(payload, &pkg); err != nil {
		return "", fmt.Errorf("decode web/package.json: %w", err)
	}
	if strings.TrimSpace(pkg.Version) == "" {
		return "", fmt.Errorf("web/package.json does not declare a version")
	}
	return strings.TrimSpace(pkg.Version), nil
}

func missingNeedles(expectation surfaceExpectation) []string {
	payload, err := os.ReadFile(expectation.Path)
	if err != nil {
		return []string{fmt.Sprintf("readable content (%v)", err)}
	}

	content := string(payload)
	missing := make([]string, 0)
	for _, needle := range expectation.Needles {
		if !strings.Contains(content, needle) {
			missing = append(missing, fmt.Sprintf("%q", needle))
		}
	}
	return missing
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
