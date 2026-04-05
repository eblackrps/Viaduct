package hyperv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestMapHyperVPowerState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  models.PowerState
	}{
		{name: "running", input: "Running", want: models.PowerOn},
		{name: "off", input: "Off", want: models.PowerOff},
		{name: "saved", input: "Saved", want: models.PowerSuspend},
		{name: "other", input: "Unknown", want: models.PowerUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mapHyperVPowerState(tt.input); got != tt.want {
				t.Fatalf("mapHyperVPowerState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapHyperVDisks(t *testing.T) {
	t.Parallel()

	var disks []hyperVDisk
	readFixture(t, "hyperv_disks.json", &disks)
	mapped := mapHyperVDisks(disks)
	if len(mapped) != 2 {
		t.Fatalf("len(mapHyperVDisks()) = %d, want 2", len(mapped))
	}
}

func TestDecodeVMs(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile(filepath.Join("testdata", "hyperv_vms.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	vms, err := decodeVMs(string(payload))
	if err != nil {
		t.Fatalf("decodeVMs() error = %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("len(decodeVMs()) = %d, want 2", len(vms))
	}
}

func TestMapHyperVNICs(t *testing.T) {
	t.Parallel()

	var adapters []hyperVAdapter
	readFixture(t, "hyperv_nics.json", &adapters)
	mapped := mapHyperVNICs(adapters)
	if len(mapped) != 2 {
		t.Fatalf("len(mapHyperVNICs()) = %d, want 2", len(mapped))
	}
}

func TestHyperVConnector_Platform(t *testing.T) {
	t.Parallel()

	connector := NewHyperVConnector(connectors.Config{})
	if connector.Platform() != models.PlatformHyperV {
		t.Fatalf("Platform() = %q, want %q", connector.Platform(), models.PlatformHyperV)
	}
}

func readFixture[T any](t *testing.T, name string, out *T) {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}

	if err := json.Unmarshal(payload, out); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", name, err)
	}
}
