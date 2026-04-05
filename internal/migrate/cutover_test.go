package migrate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestCutoverCoordinator_ExecuteCutover_Successful(t *testing.T) {
	t.Parallel()

	coordinator, plan := newCutoverHarness(t, false)
	report, err := coordinator.ExecuteCutover(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteCutover() error = %v", err)
	}
	if !report.BootVerified || report.TargetVMID == "" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestCutoverCoordinator_ExecuteCutover_BootFailureAutoRollback(t *testing.T) {
	t.Parallel()

	coordinator, plan := newCutoverHarness(t, true)
	report, err := coordinator.ExecuteCutover(context.Background(), plan)
	if err == nil {
		t.Fatal("ExecuteCutover() error = nil, want boot failure")
	}
	if !report.RolledBack {
		t.Fatalf("RolledBack = %t, want true", report.RolledBack)
	}
}

func TestCutoverCoordinator_ExecuteCutover_BootFailureNoAutoRollback(t *testing.T) {
	t.Parallel()

	coordinator, plan := newCutoverHarness(t, true)
	plan.AutoRollbackOnFailure = false

	report, err := coordinator.ExecuteCutover(context.Background(), plan)
	if err == nil {
		t.Fatal("ExecuteCutover() error = nil, want boot failure")
	}
	if report.RolledBack {
		t.Fatalf("RolledBack = %t, want false", report.RolledBack)
	}
}

func TestCutoverCoordinator_ExecuteCutover_DowntimeTracked(t *testing.T) {
	t.Parallel()

	coordinator, plan := newCutoverHarness(t, false)
	report, err := coordinator.ExecuteCutover(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteCutover() error = %v", err)
	}
	if report.TotalDowntime <= 0 {
		t.Fatalf("TotalDowntime = %s, want > 0", report.TotalDowntime)
	}
}

func newCutoverHarness(t *testing.T, bootFailure bool) (*CutoverCoordinator, *CutoverPlan) {
	t.Helper()

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	sourcePayload := append(bytes.Repeat([]byte("A"), 4096), bytes.Repeat([]byte("B"), 4096)...)
	if err := os.WriteFile(sourcePath, sourcePayload, 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	replicator := NewReplicator(ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: 4,
	}, store.NewMemoryStore())
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}
	if err := overwriteAt(sourcePath, 0, bytes.Repeat([]byte{0x33}, 1024)); err != nil {
		t.Fatalf("overwriteAt() error = %v", err)
	}

	source := &mockMigrationConnector{platform: models.PlatformVMware}
	target := &mockMigrationConnector{platform: models.PlatformProxmox}
	if bootFailure {
		target.verifyErr = context.DeadlineExceeded
	}

	stateStore := store.NewMemoryStore()
	rollback := NewRollbackManager(stateStore, source, target)
	replicator.store = stateStore

	plan := &CutoverPlan{
		MigrationID:           "warm-cutover",
		SourceVM:              sampleVirtualMachines()[0],
		TargetPlatform:        models.PlatformProxmox,
		ReplicationState:      replicator.state,
		NetworkMappings:       []MappedNIC{{Original: sampleVirtualMachines()[0].NICs[0], TargetNetwork: "vmbr0"}},
		BootTimeout:           50 * time.Millisecond,
		AutoRollbackOnFailure: true,
	}

	return NewCutoverCoordinator(source, target, replicator, rollback, nil), plan
}
