package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type releaseInfo struct {
	Name        string         `json:"name"`
	TagName     string         `json:"tagName"`
	URL         string         `json:"url"`
	Body        string         `json:"body"`
	ReleaseInfo []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name string `json:"name"`
}

type releaseListItem struct {
	TagName  string `json:"tagName"`
	IsLatest bool   `json:"isLatest"`
}

func main() {
	var (
		version string
		baseURL string
		offline bool
	)
	flag.StringVar(&version, "version", "3.2.1", "Release version without leading v")
	flag.StringVar(&baseURL, "base-url", "https://viaducthq.com", "Published site URL")
	flag.BoolVar(&offline, "offline", false, "Only validate local command wiring; skip network and tool-dependent remote checks")
	flag.Parse()

	if offline {
		fmt.Println("offline mode requested; remote release, image, cosign, and live-site checks were not run")
		return
	}

	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		failf("-version is required")
	}
	if err := checkRequiredTools("gh", "docker", "cosign", "go"); err != nil {
		failf("%v", err)
	}

	tag := "v" + version
	image := "ghcr.io/eblackrps/viaduct:" + version
	cosignIdentity := "https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/" + tag

	release, err := githubRelease(tag)
	if err != nil {
		failf("%v", err)
	}
	latestTag, err := latestReleaseTag()
	if err != nil {
		failf("%v", err)
	}
	if failures := validateRelease(tag, latestTag, image, cosignIdentity, release); len(failures) > 0 {
		failf("release metadata check failed:\n- %s", strings.Join(failures, "\n- "))
	}

	if _, err := runCommand(2*time.Minute, "docker", "manifest", "inspect", image); err != nil {
		failf("inspect GHCR image %s: %v", image, err)
	}
	if _, err := runCommand(
		5*time.Minute,
		"cosign", "verify", image,
		"--certificate-identity", cosignIdentity,
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
	); err != nil {
		failf("cosign verify %s: %v", image, err)
	}
	if _, err := runCommand(2*time.Minute, "go", "run", "./scripts/site_validate", "-base-url="+strings.TrimRight(baseURL, "/")); err != nil {
		failf("live site validation failed: %v", err)
	}

	fmt.Printf("remote release checks passed for %s (%s)\n", tag, image)
}

func githubRelease(tag string) (releaseInfo, error) {
	output, err := runCommand(
		2*time.Minute,
		"gh", "release", "view", tag,
		"--repo", "eblackrps/Viaduct",
		"--json", "name,tagName,url,assets,body",
	)
	if err != nil {
		return releaseInfo{}, fmt.Errorf("view GitHub Release %s: %w", tag, err)
	}
	var release releaseInfo
	if err := json.Unmarshal([]byte(output), &release); err != nil {
		return releaseInfo{}, fmt.Errorf("decode GitHub Release metadata: %w", err)
	}
	return release, nil
}

func latestReleaseTag() (string, error) {
	output, err := runCommand(
		2*time.Minute,
		"gh", "release", "list",
		"--repo", "eblackrps/Viaduct",
		"--limit", "10",
		"--json", "tagName,isLatest",
	)
	if err != nil {
		return "", fmt.Errorf("list GitHub Releases: %w", err)
	}
	var releases []releaseListItem
	if err := json.Unmarshal([]byte(output), &releases); err != nil {
		return "", fmt.Errorf("decode GitHub Release list: %w", err)
	}
	for _, release := range releases {
		if release.IsLatest {
			return release.TagName, nil
		}
	}
	return "", fmt.Errorf("GitHub Release list did not mark any release as latest")
}

func validateRelease(tag, latestTag, image, cosignIdentity string, release releaseInfo) []string {
	failures := make([]string, 0)
	if release.TagName != tag {
		failures = append(failures, fmt.Sprintf("tagName = %q, want %q", release.TagName, tag))
	}
	if latestTag != tag {
		failures = append(failures, fmt.Sprintf("release is not marked latest; latest is %q", latestTag))
	}
	body := strings.ToLower(release.Body)
	for _, forbidden := range []string{
		"prepared release surface",
		"after the tag workflow publishes",
		"before cutting this release",
		"expected to validate",
		"future release candidate",
	} {
		if strings.Contains(body, forbidden) {
			failures = append(failures, fmt.Sprintf("release body contains %q", forbidden))
		}
	}
	for _, required := range []string{image, cosignIdentity} {
		if !strings.Contains(release.Body, required) {
			failures = append(failures, fmt.Sprintf("release body missing %q", required))
		}
	}
	failures = append(failures, validateAssets(release.ReleaseInfo)...)
	return failures
}

func validateAssets(assets []releaseAsset) []string {
	names := make([]string, 0, len(assets))
	nameSet := make(map[string]struct{}, len(assets))
	for _, asset := range assets {
		names = append(names, asset.Name)
		nameSet[asset.Name] = struct{}{}
	}

	failures := make([]string, 0)
	for _, required := range []string{"SHA256SUMS", "SHA256SUMS.sig", "SHA256SUMS.pem"} {
		if _, ok := nameSet[required]; !ok {
			failures = append(failures, fmt.Sprintf("missing release asset %s", required))
		}
	}
	if !releaseHasImageSBOM(names) {
		failures = append(failures, "missing SBOM release asset")
	}
	for _, platform := range []string{"linux_amd64", "linux_arm64", "darwin_arm64", "windows_amd64"} {
		if !assetContainsEither(names, platform, strings.ReplaceAll(platform, "_", "-")) {
			failures = append(failures, fmt.Sprintf("missing native bundle for %s", platform))
			continue
		}
		if !assetContainsPlatformSidecar(names, platform, ".sig") {
			failures = append(failures, fmt.Sprintf("missing signature sidecar for %s", platform))
		}
		if !assetContainsPlatformSidecar(names, platform, ".pem") {
			failures = append(failures, fmt.Sprintf("missing certificate sidecar for %s", platform))
		}
	}
	return failures
}

func assetContains(names []string, needle string) bool {
	needle = strings.ToLower(needle)
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), needle) {
			return true
		}
	}
	return false
}

func assetContainsEither(names []string, needles ...string) bool {
	for _, needle := range needles {
		if assetContains(names, needle) {
			return true
		}
	}
	return false
}

func releaseHasImageSBOM(names []string) bool {
	hasSPDX := false
	hasCycloneDX := false
	for _, name := range names {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "image.spdx") || strings.Contains(lower, "spdx") {
			hasSPDX = true
		}
		if strings.Contains(lower, "image.cdx") || strings.Contains(lower, "cyclonedx") || strings.Contains(lower, "cdx") {
			hasCycloneDX = true
		}
	}
	return hasSPDX && hasCycloneDX
}

func assetContainsPlatformSidecar(names []string, platform, suffix string) bool {
	alternate := strings.ReplaceAll(platform, "_", "-")
	for _, name := range names {
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, suffix) && (strings.Contains(lower, platform) || strings.Contains(lower, alternate)) {
			return true
		}
	}
	return false
}

func checkRequiredTools(names ...string) error {
	missing := make([]string, 0)
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required tool(s) not found on PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}

func runCommand(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("%s timed out: %w", name, ctx.Err())
		}
		if message != "" {
			return "", fmt.Errorf("%w: %s", err, message)
		}
		return "", err
	}
	return stdout.String(), nil
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
