package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckReleaseSurfaces_PassingFixture_Expected(t *testing.T) {
	root := writeReleaseSurfaceFixture(t, "3.2.1")

	if failures := checkReleaseSurfaces(root, "3.2.1"); len(failures) != 0 {
		t.Fatalf("checkReleaseSurfaces() failures = %#v, want none", failures)
	}
}

func TestCheckReleaseSurfaces_StaleVersionFails_Expected(t *testing.T) {
	root := writeReleaseSurfaceFixture(t, "3.2.1")
	appendFixtureFile(t, root, "README.md", "\nghcr.io/eblackrps/viaduct:3.2.0\n")

	failures := checkReleaseSurfaces(root, "3.2.1")
	if !failureContains(failures, "README.md has stale active version reference") {
		t.Fatalf("checkReleaseSurfaces() failures = %#v, want stale README version failure", failures)
	}
}

func TestCheckReleaseSurfaces_FutureReleaseWordingFails_Expected(t *testing.T) {
	root := writeReleaseSurfaceFixture(t, "3.2.1")
	appendFixtureFile(t, root, filepath.Join("docs", "releases", "current.md"), "\nafter the tag workflow publishes\n")

	failures := checkReleaseSurfaces(root, "3.2.1")
	if !failureContains(failures, "tag workflow publishes") {
		t.Fatalf("checkReleaseSurfaces() failures = %#v, want future wording failure", failures)
	}
}

func TestCheckReleaseSurfaces_ForbiddenOverclaimFails_Expected(t *testing.T) {
	root := writeReleaseSurfaceFixture(t, "3.2.1")
	appendFixtureFile(t, root, filepath.Join("site", "index.html"), "\nenterprise ready\n")

	failures := checkReleaseSurfaces(root, "3.2.1")
	if !failureContains(failures, "enterprise ready") {
		t.Fatalf("checkReleaseSurfaces() failures = %#v, want forbidden overclaim failure", failures)
	}
}

func writeReleaseSurfaceFixture(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, filepath.Join("web", "package.json"), `{"version": "`+version+`"}`+"\n")

	expectations, _, _ := releaseSurfaceExpectations(version)
	for _, expectation := range expectations {
		if expectation.Path == filepath.Join("web", "package.json") {
			continue
		}
		writeFixtureFile(t, root, expectation.Path, strings.Join(expectation.Needles, "\n")+"\n")
	}
	return root
}

func writeFixtureFile(t *testing.T, root, path, content string) {
	t.Helper()
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", target, err)
	}
}

func appendFixtureFile(t *testing.T, root, path, content string) {
	t.Helper()
	target := filepath.Join(root, path)
	file, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(%s) error = %v", target, err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("WriteString(%s) error = %v", target, err)
	}
}

func failureContains(failures []string, needle string) bool {
	for _, failure := range failures {
		if strings.Contains(failure, needle) {
			return true
		}
	}
	return false
}
