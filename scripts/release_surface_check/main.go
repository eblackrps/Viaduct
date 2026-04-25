package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type dashboardPackage struct {
	Version string `json:"version"`
}

type surfaceExpectation struct {
	Path          string
	Needles       []string
	CheckVersions bool
	ActiveSurface bool
}

func main() {
	version, err := currentVersion(".")
	if err != nil {
		failf("resolve current version: %v", err)
	}

	failures := checkReleaseSurfaces(".", version)
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "release surface drift detected:")
		for _, failure := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", failure)
		}
		os.Exit(1)
	}

	fmt.Printf("release surfaces match v%s\n", version)
}

func currentVersion(root string) (string, error) {
	payload, err := os.ReadFile(filepath.Join(root, "web", "package.json"))
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

func checkReleaseSurfaces(root, version string) []string {
	expectations, forbiddenActivePhrases, releaseNotePath := releaseSurfaceExpectations(version)
	failures := make([]string, 0)
	for _, expectation := range expectations {
		if missing := missingNeedles(root, expectation); len(missing) > 0 {
			failures = append(
				failures,
				fmt.Sprintf("%s is missing %s", filepath.ToSlash(expectation.Path), strings.Join(missing, ", ")),
			)
		}
		if expectation.CheckVersions {
			if stale := unexpectedVersionReferences(root, expectation.Path, version); len(stale) > 0 {
				failures = append(
					failures,
					fmt.Sprintf("%s has stale active version reference(s): %s", filepath.ToSlash(expectation.Path), strings.Join(stale, ", ")),
				)
			}
		}
		if expectation.ActiveSurface {
			if forbidden := forbiddenPhrases(root, expectation.Path, forbiddenActivePhrases); len(forbidden) > 0 {
				failures = append(
					failures,
					fmt.Sprintf("%s has pre-release or unsupported wording: %s", filepath.ToSlash(expectation.Path), strings.Join(forbidden, ", ")),
				)
			}
		}
	}

	if _, err := os.Stat(filepath.Join(root, "docs", "releases", "v"+version+".md")); err != nil {
		failures = append(failures, fmt.Sprintf("%s is missing", releaseNotePath))
	}
	return failures
}

func releaseSurfaceExpectations(version string) ([]surfaceExpectation, []string, string) {
	releaseTag := "v" + version
	imageTag := "ghcr.io/eblackrps/viaduct:" + version
	mirrorTag := "docker.io/emb079/viaduct:" + version
	releaseNotePath := filepath.ToSlash(filepath.Join("docs", "releases", releaseTag+".md"))
	forbiddenActivePhrases := []string{
		"prepared release surface",
		"tag workflow publishes",
		"before cutting this release",
		"expected to validate",
		"release surface for the next tag",
		"future release candidate",
		"control plane for virtualization migration",
		"shared operator workspace",
		"operator workspace",
		"golden path",
		"evidence export",
		"traceability surface",
		"runtime posture",
		"migration estate",
		"trust signals",
		"product surface",
		"single shared backend model",
		"backend trace visibility",
		"enterprise ready",
		"seamless migration",
		"fully automated migration",
		"production proven",
		"live-proven",
	}

	expectations := []surfaceExpectation{
		{
			Path: "README.md",
			Needles: []string{
				releaseTag,
				imageTag,
				"docs/releases/current.md",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: "INSTALL.md",
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
				"docs/releases/current.md",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: "QUICKSTART.md",
			Needles: []string{
				releaseTag,
				"docs/releases/current.md",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("docs", "README.md"),
			Needles: []string{
				releaseTag,
				"releases/current.md",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("docs", "getting-started", "installation.md"),
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("docs", "getting-started", "quickstart.md"),
			Needles: []string{
				releaseTag,
				"../releases/current.md",
			},
			CheckVersions: true,
			ActiveSurface: true,
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
				"current published release",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("docs", "releases", releaseTag+".md"),
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
				"published release",
				"The " + releaseTag + " release workflow published signed images, SBOMs, checksums, and native bundles.",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("docs", "operations", "docker.md"),
			Needles: []string{
				releaseTag,
				imageTag,
				mirrorTag,
				"mkdir -p config",
				"cp configs/config.example.yaml config/config.yaml",
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: "CHANGELOG.md",
			Needles: []string{
				fmt.Sprintf("## [%s]", version),
			},
		},
		{
			Path: "RELEASE.md",
			Needles: []string{
				"make release-surface-check",
				"make support-matrix-check",
				"make release-remote-check VERSION=" + version,
			},
			ActiveSurface: true,
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
				"/releases/tag/" + releaseTag,
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("site", "social-card.svg"),
			Needles: []string{
				releaseTag,
			},
			CheckVersions: true,
			ActiveSurface: true,
		},
		{
			Path: filepath.Join("web", "package.json"),
			Needles: []string{
				fmt.Sprintf(`"version": "%s"`, version),
			},
			CheckVersions: true,
		},
		{
			Path: "Makefile",
			Needles: []string{
				"release-surface-check:",
				"site-check:",
				"site-check-live:",
				"release-remote-check:",
			},
		},
		{
			Path: filepath.Join(".github", "workflows", "ci.yml"),
			Needles: []string{
				"make release-surface-check",
				"make site-check",
			},
		},
		{
			Path: filepath.Join(".github", "workflows", "pages.yml"),
			Needles: []string{
				"actions/setup-go",
				"make site-check",
				"go run ./scripts/site_validate -base-url",
			},
		},
		{
			Path: filepath.Join(".github", "workflows", "image.yml"),
			Needles: []string{
				"for example " + releaseTag,
				"go run ./scripts/site_validate",
			},
		},
	}
	return expectations, forbiddenActivePhrases, releaseNotePath
}

func missingNeedles(root string, expectation surfaceExpectation) []string {
	payload, err := os.ReadFile(filepath.Join(root, expectation.Path))
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

func unexpectedVersionReferences(root, path, version string) []string {
	// #nosec G304 -- paths come from the fixed release-surface expectation table in this script.
	payload, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return []string{fmt.Sprintf("readable content (%v)", err)}
	}
	pattern := regexp.MustCompile(`(?:v|viaduct:|viaduct_)[0-9]+\.[0-9]+\.[0-9]+|(?:ghcr\.io/eblackrps/viaduct|docker\.io/emb079/viaduct):[0-9]+\.[0-9]+\.[0-9]+`)
	matches := pattern.FindAllString(string(payload), -1)
	stale := make([]string, 0)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		normalized := match
		for _, prefix := range []string{
			"ghcr.io/eblackrps/viaduct:",
			"docker.io/emb079/viaduct:",
			"viaduct:",
			"viaduct_",
			"v",
		} {
			normalized = strings.TrimPrefix(normalized, prefix)
		}
		if normalized == version {
			continue
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		stale = append(stale, match)
	}
	return stale
}

func forbiddenPhrases(root, path string, phrases []string) []string {
	// #nosec G304 -- paths come from the fixed release-surface expectation table in this script.
	payload, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return []string{fmt.Sprintf("readable content (%v)", err)}
	}
	content := strings.ToLower(string(payload))
	found := make([]string, 0)
	for _, phrase := range phrases {
		if strings.Contains(content, strings.ToLower(phrase)) {
			found = append(found, fmt.Sprintf("%q", phrase))
		}
	}
	return found
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
