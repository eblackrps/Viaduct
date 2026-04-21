package ci

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestReleaseWorkflow_TagPinnedCertificateIdentity_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadReleaseWorkflow(t)
	dispatchStep := workflow.stepNamed(t, "dispatch-identity-dry-run", "Verify Dispatch Tag Identity Shape")
	if !strings.Contains(dispatchStep.Run, `@refs/tags/${RELEASE_TAG}`) {
		t.Fatalf("dispatch identity step run = %q, want refs/tags identity pin", dispatchStep.Run)
	}
	if strings.Contains(dispatchStep.Run, `@refs/heads/${GITHUB_REF_NAME}`) {
		t.Fatalf("dispatch identity step still references branch heads: %q", dispatchStep.Run)
	}

	releaseMetaStep := workflow.stepNamed(t, "release", "Derive Release Metadata")
	if !strings.Contains(releaseMetaStep.Run, `CERTIFICATE_IDENTITY_REF="refs/tags/${RELEASE_TAG}"`) {
		t.Fatalf("release metadata step run = %q, want tag-pinned certificate identity ref", releaseMetaStep.Run)
	}
	if strings.Contains(releaseMetaStep.Run, `CERTIFICATE_IDENTITY_REF="${GITHUB_REF}"`) {
		t.Fatalf("release metadata step should not derive certificate identity from GITHUB_REF: %q", releaseMetaStep.Run)
	}
}

func TestReleaseAssetManifest_ExplicitCount(t *testing.T) {
	t.Parallel()

	workflow := loadReleaseWorkflow(t)
	step := workflow.stepNamed(t, "release", "Verify Expected Release Artifacts")
	artifactPattern := regexp.MustCompile(`"dist/viaduct_\$\{PACKAGE_VERSION\}_[^"]+\.(?:tar\.gz|zip)"`)
	artifacts := artifactPattern.FindAllString(step.Run, -1)
	expectedArtifacts := expectedReleaseArtifacts()
	if !slices.Equal(artifacts, expectedArtifacts) {
		t.Fatalf("artifact manifest = %#v, want %#v", artifacts, expectedArtifacts)
	}
	if !strings.Contains(step.Run, `expected_count=${#expected_artifacts[@]}`) {
		t.Fatal("release workflow does not compute expected_count from explicit manifest")
	}
	if !strings.Contains(step.Run, `actual_count=${#actual_artifacts[@]}`) {
		t.Fatal("release workflow does not compute actual_count from explicit manifest")
	}
	if !strings.Contains(step.Run, "missing expected release artifact") {
		t.Fatal("release workflow does not emit a named missing-artifact error")
	}
	if !strings.Contains(step.Run, "release artifact count") {
		t.Fatal("release workflow does not emit a named artifact-count error")
	}
}

func TestReleaseWorkflow_BinaryPathVerificationMatchesPackageMatrix_Expected(t *testing.T) {
	t.Parallel()

	workflow := loadReleaseWorkflow(t)
	step := workflow.stepNamed(t, "release", "Verify Published Bundle Checksums Before Docker Build")
	for _, binaryPath := range expectedPackageMatrixBinaryPaths(t) {
		if !strings.Contains(step.Run, "sha256sum "+binaryPath) {
			t.Fatalf("release verification step run = %q, want sha256sum for %s", step.Run, binaryPath)
		}
		if !strings.Contains(step.Run, "diff <(sha256sum "+binaryPath+" | cut -d' ' -f1)") {
			t.Fatalf("release verification step run = %q, want checksum diff for %s", step.Run, binaryPath)
		}
	}
}

func loadReleaseWorkflow(t *testing.T) workflowDefinition {
	t.Helper()
	return loadWorkflow(t, "release.yml")
}

func expectedReleaseArtifacts() []string {
	return []string{
		`"dist/viaduct_${PACKAGE_VERSION}_linux_amd64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_linux_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_darwin_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_windows_amd64.tar.gz"`,
	}
}

func expectedPackageMatrixBinaryPaths(t *testing.T) []string {
	t.Helper()

	makefilePath := filepath.Join("..", "..", "Makefile")
	payload, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	content := string(payload)
	return []string{
		extractMakeVariable(t, content, "LINUX_AMD64_BINARY"),
		extractMakeVariable(t, content, "LINUX_ARM64_BINARY"),
	}
}

func extractMakeVariable(t *testing.T, content, name string) string {
	t.Helper()

	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + ` = ([^\r\n]+)$`)
	match := pattern.FindStringSubmatch(content)
	if len(match) != 2 {
		t.Fatalf("Makefile missing variable %s", name)
	}
	return strings.TrimSpace(match[1])
}
