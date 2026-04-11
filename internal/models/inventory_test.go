package models

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestVirtualMachine_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	discoveredAt := createdAt.Add(5 * time.Minute)

	original := VirtualMachine{
		ID:         "vm-101",
		Name:       "api-01",
		Platform:   PlatformVMware,
		PowerState: PowerOn,
		CPUCount:   8,
		MemoryMB:   16384,
		Disks: []Disk{
			{
				ID:             "disk-1",
				Name:           "root",
				SizeMB:         40960,
				Thin:           true,
				StorageBackend: "vsanDatastore",
			},
		},
		NICs: []NIC{
			{
				ID:          "nic-1",
				Name:        "eth0",
				MACAddress:  "00:50:56:aa:bb:cc",
				Network:     "Production",
				Connected:   true,
				IPAddresses: []string{"10.0.0.10", "fe80::1"},
			},
		},
		GuestOS:      "Ubuntu Linux (64-bit)",
		Host:         "esxi-01",
		Cluster:      "cluster-a",
		ResourcePool: "production",
		Folder:       "/Prod/Web",
		Tags: map[string]string{
			"environment": "prod",
			"owner":       "platform",
		},
		Snapshots: []Snapshot{
			{
				ID:          "snap-1",
				Name:        "before-patch",
				Description: "Pre-maintenance snapshot",
				CreatedAt:   createdAt,
				SizeMB:      1024,
			},
		},
		CreatedAt:    createdAt,
		DiscoveredAt: discoveredAt,
		SourceRef:    "vm-101-ref",
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded VirtualMachine
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("roundtrip mismatch:\nwant: %#v\ngot:  %#v", original, decoded)
	}
}

func TestPlatform_StringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform Platform
		want     string
	}{
		{name: "vmware", platform: PlatformVMware, want: "vmware"},
		{name: "proxmox", platform: PlatformProxmox, want: "proxmox"},
		{name: "hyperv", platform: PlatformHyperV, want: "hyperv"},
		{name: "kvm", platform: PlatformKVM, want: "kvm"},
		{name: "nutanix", platform: PlatformNutanix, want: "nutanix"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.platform) != tt.want {
				t.Fatalf("string(%q) = %q, want %q", tt.name, tt.platform, tt.want)
			}
		})
	}
}

func TestPlatform_Valid_Expected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		platform Platform
		want     bool
	}{
		{name: "vmware", platform: PlatformVMware, want: true},
		{name: "proxmox", platform: PlatformProxmox, want: true},
		{name: "hyperv", platform: PlatformHyperV, want: true},
		{name: "kvm", platform: PlatformKVM, want: true},
		{name: "nutanix", platform: PlatformNutanix, want: true},
		{name: "unknown", platform: Platform("veeam"), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.platform.Valid(); got != tt.want {
				t.Fatalf("Platform(%q).Valid() = %t, want %t", tt.platform, got, tt.want)
			}
		})
	}
}

func TestSupportedPlatforms_StableOrder_Expected(t *testing.T) {
	t.Parallel()

	want := []Platform{
		PlatformHyperV,
		PlatformKVM,
		PlatformNutanix,
		PlatformProxmox,
		PlatformVMware,
	}
	if got := SupportedPlatforms(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedPlatforms() = %#v, want %#v", got, want)
	}
}

func TestPowerState_StringValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state PowerState
		want  string
	}{
		{name: "on", state: PowerOn, want: "on"},
		{name: "off", state: PowerOff, want: "off"},
		{name: "suspended", state: PowerSuspend, want: "suspended"},
		{name: "unknown", state: PowerUnknown, want: "unknown"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.state) != tt.want {
				t.Fatalf("string(%q) = %q, want %q", tt.name, tt.state, tt.want)
			}
		})
	}
}

func TestVirtualMachine_OmitsEmptyOptionals(t *testing.T) {
	t.Parallel()

	vm := VirtualMachine{
		ID:           "vm-202",
		Name:         "db-01",
		Platform:     PlatformProxmox,
		PowerState:   PowerOff,
		CPUCount:     4,
		MemoryMB:     8192,
		Disks:        []Disk{},
		NICs:         []NIC{},
		GuestOS:      "Debian GNU/Linux",
		Host:         "pve-01",
		CreatedAt:    time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Date(2026, time.April, 2, 0, 5, 0, 0, time.UTC),
		SourceRef:    "100",
	}

	payload, err := json.Marshal(vm)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, key := range []string{"cluster", "resource_pool", "folder", "tags", "snapshots"} {
		if _, exists := decoded[key]; exists {
			t.Fatalf("expected key %q to be omitted from JSON", key)
		}
	}
}

func TestNetworkInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := NetworkInfo{
		ID:     "network-1",
		Name:   "Production",
		Type:   "distributed",
		VlanID: 210,
		Switch: "dvSwitch-01",
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded NetworkInfo
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("roundtrip mismatch:\nwant: %#v\ngot:  %#v", original, decoded)
	}
}

func TestDatastoreInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := DatastoreInfo{
		ID:         "datastore-1",
		Name:       "vsanDatastore",
		Type:       "vsan",
		CapacityMB: 524288,
		FreeMB:     262144,
		Hosts:      []string{"esxi-01", "esxi-02"},
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded DatastoreInfo
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("roundtrip mismatch:\nwant: %#v\ngot:  %#v", original, decoded)
	}
}

func TestDiscoveryResult_IncludesInfrastructure(t *testing.T) {
	t.Parallel()

	discoveredAt := time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC)

	original := DiscoveryResult{
		Source:   "lab-vcenter",
		Platform: PlatformVMware,
		VMs: []VirtualMachine{
			{
				ID:           "vm-1",
				Name:         "web-01",
				Platform:     PlatformVMware,
				PowerState:   PowerOn,
				CPUCount:     4,
				MemoryMB:     8192,
				Disks:        []Disk{{ID: "disk-1", Name: "Hard disk 1", SizeMB: 40960, StorageBackend: "vsanDatastore"}},
				NICs:         []NIC{{ID: "nic-1", Name: "Network adapter 1", MACAddress: "00:50:56:aa:bb:cc", Network: "Production", Connected: true}},
				GuestOS:      "Ubuntu Linux (64-bit)",
				Host:         "esxi-01",
				Cluster:      "cluster-a",
				CreatedAt:    discoveredAt.Add(-24 * time.Hour),
				DiscoveredAt: discoveredAt,
				SourceRef:    "vm-1",
			},
		},
		Networks: []NetworkInfo{
			{ID: "network-1", Name: "Production", Type: "distributed", VlanID: 100, Switch: "dvSwitch-01"},
		},
		Datastores: []DatastoreInfo{
			{ID: "datastore-1", Name: "vsanDatastore", Type: "vsan", CapacityMB: 524288, FreeMB: 262144, Hosts: []string{"esxi-01"}},
		},
		Hosts: []HostInfo{
			{ID: "host-1", Name: "esxi-01", Cluster: "cluster-a", CPUCores: 16, MemoryMB: 131072, PowerState: PowerOn, ConnectionState: "connected"},
		},
		Clusters: []ClusterInfo{
			{ID: "domain-c1", Name: "cluster-a", Hosts: []string{"esxi-01"}, TotalCPUCores: 16, TotalMemoryMB: 131072, HAEnabled: true, DRSEnabled: true},
		},
		ResourcePools: []ResourcePoolInfo{
			{ID: "resgroup-1", Name: "production", Cluster: "cluster-a", CPULimitMHz: -1, MemoryLimitMB: -1},
		},
		DiscoveredAt: discoveredAt,
		Duration:     3 * time.Second,
		Errors:       []string{"warning: partial tag data unavailable"},
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded DiscoveryResult
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("roundtrip mismatch:\nwant: %#v\ngot:  %#v", original, decoded)
	}
}
