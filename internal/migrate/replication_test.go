package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestReplicator_StartInitialCopy_ProgressAndTargetMatch(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	sourcePayload := bytes.Repeat([]byte("warm-migrate"), 2048)
	if err := os.WriteFile(sourcePath, sourcePayload, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	progressEvents := make([]ReplicationProgress, 0)
	replicator := NewReplicator(ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: 4,
		OnProgress: func(progress ReplicationProgress) {
			progressEvents = append(progressEvents, progress)
		},
	}, store.NewMemoryStore())

	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	targetPayload, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(targetPayload, sourcePayload) {
		t.Fatal("target payload does not match source payload")
	}
	if len(progressEvents) == 0 {
		t.Fatal("expected progress events during initial copy")
	}
}

func TestReplicator_RunIncrementalSync_DirtyBlocksDetected(t *testing.T) {
	t.Parallel()

	replicator, sourcePath, targetPath := newTestReplicator(t, 8)
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	if err := overwriteAt(sourcePath, 4096, bytes.Repeat([]byte{0x7f}, 4096)); err != nil {
		t.Fatalf("overwriteAt(source) error = %v", err)
	}

	result, err := replicator.RunIncrementalSync(context.Background())
	if err != nil {
		t.Fatalf("RunIncrementalSync() error = %v", err)
	}
	if result.DirtyBlocks == 0 || result.BytesSynced == 0 {
		t.Fatalf("unexpected sync result: %#v", result)
	}

	sourcePayload, _ := os.ReadFile(sourcePath)
	targetPayload, _ := os.ReadFile(targetPath)
	if !bytes.Equal(sourcePayload, targetPayload) {
		t.Fatal("target payload does not match source after incremental sync")
	}
}

func TestReplicator_Throttle_BandwidthLimitSleeps(t *testing.T) {
	t.Parallel()

	replicator := NewReplicator(ReplicationConfig{
		SourceDisk:         "source.qcow2",
		TargetDisk:         "target.qcow2",
		BlockSizeKB:        8,
		BandwidthLimitMBps: 1,
	}, store.NewMemoryStore())

	var slept bool
	replicator.sleep = func(dur time.Duration) {
		if dur > 0 {
			slept = true
		}
	}

	replicator.applyThrottle(time.Now(), 4*1024*1024)
	if !slept {
		t.Fatal("expected throttling sleep to occur")
	}
}

func TestReplicator_StartInitialCopy_ResumeAfterInterrupt(t *testing.T) {
	t.Parallel()

	replicator, sourcePath, targetPath := newTestReplicator(t, 8)
	sourcePayload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(source) error = %v", err)
	}
	if err := os.WriteFile(targetPath, sourcePayload[:4096], 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	targetPayload, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	if !bytes.Equal(sourcePayload, targetPayload) {
		t.Fatal("target payload does not match resumed source payload")
	}
}

func TestReplicator_ExecuteCutover_FinalSync(t *testing.T) {
	t.Parallel()

	replicator, sourcePath, _ := newTestReplicator(t, 8)
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	if err := overwriteAt(sourcePath, 0, bytes.Repeat([]byte{0x42}, 2048)); err != nil {
		t.Fatalf("overwriteAt() error = %v", err)
	}

	result, err := replicator.ExecuteCutover(context.Background())
	if err != nil {
		t.Fatalf("ExecuteCutover() error = %v", err)
	}
	if !result.Success || result.FinalSyncBytes == 0 {
		t.Fatalf("unexpected cutover result: %#v", result)
	}
}

func TestReplicator_RunIncrementalSync_InterruptedResumeFromPersistedState_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	sourcePayload := append(bytes.Repeat([]byte("A"), 1024*1024), bytes.Repeat([]byte("B"), 1024*1024)...)
	if err := os.WriteFile(sourcePath, sourcePayload, 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	replicator := NewReplicator(ReplicationConfig{
		SourceDisk:         sourcePath,
		TargetDisk:         targetPath,
		BlockSizeKB:        1024,
		BandwidthLimitMBps: 1,
	}, stateStore)
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	if err := overwriteAt(sourcePath, 0, bytes.Repeat([]byte{0x11}, 1024*1024)); err != nil {
		t.Fatalf("overwriteAt(first block) error = %v", err)
	}
	if err := overwriteAt(sourcePath, 1024*1024, bytes.Repeat([]byte{0x22}, 1024*1024)); err != nil {
		t.Fatalf("overwriteAt(second block) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	replicator.sleep = func(time.Duration) {
		cancel()
	}

	if _, err := replicator.RunIncrementalSync(ctx); err == nil {
		t.Fatal("RunIncrementalSync() error = nil, want interruption")
	} else if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunIncrementalSync() error = %v, want context cancellation", err)
	}

	record, err := stateStore.GetMigration(context.Background(), store.DefaultTenantID, replicator.state.ID)
	if err != nil {
		t.Fatalf("GetMigration() error = %v", err)
	}
	if record.Phase != replicationPhaseIncremental {
		t.Fatalf("record.Phase = %q, want %q", record.Phase, replicationPhaseIncremental)
	}

	resumed := NewReplicator(ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: 1024,
	}, stateStore)
	if err := resumed.LoadState(context.Background(), replicator.state.ID); err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	result, err := resumed.RunIncrementalSync(context.Background())
	if err != nil {
		t.Fatalf("resumed RunIncrementalSync() error = %v", err)
	}
	if result.DirtyBlocks == 0 || result.BytesSynced == 0 {
		t.Fatalf("unexpected resumed sync result: %#v", result)
	}

	targetPayload, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	updatedSourcePayload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(source) error = %v", err)
	}
	if !bytes.Equal(updatedSourcePayload, targetPayload) {
		t.Fatal("target payload does not match source after resumed sync")
	}
}

func TestReplicator_LoadState_BackfillsLegacyStartedAt_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	startedAt := time.Date(2026, time.April, 4, 14, 0, 0, 0, time.UTC)
	legacyState := map[string]any{
		"id": "warm-state-legacy",
		"config": map[string]any{
			"source_disk":   "source.qcow2",
			"target_disk":   "target.qcow2",
			"block_size_kb": 4,
		},
		"phase":        replicationPhaseInitialCopy,
		"bytes_copied": 4096,
		"sync_rounds":  1,
	}
	payload, err := json.Marshal(legacyState)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := stateStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
		ID:        "warm-state-legacy",
		SpecName:  replicationStateSpecName,
		Phase:     replicationPhaseInitialCopy,
		StartedAt: startedAt,
		UpdatedAt: startedAt.Add(time.Minute),
		RawJSON:   payload,
	}); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	replicator := NewReplicator(ReplicationConfig{}, stateStore)
	if err := replicator.LoadState(context.Background(), "warm-state-legacy"); err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !replicator.state.StartedAt.Equal(startedAt) {
		t.Fatalf("StartedAt = %s, want %s", replicator.state.StartedAt, startedAt)
	}
}

func newTestReplicator(t *testing.T, blockSizeKB int) (*Replicator, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	sourcePayload := append(bytes.Repeat([]byte("A"), 4096), bytes.Repeat([]byte("B"), 4096)...)
	if err := os.WriteFile(sourcePath, sourcePayload, 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	return NewReplicator(ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: blockSizeKB,
	}, store.NewMemoryStore()), sourcePath, targetPath
}

func overwriteAt(path string, offset int64, payload []byte) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteAt(payload, offset)
	return err
}
