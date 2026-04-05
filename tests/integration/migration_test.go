package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

type mockIntegrationConnector struct {
	mu               sync.Mutex
	platform         models.Platform
	result           *models.DiscoveryResult
	verifyErr        error
	exportedDisks    map[string][]string
	createVMOverride func(context.Context, models.VirtualMachine, []string, string, string) (string, error)
	createdVMs       []string
	networkConfigs   map[string][]migratepkg.MappedNIC
	restored         map[string]models.PowerState
	removed          []string
}

func (m *mockIntegrationConnector) Connect(ctx context.Context) error { return nil }
func (m *mockIntegrationConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	return m.result, nil
}
func (m *mockIntegrationConnector) Platform() models.Platform { return m.platform }
func (m *mockIntegrationConnector) Close() error              { return nil }
func (m *mockIntegrationConnector) ExportVMDisks(ctx context.Context, vm models.VirtualMachine) ([]string, error) {
	return append([]string(nil), m.exportedDisks[vm.ID]...), nil
}
func (m *mockIntegrationConnector) CreateVM(ctx context.Context, vm models.VirtualMachine, convertedDisks []string, targetHost, targetStorage string) (string, error) {
	if m.createVMOverride != nil {
		return m.createVMOverride(ctx, vm, convertedDisks, targetHost, targetStorage)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	vmID := vm.ID + "-target"
	m.createdVMs = append(m.createdVMs, vmID)
	return vmID, nil
}
func (m *mockIntegrationConnector) ConfigureVMNetworks(ctx context.Context, vmID string, nics []migratepkg.MappedNIC) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.networkConfigs == nil {
		m.networkConfigs = make(map[string][]migratepkg.MappedNIC)
	}
	m.networkConfigs[vmID] = append([]migratepkg.MappedNIC(nil), nics...)
	return nil
}
func (m *mockIntegrationConnector) VerifyVM(ctx context.Context, vmID string) error {
	return m.verifyErr
}
func (m *mockIntegrationConnector) PowerOffVM(ctx context.Context, vmID string) error { return nil }
func (m *mockIntegrationConnector) PowerOnVM(ctx context.Context, vmID string) error  { return nil }
func (m *mockIntegrationConnector) RestoreVMPowerState(ctx context.Context, vmID string, state models.PowerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.restored == nil {
		m.restored = make(map[string]models.PowerState)
	}
	m.restored[vmID] = state
	return nil
}
func (m *mockIntegrationConnector) RemoveVM(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removed = append(m.removed, vmID)
	return nil
}
func (m *mockIntegrationConnector) CreateVMSnapshot(ctx context.Context, vmID string) (string, error) {
	return vmID + "-snapshot", nil
}

var _ connectors.Connector = (*mockIntegrationConnector)(nil)

func TestMigration_EndToEnd_Success(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)

	source := &mockIntegrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: createExportedDisks(t, sourceResult.VMs[:3]),
	}
	target := &mockIntegrationConnector{
		platform: targetResult.Platform,
		result:   targetResult,
	}
	stateStore := store.NewMemoryStore()

	report, err := migratepkg.NewPreflightChecker(source, target, spec).RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if !report.CanProceed {
		t.Fatalf("CanProceed = false, want true: %#v", report)
	}

	orchestrator := migratepkg.NewOrchestrator(source, target, stateStore, nil)
	orchestrator.SetIDGenerator(func() string { return "migration-success" })
	setFakeConverter(orchestrator)

	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Phase != migratepkg.PhaseComplete {
		t.Fatalf("Phase = %s, want %s", state.Phase, migratepkg.PhaseComplete)
	}
	if len(state.Workloads) != 3 {
		t.Fatalf("len(Workloads) = %d, want 3", len(state.Workloads))
	}
	for _, workload := range state.Workloads {
		if workload.Phase != migratepkg.PhaseComplete {
			t.Fatalf("workload phase = %s, want %s", workload.Phase, migratepkg.PhaseComplete)
		}
	}

	if _, err := stateStore.GetMigration(context.Background(), store.DefaultTenantID, "migration-success"); err != nil {
		t.Fatalf("GetMigration() error = %v", err)
	}
}

func TestMigration_EndToEnd_DryRun(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)
	spec.Options.DryRun = true

	source := &mockIntegrationConnector{platform: sourceResult.Platform, result: sourceResult}
	target := &mockIntegrationConnector{platform: targetResult.Platform, result: targetResult}

	orchestrator := migratepkg.NewOrchestrator(source, target, store.NewMemoryStore(), nil)
	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Phase != migratepkg.PhasePlan {
		t.Fatalf("Phase = %s, want %s", state.Phase, migratepkg.PhasePlan)
	}
	if len(target.createdVMs) != 0 {
		t.Fatalf("created VMs = %d, want 0", len(target.createdVMs))
	}
}

func TestMigration_EndToEnd_Rollback(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)

	source := &mockIntegrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: createExportedDisks(t, sourceResult.VMs[:3]),
	}
	target := &mockIntegrationConnector{
		platform: targetResult.Platform,
		result:   targetResult,
	}
	stateStore := store.NewMemoryStore()

	orchestrator := migratepkg.NewOrchestrator(source, target, stateStore, nil)
	orchestrator.SetIDGenerator(func() string { return "migration-rollback" })
	setFakeConverter(orchestrator)

	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	result, err := migratepkg.NewRollbackManager(stateStore, source, target).Rollback(context.Background(), state.ID)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if result.TargetVMsRemoved != 3 {
		t.Fatalf("TargetVMsRemoved = %d, want 3", result.TargetVMsRemoved)
	}
	if len(source.restored) != 3 {
		t.Fatalf("restored VMs = %d, want 3", len(source.restored))
	}

	record, err := stateStore.GetMigration(context.Background(), store.DefaultTenantID, state.ID)
	if err != nil {
		t.Fatalf("GetMigration() error = %v", err)
	}

	var persisted migratepkg.MigrationState
	if err := json.Unmarshal(record.RawJSON, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if persisted.Phase != migratepkg.PhaseRolledBack {
		t.Fatalf("Phase = %s, want %s", persisted.Phase, migratepkg.PhaseRolledBack)
	}
}

func TestMigration_PreflightFailure(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	targetResult.Datastores[0].FreeMB = 1
	targetResult.Datastores[1].FreeMB = 1

	report, err := migratepkg.NewPreflightChecker(
		&mockIntegrationConnector{platform: sourceResult.Platform, result: sourceResult},
		&mockIntegrationConnector{platform: targetResult.Platform, result: targetResult},
		loadSpecFixture(t),
	).RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestMigration_NetworkRemapping(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)

	source := &mockIntegrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: createExportedDisks(t, sourceResult.VMs[:3]),
	}
	target := &mockIntegrationConnector{
		platform: targetResult.Platform,
		result:   targetResult,
	}

	orchestrator := migratepkg.NewOrchestrator(source, target, store.NewMemoryStore(), nil)
	orchestrator.SetIDGenerator(func() string { return "migration-network" })
	setFakeConverter(orchestrator)

	if _, err := orchestrator.Execute(context.Background(), spec); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for vmID, nics := range target.networkConfigs {
		if len(nics) == 0 {
			t.Fatalf("%s mapped NIC count = 0, want > 0", vmID)
		}
		for _, nic := range nics {
			if nic.Original.Network == "VM Network" && nic.TargetNetwork != "vmbr0" {
				t.Fatalf("TargetNetwork = %q, want vmbr0", nic.TargetNetwork)
			}
			if nic.Original.Network == "Management" && nic.TargetNetwork != "vmbr1" {
				t.Fatalf("TargetNetwork = %q, want vmbr1", nic.TargetNetwork)
			}
		}
	}
}

func TestMigration_Preflight_ScheduledWindowBlocked(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)
	spec.Options.Window = migratepkg.ExecutionWindow{
		NotBefore: time.Now().UTC().Add(2 * time.Hour),
	}

	report, err := migratepkg.NewPreflightChecker(
		&mockIntegrationConnector{platform: sourceResult.Platform, result: sourceResult},
		&mockIntegrationConnector{platform: targetResult.Platform, result: targetResult},
		spec,
	).RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestMigration_Resume_AfterImportFailure(t *testing.T) {
	t.Parallel()

	sourceResult := loadInventoryFixture(t, "mock_source_inventory.json")
	targetResult := loadInventoryFixture(t, "mock_target_inventory.json")
	spec := loadSpecFixture(t)

	source := &mockIntegrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: createExportedDisks(t, sourceResult.VMs[:3]),
	}
	target := &mockIntegrationConnector{
		platform: targetResult.Platform,
		result:   targetResult,
	}
	stateStore := store.NewMemoryStore()

	orchestrator := migratepkg.NewOrchestrator(source, target, stateStore, nil)
	orchestrator.SetIDGenerator(func() string { return "migration-resume" })
	setFakeConverter(orchestrator)

	targetCreateFail := true
	target.createVMOverride = func(ctx context.Context, vm models.VirtualMachine, convertedDisks []string, targetHost, targetStorage string) (string, error) {
		if targetCreateFail {
			return "", fmt.Errorf("temporary target create failure")
		}
		vmID := vm.ID + "-target"
		target.mu.Lock()
		defer target.mu.Unlock()
		target.createdVMs = append(target.createdVMs, vmID)
		return vmID, nil
	}

	state, err := orchestrator.Execute(context.Background(), spec)
	if err == nil {
		t.Fatal("Execute() error = nil, want import failure")
	}
	if state == nil {
		t.Fatal("Execute() state = nil, want failed state")
	}

	targetCreateFail = false
	resumed, err := orchestrator.Resume(context.Background(), state.ID, spec)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if resumed.Phase != migratepkg.PhaseComplete {
		t.Fatalf("Phase = %s, want %s", resumed.Phase, migratepkg.PhaseComplete)
	}
}

func loadSpecFixture(t *testing.T) *migratepkg.MigrationSpec {
	t.Helper()
	spec, err := migratepkg.ParseSpec(filepath.Join("testdata", "test_migration_spec.yaml"))
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}
	return spec
}

func loadInventoryFixture(t *testing.T, name string) *models.DiscoveryResult {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}

	var result models.DiscoveryResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", name, err)
	}

	return &result
}

func createExportedDisks(t *testing.T, vms []models.VirtualMachine) map[string][]string {
	t.Helper()

	paths := make(map[string][]string, len(vms))
	for _, vm := range vms {
		sourcePath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.vmdk", vm.ID))
		if err := os.WriteFile(sourcePath, []byte("source"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		paths[vm.ID] = []string{sourcePath}
	}
	return paths
}

func setFakeConverter(orchestrator *migratepkg.Orchestrator) {
	orchestrator.SetDiskConverter(func(ctx context.Context, req migratepkg.ConversionRequest) (*migratepkg.ConversionResult, error) {
		if err := os.MkdirAll(filepath.Dir(req.TargetPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(req.TargetPath, []byte("converted"), 0o644); err != nil {
			return nil, err
		}
		info, err := os.Stat(req.TargetPath)
		if err != nil {
			return nil, err
		}
		return &migratepkg.ConversionResult{
			SourcePath:      req.SourcePath,
			TargetPath:      req.TargetPath,
			SourceFormat:    req.SourceFormat,
			TargetFormat:    req.TargetFormat,
			SourceSizeBytes: 128,
			TargetSizeBytes: info.Size(),
		}, nil
	})
}
