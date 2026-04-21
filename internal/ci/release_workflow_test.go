package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseAssetManifest_ExplicitCount(t *testing.T) {
	t.Parallel()

	workflowPath := filepath.Join("..", "..", ".github", "workflows", "release.yml")
	payload, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(release workflow) error = %v", err)
	}

	content := string(payload)
	expectedEntries := []string{
		`"dist/viaduct_${PACKAGE_VERSION}_linux_amd64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_linux_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_darwin_arm64.tar.gz"`,
		`"dist/viaduct_${PACKAGE_VERSION}_windows_amd64.tar.gz"`,
	}
	for _, entry := range expectedEntries {
		if !strings.Contains(content, entry) {
			t.Fatalf("release workflow missing expected artifact manifest entry %s", entry)
		}
	}
	if !strings.Contains(content, `expected_count=${#expected_artifacts[@]}`) {
		t.Fatal("release workflow does not compute expected_count from explicit manifest")
	}
	if !strings.Contains(content, `actual_count=${#actual_artifacts[@]}`) {
		t.Fatal("release workflow does not compute actual_count from explicit manifest")
	}
	if !strings.Contains(content, "missing expected release artifact") {
		t.Fatal("release workflow does not emit a named missing-artifact error")
	}
	if !strings.Contains(content, "release artifact count") {
		t.Fatal("release workflow does not emit a named artifact-count error")
	}
}
