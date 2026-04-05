package vmware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/vmware/govmomi/vim25/types"
)

func TestMapPowerState_SupportedStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input types.VirtualMachinePowerState
		want  models.PowerState
	}{
		{name: "powered on", input: types.VirtualMachinePowerStatePoweredOn, want: models.PowerOn},
		{name: "powered off", input: types.VirtualMachinePowerStatePoweredOff, want: models.PowerOff},
		{name: "suspended", input: types.VirtualMachinePowerStateSuspended, want: models.PowerSuspend},
		{name: "unknown", input: "unknown", want: models.PowerUnknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mapPowerState(tt.input); got != tt.want {
				t.Fatalf("mapPowerState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapDisks_VirtualDisk_ReturnsNormalizedDisks(t *testing.T) {
	t.Parallel()

	devices := []types.BaseVirtualDevice{
		&types.VirtualDisk{
			VirtualDevice: types.VirtualDevice{
				Key:        2000,
				DeviceInfo: &types.Description{Label: "Hard disk 1"},
				Backing: &types.VirtualDiskFlatVer2BackingInfo{
					VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{FileName: "[vsanDatastore] web-01/web-01.vmdk"},
					ThinProvisioned:              types.NewBool(true),
				},
			},
			CapacityInKB: 32 * 1024,
		},
	}

	disks := mapDisks(devices)
	if len(disks) != 1 {
		t.Fatalf("len(disks) = %d, want 1", len(disks))
	}

	if disks[0].SizeMB != 32 {
		t.Fatalf("SizeMB = %d, want 32", disks[0].SizeMB)
	}
}

func TestMapNICs_VirtualEthernetCard_ReturnsNormalizedNICs(t *testing.T) {
	t.Parallel()

	devices := []types.BaseVirtualDevice{
		&types.VirtualVmxnet3{
			VirtualVmxnet: types.VirtualVmxnet{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Key:        4000,
						DeviceInfo: &types.Description{Label: "Network adapter 1"},
						Backing: &types.VirtualEthernetCardNetworkBackingInfo{
							VirtualDeviceDeviceBackingInfo: types.VirtualDeviceDeviceBackingInfo{DeviceName: "Production"},
						},
						Connectable: &types.VirtualDeviceConnectInfo{Connected: true},
					},
					MacAddress: "00:50:56:AA:BB:CC",
				},
			},
		},
	}

	guestNics := []types.GuestNicInfo{
		{
			MacAddress: "00:50:56:AA:BB:CC",
			Network:    "Production",
			IpAddress:  []string{"10.0.0.10"},
		},
	}

	nics := mapNICs(devices, guestNics)
	if len(nics) != 1 {
		t.Fatalf("len(nics) = %d, want 1", len(nics))
	}

	if nics[0].MACAddress != "00:50:56:AA:BB:CC" {
		t.Fatalf("MACAddress = %q, want %q", nics[0].MACAddress, "00:50:56:AA:BB:CC")
	}
}

func TestVMwareConnector_ConnectFailure_UnreachableAddressReturnsWrappedError(t *testing.T) {
	t.Parallel()

	connector := NewVMwareConnector(connectors.Config{
		Address:  "https://127.0.0.1:1/sdk",
		Username: "user",
		Password: "pass",
		Insecure: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := connector.Connect(ctx)
	if err == nil {
		t.Fatal("Connect() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "vmware: connect") {
		t.Fatalf("Connect() error = %q, want wrapped vmware connect error", err.Error())
	}
}

func TestVMwareConnector_Platform_ReturnsVMware(t *testing.T) {
	t.Parallel()

	connector := NewVMwareConnector(connectors.Config{})
	if got := connector.Platform(); got != models.PlatformVMware {
		t.Fatalf("Platform() = %q, want %q", got, models.PlatformVMware)
	}
}
