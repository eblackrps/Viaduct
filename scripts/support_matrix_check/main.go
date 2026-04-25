package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type supportMatrix struct {
	Platforms []supportPlatform `json:"platforms"`
}

type supportPlatform struct {
	Name        string   `json:"name"`
	SiteNames   []string `json:"site_names"`
	ReadmeNames []string `json:"readme_names"`
	Validation  string   `json:"validation"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	failures := checkSupportMatrix(*root)
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "support matrix drift detected:")
		for _, failure := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", failure)
		}
		os.Exit(1)
	}
	fmt.Println("support matrix claims match checked docs")
}

func checkSupportMatrix(root string) []string {
	matrix, err := loadSupportMatrix(root)
	if err != nil {
		return []string{err.Error()}
	}
	failures := validateSupportMatrixData(matrix)

	supportMarkdown, err := readRepoFile(root, filepath.Join("docs", "reference", "support-matrix.md"))
	if err != nil {
		failures = append(failures, err.Error())
	} else {
		failures = append(failures, validateSupportMarkdown(matrix, supportMarkdown)...)
	}

	siteIndex, err := readRepoFile(root, filepath.Join("site", "index.html"))
	if err != nil {
		failures = append(failures, err.Error())
	} else {
		failures = append(failures, validateNamedClaims("site/index.html", siteIndex, matrix, func(platform supportPlatform) []string {
			return platform.SiteNames
		})...)
		failures = append(failures, validateSitePlatformList(siteIndex, matrix)...)
	}

	readme, err := readRepoFile(root, "README.md")
	if err != nil {
		failures = append(failures, err.Error())
	} else {
		failures = append(failures, validateNamedClaims("README.md", readme, matrix, func(platform supportPlatform) []string {
			return platform.ReadmeNames
		})...)
		failures = append(failures, validateReadmePlatformTable(readme, matrix)...)
	}

	for _, path := range []string{
		"README.md",
		"INSTALL.md",
		"QUICKSTART.md",
		filepath.Join("docs", "architecture.md"),
		filepath.Join("docs", "getting-started", "installation.md"),
		filepath.Join("docs", "getting-started", "quickstart.md"),
		filepath.Join("site", "index.html"),
	} {
		content, err := readRepoFile(root, path)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if forbidden := forbiddenSupportClaims(content); len(forbidden) > 0 {
			failures = append(failures, fmt.Sprintf("%s contains unsupported support claim(s): %s", filepath.ToSlash(path), strings.Join(forbidden, ", ")))
		}
	}

	return failures
}

func loadSupportMatrix(root string) (supportMatrix, error) {
	payload, err := os.ReadFile(filepath.Join(root, "docs", "reference", "support-matrix.json"))
	if err != nil {
		return supportMatrix{}, fmt.Errorf("read docs/reference/support-matrix.json: %w", err)
	}
	var matrix supportMatrix
	if err := json.Unmarshal(payload, &matrix); err != nil {
		return supportMatrix{}, fmt.Errorf("decode docs/reference/support-matrix.json: %w", err)
	}
	return matrix, nil
}

func validateSupportMatrixData(matrix supportMatrix) []string {
	failures := make([]string, 0)
	seen := make(map[string]struct{}, len(matrix.Platforms))
	for _, platform := range matrix.Platforms {
		name := strings.TrimSpace(platform.Name)
		if name == "" {
			failures = append(failures, "support matrix contains an unnamed platform")
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			failures = append(failures, fmt.Sprintf("support matrix repeats platform %q", name))
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(platform.Validation) == "" {
			failures = append(failures, fmt.Sprintf("%s is missing validation status", name))
		}
		if !strings.Contains(strings.ToLower(platform.Validation), "not claimed") && !strings.Contains(strings.ToLower(platform.Validation), "fixture") {
			failures = append(failures, fmt.Sprintf("%s validation should state fixture scope or not-claimed live coverage", name))
		}
	}
	return failures
}

func validateSupportMarkdown(matrix supportMatrix, markdown string) []string {
	failures := make([]string, 0)
	for _, platform := range matrix.Platforms {
		if !strings.Contains(markdown, platform.Name) {
			failures = append(failures, fmt.Sprintf("docs/reference/support-matrix.md missing %q from JSON support matrix", platform.Name))
		}
	}
	return failures
}

func validateNamedClaims(label, content string, matrix supportMatrix, names func(supportPlatform) []string) []string {
	failures := make([]string, 0)
	for _, platform := range matrix.Platforms {
		if !containsAny(content, append(names(platform), platform.Name)) {
			failures = append(failures, fmt.Sprintf("%s missing support-matrix platform %q", label, platform.Name))
		}
	}
	return failures
}

func validateSitePlatformList(content string, matrix supportMatrix) []string {
	block := regexp.MustCompile(`(?s)<ul class="platform-list"[^>]*>(.*?)</ul>`).FindStringSubmatch(content)
	if len(block) != 2 {
		return []string{"site/index.html missing platform-list"}
	}
	itemPattern := regexp.MustCompile(`(?s)<li>(.*?)</li>`)
	failures := make([]string, 0)
	for _, match := range itemPattern.FindAllStringSubmatch(block[1], -1) {
		if len(match) != 2 {
			continue
		}
		item := strings.TrimSpace(stripTags(match[1]))
		if item == "" || platformMatches(matrix, item, func(platform supportPlatform) []string { return platform.SiteNames }) {
			continue
		}
		failures = append(failures, fmt.Sprintf("site/index.html platform-list contains unsupported platform %q", item))
	}
	return failures
}

func validateReadmePlatformTable(content string, matrix supportMatrix) []string {
	section := markdownSection(content, "## Platform Coverage")
	if section == "" {
		return []string{"README.md missing Platform Coverage section"}
	}
	failures := make([]string, 0)
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") || strings.Contains(trimmed, "Platform / Integration") {
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 3 {
			continue
		}
		name := strings.TrimSpace(cells[1])
		if name == "" || platformMatches(matrix, name, func(platform supportPlatform) []string { return platform.ReadmeNames }) {
			continue
		}
		failures = append(failures, fmt.Sprintf("README.md Platform Coverage table contains unsupported platform %q", name))
	}
	return failures
}

func platformMatches(matrix supportMatrix, claim string, names func(supportPlatform) []string) bool {
	for _, platform := range matrix.Platforms {
		candidates := append([]string{platform.Name}, names(platform)...)
		for _, candidate := range candidates {
			if strings.EqualFold(strings.TrimSpace(claim), strings.TrimSpace(candidate)) {
				return true
			}
		}
	}
	return false
}

func forbiddenSupportClaims(content string) []string {
	lower := strings.ToLower(content)
	phrases := []string{
		"production proven",
		"production-proven",
		"fully automated migration",
		"fully-automated migration",
		"seamless migration",
		"enterprise ready",
		"enterprise-ready",
		"live-proven",
	}
	found := make([]string, 0)
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase) {
			found = append(found, fmt.Sprintf("%q", phrase))
		}
	}
	return found
}

func containsAny(content string, needles []string) bool {
	normalizedContent := normalizeWhitespace(content)
	for _, needle := range needles {
		trimmed := strings.TrimSpace(needle)
		if trimmed == "" {
			continue
		}
		if strings.Contains(content, trimmed) || strings.Contains(normalizedContent, normalizeWhitespace(trimmed)) {
			return true
		}
	}
	return false
}

func normalizeWhitespace(content string) string {
	return strings.Join(strings.Fields(content), " ")
}

func markdownSection(content, heading string) string {
	start := strings.Index(content, heading)
	if start < 0 {
		return ""
	}
	rest := content[start+len(heading):]
	if next := regexp.MustCompile(`(?m)^## `).FindStringIndex(rest); next != nil {
		return rest[:next[0]]
	}
	return rest
}

func stripTags(content string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, "")
}

func readRepoFile(root, path string) (string, error) {
	payload, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	return string(payload), nil
}
