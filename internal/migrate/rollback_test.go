package migrate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestCreateRecoveryPoint(t *testing.T) {
	t.Parallel()

	manager := NewRollbackManager(store.NewMemoryStore(), &mockMigrationConnector{}, &mockMigrationConnector{})
	state := &MigrationState{
		ID:       "mig-001",
		SpecName: "phase2-test",
		Phase:    PhasePlan,
		Workloads: []WorkloadMigration{
			{VM: sampleVirtualMachines()[0]},
		},
	}

	point, err := manager.CreateRecoveryPoint(context.Background(), state)
	if err != nil {
		t.Fatalf("CreateRecoveryPoint() error = %v", err)
	}
	if point.MigrationID != "mig-001" || len(point.SourceVMs) != 1 {
		t.Fatalf("unexpected recovery point: %#v", point)
	}
}

func TestRollback_FullMigration(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	source := &mockMigrationConnector{}
	target := &mockMigrationConnector{}
	manager := NewRollbackManager(stateStore, source, target)

	converted := filepath.Join(t.TempDir(), "disk.qcow2")
	if err := os.WriteFile(converted, []byte("converted"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state := &MigrationState{
		ID:       "mig-full",
		SpecName: "phase2-test",
		Phase:    PhaseComplete,
		Workloads: []WorkloadMigration{
			{
				VM:                 sampleVirtualMachines()[0],
				TargetVMID:         "vm-1-target",
				ConvertedDiskPaths: []string{converted},
			},
		},
		StartedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := persistMigrationState(context.Background(), stateStore, state); err != nil {
		t.Fatalf("persistMigrationState() error = %v", err)
	}
	if _, err := manager.CreateRecoveryPoint(context.Background(), state); err != nil {
		t.Fatalf("CreateRecoveryPoint() error = %v", err)
	}

	result, err := manager.Rollback(context.Background(), "mig-full")
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if result.TargetVMsRemoved != 1 || result.SourceVMsRestored != 1 {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
}

func TestRollback_PartialMigration(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	manager := NewRollbackManager(stateStore, &mockMigrationConnector{}, &mockMigrationConnector{})

	converted := filepath.Join(t.TempDir(), "partial.qcow2")
	if err := os.WriteFile(converted, []byte("converted"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state := &MigrationState{
		ID:       "mig-partial",
		SpecName: "phase2-test",
		Phase:    PhaseFailed,
		Workloads: []WorkloadMigration{
			{
				VM:                 sampleVirtualMachines()[0],
				ConvertedDiskPaths: []string{converted},
			},
		},
		StartedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := persistMigrationState(context.Background(), stateStore, state); err != nil {
		t.Fatalf("persistMigrationState() error = %v", err)
	}
	if _, err := manager.CreateRecoveryPoint(context.Background(), state); err != nil {
		t.Fatalf("CreateRecoveryPoint() error = %v", err)
	}

	result, err := manager.Rollback(context.Background(), "mig-partial")
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if result.FilesCleanedUp != 1 {
		t.Fatalf("FilesCleanedUp = %d, want 1", result.FilesCleanedUp)
	}
}

func TestRollback_Idempotent(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	source := &mockMigrationConnector{}
	target := &mockMigrationConnector{}
	manager := NewRollbackManager(stateStore, source, target)

	state := &MigrationState{
		ID:       "mig-idempotent",
		SpecName: "phase2-test",
		Phase:    PhaseComplete,
		Workloads: []WorkloadMigration{
			{
				VM:         sampleVirtualMachines()[0],
				TargetVMID: "vm-1-target",
			},
		},
		StartedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := persistMigrationState(context.Background(), stateStore, state); err != nil {
		t.Fatalf("persistMigrationState() error = %v", err)
	}
	if _, err := manager.CreateRecoveryPoint(context.Background(), state); err != nil {
		t.Fatalf("CreateRecoveryPoint() error = %v", err)
	}

	if _, err := manager.Rollback(context.Background(), "mig-idempotent"); err != nil {
		t.Fatalf("Rollback() first error = %v", err)
	}
	if _, err := manager.Rollback(context.Background(), "mig-idempotent"); err != nil {
		t.Fatalf("Rollback() second error = %v", err)
	}
}

func TestRollback_MissingRecoveryPoint(t *testing.T) {
	t.Parallel()

	manager := NewRollbackManager(store.NewMemoryStore(), &mockMigrationConnector{}, &mockMigrationConnector{})
	if _, err := manager.Rollback(context.Background(), "missing"); err == nil {
		t.Fatal("Rollback() error = nil, want error")
	}
}

func TestRollback_PartialCleanupFailure_MarksMigrationFailed_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	source := &mockMigrationConnector{}
	target := &mockMigrationConnector{removeErr: os.ErrPermission}
	manager := NewRollbackManager(stateStore, source, target)

	state := &MigrationState{
		ID:       "mig-rollback-failed",
		SpecName: "phase3-hardening",
		Phase:    PhaseComplete,
		Workloads: []WorkloadMigration{
			{
				VM:         sampleVirtualMachines()[0],
				TargetVMID: "vm-1-target",
			},
		},
		StartedAt: nowUTC(),
		UpdatedAt: nowUTC(),
	}
	if err := persistMigrationState(context.Background(), stateStore, state); err != nil {
		t.Fatalf("persistMigrationState() error = %v", err)
	}
	if _, err := manager.CreateRecoveryPoint(context.Background(), state); err != nil {
		t.Fatalf("CreateRecoveryPoint() error = %v", err)
	}

	result, err := manager.Rollback(context.Background(), state.ID)
	if err == nil {
		t.Fatal("Rollback() error = nil, want partial cleanup failure")
	}
	if result == nil || len(result.Errors) == 0 {
		t.Fatalf("unexpected rollback result: %#v", result)
	}

	record, getErr := stateStore.GetMigration(context.Background(), store.DefaultTenantID, state.ID)
	if getErr != nil {
		t.Fatalf("GetMigration() error = %v", getErr)
	}

	var persisted MigrationState
	if unmarshalErr := json.Unmarshal(record.RawJSON, &persisted); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if persisted.Phase != PhaseFailed {
		t.Fatalf("persisted phase = %s, want %s", persisted.Phase, PhaseFailed)
	}
	if len(persisted.Errors) == 0 {
		t.Fatalf("persisted.Errors = %#v, want rollback diagnostics", persisted.Errors)
	}
}
