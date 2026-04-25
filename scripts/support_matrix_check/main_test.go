package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSupportMatrix_PassingFixture_Expected(t *testing.T) {
	root := writeSupportMatrixFixture(t)

	if failures := checkSupportMatrix(root); len(failures) != 0 {
		t.Fatalf("checkSupportMatrix() failures = %#v, want none", failures)
	}
}

func TestCheckSupportMatrix_UnsupportedSitePlatformFails_Expected(t *testing.T) {
	root := writeSupportMatrixFixture(t)
	writeSupportFixtureFile(t, root, filepath.Join("site", "index.html"), `<ul class="platform-list"><li>VMware vSphere</li><li>Proxmox VE</li><li>OpenStack</li></ul>`)

	failures := checkSupportMatrix(root)
	if !supportFailureContains(failures, `unsupported platform "OpenStack"`) {
		t.Fatalf("checkSupportMatrix() failures = %#v, want unsupported platform failure", failures)
	}
}

func TestCheckSupportMatrix_ForbiddenClaimFails_Expected(t *testing.T) {
	root := writeSupportMatrixFixture(t)
	appendSupportFixtureFile(t, root, "README.md", "\nproduction proven\n")

	failures := checkSupportMatrix(root)
	if !supportFailureContains(failures, "production proven") {
		t.Fatalf("checkSupportMatrix() failures = %#v, want forbidden claim failure", failures)
	}
}

func writeSupportMatrixFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	matrix := `{
  "platforms": [
    {"name":"VMware vSphere","site_names":["VMware vSphere"],"readme_names":["VMware vSphere"],"validation":"implemented; fixture tested; live lab not claimed"},
    {"name":"Proxmox VE","site_names":["Proxmox VE"],"readme_names":["Proxmox VE"],"validation":"implemented; fixture tested; live lab not claimed"}
  ]
}`
	writeSupportFixtureFile(t, root, filepath.Join("docs", "reference", "support-matrix.json"), matrix)
	writeSupportFixtureFile(t, root, filepath.Join("docs", "reference", "support-matrix.md"), "VMware vSphere\nProxmox VE\n")
	writeSupportFixtureFile(t, root, filepath.Join("site", "index.html"), `<ul class="platform-list"><li>VMware vSphere</li><li>Proxmox VE</li></ul>`)
	writeSupportFixtureFile(t, root, "README.md", "## Platform Coverage\n\n| Platform / Integration | Status |\n| --- | --- |\n| VMware vSphere | Implemented |\n| Proxmox VE | Implemented |\n\n## Next\n")
	for _, path := range []string{
		"INSTALL.md",
		"QUICKSTART.md",
		filepath.Join("docs", "architecture.md"),
		filepath.Join("docs", "getting-started", "installation.md"),
		filepath.Join("docs", "getting-started", "quickstart.md"),
	} {
		writeSupportFixtureFile(t, root, path, "plain support text\n")
	}
	return root
}

func writeSupportFixtureFile(t *testing.T, root, path, content string) {
	t.Helper()
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", target, err)
	}
}

func appendSupportFixtureFile(t *testing.T, root, path, content string) {
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

func supportFailureContains(failures []string, needle string) bool {
	for _, failure := range failures {
		if strings.Contains(failure, needle) {
			return true
		}
	}
	return false
}
