package proxmox

import (
	"context"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestMapProxmoxPowerState_KnownStatesReturnMappedValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  models.PowerState
	}{
		{name: "running", input: "running", want: models.PowerOn},
		{name: "stopped", input: "stopped", want: models.PowerOff},
		{name: "paused", input: "paused", want: models.PowerSuspend},
		{name: "unknown", input: "mystery", want: models.PowerUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapProxmoxPowerState(tt.input); got != tt.want {
				t.Fatalf("mapProxmoxPowerState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseProxmoxDisks_ScsiDisk_ReturnsDisk(t *testing.T) {
	t.Parallel()

	disks := parseProxmoxDisks(map[string]interface{}{
		"scsi0": "local-lvm:vm-100-disk-0,size=32G",
	})

	if len(disks) != 1 {
		t.Fatalf("len(disks) = %d, want 1", len(disks))
	}

	if disks[0].SizeMB != 32768 {
		t.Fatalf("SizeMB = %d, want 32768", disks[0].SizeMB)
	}
}

func TestParseProxmoxNICs_NetConfig_ReturnsNIC(t *testing.T) {
	t.Parallel()

	nics := parseProxmoxNICs(map[string]interface{}{
		"net0": "virtio=AA:BB:CC:DD:EE:FF,bridge=vmbr0",
	})

	if len(nics) != 1 {
		t.Fatalf("len(nics) = %d, want 1", len(nics))
	}

	if nics[0].MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("MACAddress = %q, want %q", nics[0].MACAddress, "AA:BB:CC:DD:EE:FF")
	}
}

func TestProxmoxConnector_Platform_ReturnsProxmox(t *testing.T) {
	t.Parallel()

	connector := NewProxmoxConnector(connectors.Config{})
	if got := connector.Platform(); got != models.PlatformProxmox {
		t.Fatalf("Platform() = %q, want %q", got, models.PlatformProxmox)
	}
}

func TestProxmoxConnector_DiscoverBeforeConnect_ReturnsError(t *testing.T) {
	t.Parallel()

	connector := NewProxmoxConnector(connectors.Config{})
	_, err := connector.Discover(context.Background())
	if err == nil {
		t.Fatal("Discover() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("Discover() error = %q, want substring %q", err.Error(), "not connected")
	}
}
