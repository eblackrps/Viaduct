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
	version, err := currentVersion()
	if err != nil {
		failf("resolve current version: %v", err)
	}

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
			Path: filepath.Join(".github", "workflows", "image.yml"),
			Needles: []string{
				"for example " + releaseTag,
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
		if expectation.CheckVersions {
			if stale := unexpectedVersionReferences(expectation.Path, version); len(stale) > 0 {
				failures = append(
					failures,
					fmt.Sprintf("%s has stale active version reference(s): %s", filepath.ToSlash(expectation.Path), strings.Join(stale, ", ")),
				)
			}
		}
		if expectation.ActiveSurface {
			if forbidden := forbiddenPhrases(expectation.Path, forbiddenActivePhrases); len(forbidden) > 0 {
				failures = append(
					failures,
					fmt.Sprintf("%s has pre-release wording: %s", filepath.ToSlash(expectation.Path), strings.Join(forbidden, ", ")),
				)
			}
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

func unexpectedVersionReferences(path, version string) []string {
	// #nosec G304 -- paths come from the fixed release-surface expectation table in this script.
	payload, err := os.ReadFile(path)
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

func forbiddenPhrases(path string, phrases []string) []string {
	// #nosec G304 -- paths come from the fixed release-surface expectation table in this script.
	payload, err := os.ReadFile(path)
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
