package migrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
	"github.com/google/uuid"
)

const (
	replicationPhaseInitialCopy   = "initial-copy"
	replicationPhaseIncremental   = "incremental-sync"
	replicationPhaseFinalSync     = "final-sync"
	replicationStateSpecName      = "warm-replication"
	defaultReplicationBlockSizeKB = 1024
)

// ReplicationConfig configures warm migration disk replication.
type ReplicationConfig struct {
	// SourceDisk is the source-side disk image path.
	SourceDisk string `json:"source_disk" yaml:"source_disk"`
	// TargetDisk is the target-side disk image path.
	TargetDisk string `json:"target_disk" yaml:"target_disk"`
	// SourceFormat is the source disk format.
	SourceFormat DiskFormat `json:"source_format" yaml:"source_format"`
	// TargetFormat is the target disk format.
	TargetFormat DiskFormat `json:"target_format" yaml:"target_format"`
	// BandwidthLimitMBps caps replication bandwidth in mebibytes per second. Zero means unlimited.
	BandwidthLimitMBps int `json:"bandwidth_limit_mbps,omitempty" yaml:"bandwidth_limit_mbps,omitempty"`
	// BlockSizeKB is the comparison and copy block size in kibibytes.
	BlockSizeKB int `json:"block_size_kb,omitempty" yaml:"block_size_kb,omitempty"`
	// OnProgress receives replication progress updates.
	OnProgress func(ReplicationProgress) `json:"-" yaml:"-"`
}

// ReplicationProgress reports warm migration replication progress.
type ReplicationProgress struct {
	// Phase is the current replication phase.
	Phase string `json:"phase" yaml:"phase"`
	// BytesCopied is the number of bytes copied so far.
	BytesCopied int64 `json:"bytes_copied" yaml:"bytes_copied"`
	// BytesTotal is the total number of bytes to copy.
	BytesTotal int64 `json:"bytes_total" yaml:"bytes_total"`
	// Percent is the completion percentage.
	Percent float64 `json:"percent" yaml:"percent"`
	// DirtyBlocks is the number of dirty blocks in the current sync round.
	DirtyBlocks int `json:"dirty_blocks" yaml:"dirty_blocks"`
	// SyncRound is the sync round number.
	SyncRound int `json:"sync_round" yaml:"sync_round"`
	// EstimatedCutoverSeconds is the estimated cutover duration in seconds.
	EstimatedCutoverSeconds int `json:"estimated_cutover_seconds" yaml:"estimated_cutover_seconds"`
}

// ReplicationState captures persisted warm migration replication state.
type ReplicationState struct {
	// ID is the replication identifier.
	ID string `json:"id" yaml:"id"`
	// Config is the effective replication configuration.
	Config ReplicationConfig `json:"config" yaml:"config"`
	// StartedAt is when warm replication was first started.
	StartedAt time.Time `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	// Phase is the current replication phase.
	Phase string `json:"phase" yaml:"phase"`
	// BytesCopied is the cumulative number of bytes copied.
	BytesCopied int64 `json:"bytes_copied" yaml:"bytes_copied"`
	// DirtyBitmap tracks dirty blocks from the most recent sync round.
	DirtyBitmap []byte `json:"dirty_bitmap,omitempty" yaml:"dirty_bitmap,omitempty"`
	// LastSyncAt is when the last sync completed.
	LastSyncAt time.Time `json:"last_sync_at" yaml:"last_sync_at"`
	// SyncRounds is the number of incremental sync rounds that have completed.
	SyncRounds int `json:"sync_rounds" yaml:"sync_rounds"`
}

// SyncResult captures the outcome of an incremental sync round.
type SyncResult struct {
	// DirtyBlocks is the number of dirty blocks detected.
	DirtyBlocks int `json:"dirty_blocks" yaml:"dirty_blocks"`
	// BytesSynced is the number of bytes written during the sync round.
	BytesSynced int64 `json:"bytes_synced" yaml:"bytes_synced"`
	// Duration is how long the sync round took.
	Duration time.Duration `json:"duration" yaml:"duration"`
	// Round is the completed sync round number.
	Round int `json:"round" yaml:"round"`
}

// CutoverResult captures the final sync outcome used during cutover.
type CutoverResult struct {
	// DowntimeSeconds is the estimated cutover downtime.
	DowntimeSeconds int `json:"downtime_seconds" yaml:"downtime_seconds"`
	// FinalSyncBytes is the number of bytes copied during the final sync.
	FinalSyncBytes int64 `json:"final_sync_bytes" yaml:"final_sync_bytes"`
	// Success reports whether cutover replication completed successfully.
	Success bool `json:"success" yaml:"success"`
}

// Replicator performs initial copy and change sync for warm migration workflows.
type Replicator struct {
	state  *ReplicationState
	store  store.Store
	sleep  func(time.Duration)
	now    func() time.Time
	newID  func() string
	buffer []byte
}

// NewReplicator creates a warm migration replicator.
func NewReplicator(config ReplicationConfig, stateStore store.Store) *Replicator {
	if config.BlockSizeKB <= 0 {
		config.BlockSizeKB = defaultReplicationBlockSizeKB
	}

	return &Replicator{
		state: &ReplicationState{
			ID:     uuid.NewString(),
			Config: config,
		},
		store:  stateStore,
		sleep:  time.Sleep,
		now:    nowUTC,
		newID:  uuid.NewString,
		buffer: make([]byte, config.BlockSizeKB*1024),
	}
}

// LoadState restores previously persisted warm-migration state from the configured store.
func (r *Replicator) LoadState(ctx context.Context, migrationID string) error {
	if r == nil {
		return fmt.Errorf("load replication state: replicator is nil")
	}
	if r.store == nil {
		return fmt.Errorf("load replication state: store is nil")
	}
	if stringsTrimSpace(migrationID) == "" {
		return fmt.Errorf("load replication state: migration ID is required")
	}

	record, err := r.store.GetMigration(ctx, store.TenantIDFromContext(ctx), migrationID)
	if err != nil {
		return fmt.Errorf("load replication state: %w", err)
	}

	var state ReplicationState
	if err := json.Unmarshal(record.RawJSON, &state); err != nil {
		return fmt.Errorf("load replication state: decode %s: %w", migrationID, err)
	}

	state.Config.OnProgress = r.state.Config.OnProgress
	if state.Config.SourceDisk == "" {
		state.Config.SourceDisk = r.state.Config.SourceDisk
	}
	if state.Config.TargetDisk == "" {
		state.Config.TargetDisk = r.state.Config.TargetDisk
	}
	if state.StartedAt.IsZero() {
		state.StartedAt = record.StartedAt
	}

	r.state = &state
	if err := r.validateConfig(); err != nil {
		return fmt.Errorf("load replication state: %w", err)
	}

	return nil
}

// StartInitialCopy performs the initial bulk copy from source to target.
func (r *Replicator) StartInitialCopy(ctx context.Context) error {
	if err := r.validateConfig(); err != nil {
		return fmt.Errorf("start initial copy: %w", err)
	}
	if r.state.StartedAt.IsZero() {
		r.state.StartedAt = r.now()
	}

	sourceInfo, err := os.Stat(r.state.Config.SourceDisk)
	if err != nil {
		return fmt.Errorf("start initial copy: stat source disk: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.state.Config.TargetDisk), 0o755); err != nil {
		return fmt.Errorf("start initial copy: create target directory: %w", err)
	}

	sourceFile, err := os.Open(r.state.Config.SourceDisk)
	if err != nil {
		return fmt.Errorf("start initial copy: open source disk: %w", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(r.state.Config.TargetDisk, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("start initial copy: open target disk: %w", err)
	}
	defer targetFile.Close()

	targetInfo, err := targetFile.Stat()
	if err != nil {
		return fmt.Errorf("start initial copy: stat target disk: %w", err)
	}

	resumeOffset := targetInfo.Size()
	if resumeOffset > sourceInfo.Size() {
		resumeOffset = 0
		if err := targetFile.Truncate(0); err != nil {
			return fmt.Errorf("start initial copy: truncate oversized target: %w", err)
		}
	}

	if _, err := sourceFile.Seek(resumeOffset, io.SeekStart); err != nil {
		return fmt.Errorf("start initial copy: seek source disk: %w", err)
	}
	if _, err := targetFile.Seek(resumeOffset, io.SeekStart); err != nil {
		return fmt.Errorf("start initial copy: seek target disk: %w", err)
	}

	r.state.Phase = replicationPhaseInitialCopy
	r.state.BytesCopied = resumeOffset
	r.state.DirtyBitmap = make([]byte, blockCount(sourceInfo.Size(), r.blockSizeBytes()))
	startedAt := time.Now()

	for {
		if err := ctx.Err(); err != nil {
			return r.persistAndWrap(ctx, "start initial copy", err)
		}

		n, readErr := sourceFile.Read(r.buffer)
		if n > 0 {
			if _, err := targetFile.Write(r.buffer[:n]); err != nil {
				return fmt.Errorf("start initial copy: write target disk: %w", err)
			}
			r.state.BytesCopied += int64(n)
			r.applyThrottle(startedAt, r.state.BytesCopied-resumeOffset)
			r.emitProgress(sourceInfo.Size(), 0)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return r.persistAndWrap(ctx, "start initial copy", fmt.Errorf("read source disk: %w", readErr))
		}
	}

	r.state.LastSyncAt = r.now()
	if err := r.persist(ctx); err != nil {
		return fmt.Errorf("start initial copy: %w", err)
	}

	return nil
}

// RunIncrementalSync synchronizes dirty blocks from source to target.
func (r *Replicator) RunIncrementalSync(ctx context.Context) (*SyncResult, error) {
	result, err := r.runSync(ctx, replicationPhaseIncremental)
	if err != nil {
		return nil, fmt.Errorf("run incremental sync: %w", err)
	}
	return result, nil
}

// ExecuteCutover runs the final sync and returns the estimated cutover outcome.
func (r *Replicator) ExecuteCutover(ctx context.Context) (*CutoverResult, error) {
	result, err := r.runSync(ctx, replicationPhaseFinalSync)
	if err != nil {
		return nil, fmt.Errorf("execute cutover: %w", err)
	}

	downtimeSeconds := result.Duration.Seconds()
	if downtimeSeconds < 1 && result.BytesSynced > 0 {
		downtimeSeconds = 1
	}

	return &CutoverResult{
		DowntimeSeconds: int(math.Ceil(downtimeSeconds)),
		FinalSyncBytes:  result.BytesSynced,
		Success:         true,
	}, nil
}

func (r *Replicator) runSync(ctx context.Context, phase string) (*SyncResult, error) {
	if err := r.validateConfig(); err != nil {
		return nil, err
	}
	if r.state.StartedAt.IsZero() {
		r.state.StartedAt = r.now()
	}

	sourceFile, err := os.Open(r.state.Config.SourceDisk)
	if err != nil {
		return nil, fmt.Errorf("open source disk: %w", err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat source disk: %w", err)
	}

	targetFile, err := os.OpenFile(r.state.Config.TargetDisk, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open target disk: %w", err)
	}
	defer targetFile.Close()

	targetInfo, err := targetFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat target disk: %w", err)
	}

	blockSize := int64(r.blockSizeBytes())
	totalBlocks := blockCount(sourceInfo.Size(), int(blockSize))
	r.state.Phase = phase
	r.state.DirtyBitmap = make([]byte, totalBlocks)
	r.state.SyncRounds++
	startedAt := time.Now()

	dirtyBlocks := 0
	var bytesSynced int64
	sourceBuffer := make([]byte, blockSize)
	targetBuffer := make([]byte, blockSize)

	for blockIndex := 0; blockIndex < totalBlocks; blockIndex++ {
		if err := ctx.Err(); err != nil {
			return nil, r.persistAndWrap(ctx, "run sync", err)
		}

		offset := int64(blockIndex) * blockSize
		sourceLength, err := readAtMost(sourceFile, sourceBuffer, offset)
		if err != nil {
			return nil, r.persistAndWrap(ctx, "run sync", fmt.Errorf("read source block %d: %w", blockIndex, err))
		}
		targetLength := 0
		if offset < targetInfo.Size() {
			targetLength, err = readAtMost(targetFile, targetBuffer, offset)
			if err != nil {
				return nil, r.persistAndWrap(ctx, "run sync", fmt.Errorf("read target block %d: %w", blockIndex, err))
			}
		}

		if sourceLength == targetLength && bytes.Equal(sourceBuffer[:sourceLength], targetBuffer[:targetLength]) {
			continue
		}

		dirtyBlocks++
		r.state.DirtyBitmap[blockIndex] = 1

		if _, err := targetFile.WriteAt(sourceBuffer[:sourceLength], offset); err != nil {
			return nil, r.persistAndWrap(ctx, "run sync", fmt.Errorf("write target block %d: %w", blockIndex, err))
		}
		bytesSynced += int64(sourceLength)
		r.state.BytesCopied += int64(sourceLength)
		r.applyThrottle(startedAt, bytesSynced)
	}

	r.state.LastSyncAt = r.now()
	if err := r.persist(ctx); err != nil {
		return nil, err
	}

	r.emitProgress(sourceInfo.Size(), dirtyBlocks)
	return &SyncResult{
		DirtyBlocks: dirtyBlocks,
		BytesSynced: bytesSynced,
		Duration:    time.Since(startedAt),
		Round:       r.state.SyncRounds,
	}, nil
}

func (r *Replicator) validateConfig() error {
	if r == nil || r.state == nil {
		return fmt.Errorf("replicator is nil")
	}
	if r.state.ID == "" {
		r.state.ID = r.newID()
	}
	if r.state.Config.BlockSizeKB <= 0 {
		r.state.Config.BlockSizeKB = defaultReplicationBlockSizeKB
	}
	if stringsTrimSpace(r.state.Config.SourceDisk) == "" {
		return fmt.Errorf("source disk is required")
	}
	if stringsTrimSpace(r.state.Config.TargetDisk) == "" {
		return fmt.Errorf("target disk is required")
	}
	if len(r.buffer) != r.blockSizeBytes() {
		r.buffer = make([]byte, r.blockSizeBytes())
	}
	return nil
}

func (r *Replicator) blockSizeBytes() int {
	blockSizeKB := r.state.Config.BlockSizeKB
	if blockSizeKB <= 0 {
		blockSizeKB = defaultReplicationBlockSizeKB
	}
	return blockSizeKB * 1024
}

func (r *Replicator) applyThrottle(startedAt time.Time, copiedBytes int64) {
	if r.state.Config.BandwidthLimitMBps <= 0 || copiedBytes <= 0 {
		return
	}

	targetDuration := time.Duration(float64(copiedBytes) / float64(r.state.Config.BandwidthLimitMBps*1024*1024) * float64(time.Second))
	sleepFor := targetDuration - time.Since(startedAt)
	if sleepFor > 0 {
		r.sleep(sleepFor)
	}
}

func (r *Replicator) emitProgress(totalBytes int64, dirtyBlocks int) {
	if r.state.Config.OnProgress == nil {
		return
	}

	percent := 100.0
	if totalBytes > 0 {
		percent = (float64(r.state.BytesCopied) / float64(totalBytes)) * 100
		if percent > 100 {
			percent = 100
		}
	}

	r.state.Config.OnProgress(ReplicationProgress{
		Phase:                   r.state.Phase,
		BytesCopied:             r.state.BytesCopied,
		BytesTotal:              totalBytes,
		Percent:                 math.Round(percent*100) / 100,
		DirtyBlocks:             dirtyBlocks,
		SyncRound:               r.state.SyncRounds,
		EstimatedCutoverSeconds: estimatedCutoverSeconds(dirtyBlocks, totalBytes, r.state.Config.BandwidthLimitMBps, r.blockSizeBytes()),
	})
}

func (r *Replicator) persist(ctx context.Context) error {
	if r.store == nil {
		return nil
	}
	if r.state.StartedAt.IsZero() {
		r.state.StartedAt = r.now()
	}
	persistCtx := ctx
	if ctx != nil && ctx.Err() != nil {
		persistCtx = store.ContextWithTenantID(context.Background(), store.TenantIDFromContext(ctx))
	}

	payload, err := json.Marshal(r.state)
	if err != nil {
		return fmt.Errorf("persist replication state: %w", err)
	}

	return r.store.SaveMigration(persistCtx, store.TenantIDFromContext(persistCtx), store.MigrationRecord{
		ID:        r.state.ID,
		TenantID:  store.TenantIDFromContext(persistCtx),
		SpecName:  replicationStateSpecName,
		Phase:     r.state.Phase,
		StartedAt: firstNonZeroTime(r.state.StartedAt, r.state.LastSyncAt, r.now()),
		UpdatedAt: r.now(),
		RawJSON:   payload,
	})
}

func (r *Replicator) persistAndWrap(ctx context.Context, operation string, err error) error {
	if persistErr := r.persist(ctx); persistErr != nil {
		return fmt.Errorf("%s: %w: %v", operation, err, persistErr)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func estimatedCutoverSeconds(dirtyBlocks int, totalBytes int64, bandwidthMBps int, blockSizeBytes int) int {
	if dirtyBlocks == 0 {
		return 0
	}
	if bandwidthMBps <= 0 {
		return 1
	}

	bytesToCopy := float64(dirtyBlocks * blockSizeBytes)
	seconds := bytesToCopy / float64(bandwidthMBps*1024*1024)
	return int(math.Ceil(seconds))
}

func blockCount(size int64, blockSizeBytes int) int {
	if size <= 0 || blockSizeBytes <= 0 {
		return 0
	}
	return int(math.Ceil(float64(size) / float64(blockSizeBytes)))
}

func readAtMost(file *os.File, buffer []byte, offset int64) (int, error) {
	n, err := file.ReadAt(buffer, offset)
	if err == io.EOF {
		return n, nil
	}
	return n, err
}

func stringsTrimSpace(value string) string {
	return strings.TrimSpace(value)
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}
