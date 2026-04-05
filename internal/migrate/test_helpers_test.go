package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

type mockMigrationConnector struct {
	mu             sync.Mutex
	platform       models.Platform
	result         *models.DiscoveryResult
	connectErr     error
	discoverErr    error
	exportErr      error
	createErr      error
	configureErr   error
	verifyErr      error
	removeErr      error
	snapshotErr    error
	exportedDisks  map[string][]string
	createdVMs     []string
	networkConfigs map[string][]MappedNIC
	poweredOff     []string
	poweredOn      []string
	restored       map[string]models.PowerState
	removed        []string
	snapshots      []string
}

func (m *mockMigrationConnector) Connect(ctx context.Context) error {
	return m.connectErr
}

func (m *mockMigrationConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	return m.result, m.discoverErr
}

func (m *mockMigrationConnector) Platform() models.Platform {
	return m.platform
}

func (m *mockMigrationConnector) Close() error {
	return nil
}

func (m *mockMigrationConnector) ExportVMDisks(ctx context.Context, vm models.VirtualMachine) ([]string, error) {
	if m.exportErr != nil {
		return nil, m.exportErr
	}
	return append([]string(nil), m.exportedDisks[vm.ID]...), nil
}

func (m *mockMigrationConnector) CreateVM(ctx context.Context, vm models.VirtualMachine, convertedDisks []string, targetHost, targetStorage string) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	vmID := vm.ID + "-target"
	m.createdVMs = append(m.createdVMs, vmID)
	return vmID, nil
}

func (m *mockMigrationConnector) ConfigureVMNetworks(ctx context.Context, vmID string, nics []MappedNIC) error {
	if m.configureErr != nil {
		return m.configureErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.networkConfigs == nil {
		m.networkConfigs = make(map[string][]MappedNIC)
	}
	m.networkConfigs[vmID] = append([]MappedNIC(nil), nics...)
	return nil
}

func (m *mockMigrationConnector) VerifyVM(ctx context.Context, vmID string) error {
	return m.verifyErr
}

func (m *mockMigrationConnector) PowerOffVM(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.poweredOff = append(m.poweredOff, vmID)
	return nil
}

func (m *mockMigrationConnector) PowerOnVM(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.poweredOn = append(m.poweredOn, vmID)
	return nil
}

func (m *mockMigrationConnector) RestoreVMPowerState(ctx context.Context, vmID string, state models.PowerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.restored == nil {
		m.restored = make(map[string]models.PowerState)
	}
	m.restored[vmID] = state
	return nil
}

func (m *mockMigrationConnector) RemoveVM(ctx context.Context, vmID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.removed = append(m.removed, vmID)
	return nil
}

func (m *mockMigrationConnector) CreateVMSnapshot(ctx context.Context, vmID string) (string, error) {
	if m.snapshotErr != nil {
		return "", m.snapshotErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	snapshotID := vmID + "-snapshot"
	m.snapshots = append(m.snapshots, snapshotID)
	return snapshotID, nil
}

var _ connectors.Connector = (*mockMigrationConnector)(nil)

func fakeConvertDisk(ctx context.Context, req ConversionRequest) (*ConversionResult, error) {
	if err := os.MkdirAll(filepath.Dir(req.TargetPath), 0o755); err != nil {
		return nil, fmt.Errorf("fake convert: mkdir: %w", err)
	}
	if err := os.WriteFile(req.TargetPath, []byte("converted"), 0o644); err != nil {
		return nil, fmt.Errorf("fake convert: write target: %w", err)
	}

	info, err := os.Stat(req.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("fake convert: stat target: %w", err)
	}

	if req.OnProgress != nil {
		req.OnProgress(100)
	}

	return &ConversionResult{
		SourcePath:      req.SourcePath,
		TargetPath:      req.TargetPath,
		SourceFormat:    req.SourceFormat,
		TargetFormat:    req.TargetFormat,
		SourceSizeBytes: 128,
		TargetSizeBytes: info.Size(),
		Duration:        time.Millisecond,
		SourceChecksum:  "source",
		TargetChecksum:  "target",
	}, nil
}

func sampleVirtualMachines() []models.VirtualMachine {
	return []models.VirtualMachine{
		{
			ID:         "vm-1",
			Name:       "web-01",
			Platform:   models.PlatformVMware,
			PowerState: models.PowerOn,
			CPUCount:   2,
			MemoryMB:   2048,
			Disks: []models.Disk{
				{ID: "disk-1", Name: "web-01.vmdk", SizeMB: 10240, Thin: true, StorageBackend: "datastore1"},
			},
			NICs: []models.NIC{
				{ID: "nic-1", Name: "Network adapter 1", Network: "VM Network", Connected: true},
			},
			Tags: map[string]string{"env": "production"},
		},
		{
			ID:         "vm-2",
			Name:       "web-02",
			Platform:   models.PlatformVMware,
			PowerState: models.PowerOn,
			CPUCount:   2,
			MemoryMB:   2048,
			Disks: []models.Disk{
				{ID: "disk-2", Name: "web-02.vmdk", SizeMB: 10240, Thin: true, StorageBackend: "datastore1"},
			},
			NICs: []models.NIC{
				{ID: "nic-2", Name: "Network adapter 1", Network: "VM Network", Connected: true},
			},
			Tags: map[string]string{"env": "production"},
		},
		{
			ID:         "vm-3",
			Name:       "db-01",
			Platform:   models.PlatformVMware,
			PowerState: models.PowerOff,
			CPUCount:   4,
			MemoryMB:   4096,
			Disks: []models.Disk{
				{ID: "disk-3", Name: "db-01.vmdk", SizeMB: 20480, Thin: false, StorageBackend: "datastore1"},
			},
			NICs: []models.NIC{
				{ID: "nic-3", Name: "Network adapter 1", Network: "Management", Connected: true},
			},
			Tags: map[string]string{"env": "development"},
		},
	}
}

func sampleSourceResult() *models.DiscoveryResult {
	return &models.DiscoveryResult{
		Source:       "vcsa.lab.local",
		Platform:     models.PlatformVMware,
		VMs:          sampleVirtualMachines(),
		DiscoveredAt: time.Date(2026, time.April, 3, 12, 0, 0, 0, time.UTC),
		Networks: []models.NetworkInfo{
			{Name: "VM Network", VlanID: 0},
			{Name: "Management", VlanID: 10},
		},
		Datastores: []models.DatastoreInfo{
			{Name: "datastore1", FreeMB: 500000},
		},
		Hosts: []models.HostInfo{
			{Name: "esx-01", CPUCores: 32, MemoryMB: 131072},
		},
	}
}

func sampleTargetResult() *models.DiscoveryResult {
	return &models.DiscoveryResult{
		Source:       "pve.lab.local",
		Platform:     models.PlatformProxmox,
		DiscoveredAt: time.Date(2026, time.April, 3, 12, 5, 0, 0, time.UTC),
		Networks: []models.NetworkInfo{
			{Name: "vmbr0", VlanID: 0},
			{Name: "vmbr1", VlanID: 10},
		},
		Datastores: []models.DatastoreInfo{
			{Name: "local-lvm", FreeMB: 500000},
			{Name: "ceph-ssd", FreeMB: 750000},
		},
		Hosts: []models.HostInfo{
			{Name: "pve-01", CPUCores: 48, MemoryMB: 262144},
		},
	}
}

func sampleSpec() *MigrationSpec {
	return &MigrationSpec{
		Name: "phase2-test",
		Source: SourceSpec{
			Address:  "vcsa.lab.local",
			Platform: models.PlatformVMware,
		},
		Target: TargetSpec{
			Address:        "pve.lab.local",
			Platform:       models.PlatformProxmox,
			DefaultHost:    "pve-01",
			DefaultStorage: "local-lvm",
		},
		Workloads: []WorkloadSelector{
			{
				Match: MatchCriteria{
					NamePattern: "web-*",
					Exclude:     []string{"web-99"},
				},
				Overrides: WorkloadOverrides{
					TargetHost:    "pve-01",
					TargetStorage: "local-lvm",
					NetworkMap: map[string]string{
						"VM Network": "vmbr0",
						"Management": "vmbr1",
					},
				},
			},
			{
				Match: MatchCriteria{
					Tags: map[string]string{"env": "production"},
				},
				Overrides: WorkloadOverrides{
					NetworkMap: map[string]string{
						"VM Network": "vmbr0",
					},
				},
			},
		},
		Options: MigrationOptions{
			Parallel:       2,
			ShutdownSource: true,
			VerifyBoot:     true,
		},
	}
}
