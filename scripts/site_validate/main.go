package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type dashboardPackage struct {
	Version string `json:"version"`
}

func main() {
	var (
		siteDir string
		baseURL string
	)
	flag.StringVar(&siteDir, "site-dir", "site", "Path to the static site directory")
	flag.StringVar(&baseURL, "base-url", "", "Optional deployed site URL to verify with HTTP GET requests")
	flag.Parse()

	version, err := currentVersion()
	if err != nil {
		failf("resolve current version: %v", err)
	}
	if err := validateLocalSite(siteDir, version); err != nil {
		failf("validate local site: %v", err)
	}
	if strings.TrimSpace(baseURL) != "" {
		if err := validateDeployedSite(strings.TrimRight(strings.TrimSpace(baseURL), "/")); err != nil {
			failf("validate deployed site: %v", err)
		}
	}
	fmt.Printf("site validation passed for v%s\n", version)
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

func validateLocalSite(siteDir, version string) error {
	indexPath := filepath.Join(siteDir, "index.html")
	// #nosec G304 -- siteDir is an explicit release-owner validation input and only the static site index is read.
	index, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", indexPath, err)
	}
	content := string(index)
	for _, required := range []string{
		"v" + version,
		"ghcr.io/eblackrps/viaduct:" + version,
		"/releases/tag/v" + version,
	} {
		if !strings.Contains(content, required) {
			return fmt.Errorf("index.html missing %q", required)
		}
	}
	if stale := staleVersionReferences(content, version); len(stale) > 0 {
		return fmt.Errorf("index.html contains stale version reference(s): %s", strings.Join(stale, ", "))
	}
	if err := validateLocalReferences(siteDir, "index.html", content); err != nil {
		return err
	}
	// #nosec G304 -- optional 404.html is read only from the explicit static site directory being validated.
	if payload, err := os.ReadFile(filepath.Join(siteDir, "404.html")); err == nil {
		if err := validateLocalReferences(siteDir, "404.html", string(payload)); err != nil {
			return err
		}
	}
	pagesWorkflow, err := os.ReadFile(filepath.Join(".github", "workflows", "pages.yml"))
	if err != nil {
		return fmt.Errorf("read pages workflow: %w", err)
	}
	if !strings.Contains(string(pagesWorkflow), "path: site") {
		return fmt.Errorf("pages workflow does not upload the site directory")
	}
	return nil
}

func staleVersionReferences(content, version string) []string {
	pattern := regexp.MustCompile(`(?:v|viaduct:)[0-9]+\.[0-9]+\.[0-9]+`)
	matches := pattern.FindAllString(content, -1)
	stale := make([]string, 0)
	seen := make(map[string]struct{})
	for _, match := range matches {
		normalized := strings.TrimPrefix(strings.TrimPrefix(match, "viaduct:"), "v")
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

func validateLocalReferences(siteDir, sourceName, content string) error {
	pattern := regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(content, -1) {
		if len(match) != 2 || externalReference(match[1]) {
			continue
		}
		ref := strings.TrimSpace(match[1])
		if ref == "" || ref == "/" || strings.HasPrefix(ref, "#") {
			continue
		}
		candidate := filepath.Join(siteDir, filepath.FromSlash(strings.TrimPrefix(ref, "./")))
		if _, err := os.Stat(candidate); err != nil {
			return fmt.Errorf("%s references missing local asset %s", sourceName, filepath.ToSlash(candidate))
		}
	}
	return nil
}

func externalReference(ref string) bool {
	return strings.HasPrefix(ref, "http://") ||
		strings.HasPrefix(ref, "https://") ||
		strings.HasPrefix(ref, "mailto:") ||
		strings.HasPrefix(ref, "tel:")
}

func validateDeployedSite(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("-base-url must be an absolute URL")
	}
	for _, path := range []string{
		"/",
		"/styles.css",
		"/favicon.svg",
		"/social-card.svg",
		"/assets/pilot-workspace.png",
		"/assets/dependency-graph.png",
		"/assets/reports-history.png",
	} {
		target := strings.TrimRight(baseURL, "/") + path
		if err := expectHTTP200(target); err != nil {
			return err
		}
	}
	return nil
}

func expectHTTP200(target string) error {
	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(target)
	if err != nil {
		return fmt.Errorf("GET %s: %w", target, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", target, response.StatusCode)
	}
	return nil
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
