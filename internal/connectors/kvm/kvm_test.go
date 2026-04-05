package kvm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestParseDomainXML_LinuxDomain_Expected(t *testing.T) {
	t.Parallel()

	payload := readKVMFixture(t, "domain_linux.xml")
	domain, err := parseDomainXML(payload)
	if err != nil {
		t.Fatalf("parseDomainXML() error = %v", err)
	}

	vm := mapDomainToVM(domain, models.PowerOn, "kvm-host-01")
	if vm.Name != "ubuntu-web-01" || vm.CPUCount != 4 || len(vm.NICs) != 1 {
		t.Fatalf("unexpected VM: %#v", vm)
	}
}

func TestParseDomainXML_WindowsDomain_Expected(t *testing.T) {
	t.Parallel()

	payload := readKVMFixture(t, "domain_windows.xml")
	domain, err := parseDomainXML(payload)
	if err != nil {
		t.Fatalf("parseDomainXML() error = %v", err)
	}

	vm := mapDomainToVM(domain, models.PowerOff, "kvm-host-02")
	if vm.Name != "windows-app-01" || vm.MemoryMB != 8192 || len(vm.Disks) != 1 {
		t.Fatalf("unexpected VM: %#v", vm)
	}
}

func TestMapKVMPowerState_Values_Expected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		state string
		want  models.PowerState
	}{
		{name: "running", state: "running", want: models.PowerOn},
		{name: "paused", state: "paused", want: models.PowerSuspend},
		{name: "shutoff", state: "shutoff", want: models.PowerOff},
		{name: "unknown", state: "nostate", want: models.PowerUnknown},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := mapKVMPowerStateName(tc.state); got != tc.want {
				t.Fatalf("mapKVMPowerStateName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestKVMConnector_Platform_Expected(t *testing.T) {
	t.Parallel()

	connector := NewKVMConnector(connectors.Config{})
	if got := connector.Platform(); got != models.PlatformKVM {
		t.Fatalf("Platform() = %q, want %q", got, models.PlatformKVM)
	}
}

func TestParseStoragePoolXML_File_Expected(t *testing.T) {
	t.Parallel()

	payload := readKVMFixture(t, "storagepool.xml")
	pool, err := parseStoragePoolXML(payload)
	if err != nil {
		t.Fatalf("parseStoragePoolXML() error = %v", err)
	}
	if pool.Name != "default" || pool.Type != "dir" {
		t.Fatalf("unexpected pool: %#v", pool)
	}
}

func readKVMFixture(t *testing.T, name string) string {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}
	return string(payload)
}
