package migrate

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

// RecoveryPoint captures enough metadata to roll back a migration.
type RecoveryPoint struct {
	MigrationID    string            `json:"migration_id"`
	Phase          MigrationPhase    `json:"phase"`
	CreatedAt      time.Time         `json:"created_at"`
	SourceVMs      []VMRecoveryInfo  `json:"source_vms"`
	TargetVMs      []string          `json:"target_vms,omitempty"`
	ConvertedFiles []string          `json:"converted_files,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// VMRecoveryInfo stores source-side recovery metadata for a single workload.
type VMRecoveryInfo struct {
	VM                 models.VirtualMachine `json:"vm"`
	OriginalPowerState models.PowerState     `json:"original_power_state"`
	SnapshotID         string                `json:"snapshot_id,omitempty"`
}

// RollbackResult summarizes actions taken during rollback.
type RollbackResult struct {
	MigrationID       string        `json:"migration_id"`
	TargetVMsRemoved  int           `json:"target_vms_removed"`
	FilesCleanedUp    int           `json:"files_cleaned_up"`
	SourceVMsRestored int           `json:"source_vms_restored"`
	Errors            []string      `json:"errors,omitempty"`
	Duration          time.Duration `json:"duration"`
}

// RollbackManager manages recovery point creation and rollback execution.
type RollbackManager struct {
	store           store.Store
	sourceConnector connectors.Connector
	targetConnector connectors.Connector
}

// NewRollbackManager creates a rollback manager for migration recovery.
func NewRollbackManager(stateStore store.Store, source, target connectors.Connector) *RollbackManager {
	return &RollbackManager{
		store:           stateStore,
		sourceConnector: source,
		targetConnector: target,
	}
}

// CreateRecoveryPoint captures source and target state needed for rollback.
func (m *RollbackManager) CreateRecoveryPoint(ctx context.Context, state *MigrationState) (*RecoveryPoint, error) {
	if state == nil {
		return nil, fmt.Errorf("create recovery point: state is nil")
	}

	point := &RecoveryPoint{
		MigrationID:    state.ID,
		Phase:          state.Phase,
		CreatedAt:      nowUTC(),
		SourceVMs:      make([]VMRecoveryInfo, 0, len(state.Workloads)),
		TargetVMs:      make([]string, 0, len(state.Workloads)),
		ConvertedFiles: make([]string, 0),
		Metadata: map[string]string{
			"spec_name": state.SpecName,
		},
	}

	for _, workload := range state.Workloads {
		info := VMRecoveryInfo{
			VM:                 workload.VM,
			OriginalPowerState: workload.VM.PowerState,
		}
		if snapshotter, ok := m.sourceConnector.(vmSnapshotter); ok {
			snapshotID, err := snapshotter.CreateVMSnapshot(ctx, workload.VM.ID)
			if err != nil {
				return nil, fmt.Errorf("create recovery point for %s: %w", workload.VM.Name, err)
			}
			info.SnapshotID = snapshotID
		}
		point.SourceVMs = append(point.SourceVMs, info)
		if workload.TargetVMID != "" {
			point.TargetVMs = append(point.TargetVMs, workload.TargetVMID)
		}
		point.ConvertedFiles = append(point.ConvertedFiles, workload.ConvertedDiskPaths...)
	}

	if err := persistRecoveryPoint(ctx, m.store, point); err != nil {
		return nil, fmt.Errorf("create recovery point: %w", err)
	}

	return point, nil
}

// Rollback restores source-side state and removes target-side artifacts for a migration.
func (m *RollbackManager) Rollback(ctx context.Context, migrationID string) (*RollbackResult, error) {
	startedAt := time.Now()
	point, err := loadRecoveryPoint(ctx, m.store, migrationID)
	if err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}

	state, err := loadMigrationState(ctx, m.store, migrationID)
	if err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}

	result := &RollbackResult{
		MigrationID: migrationID,
		Errors:      make([]string, 0),
	}

	if controller, ok := m.targetConnector.(vmPowerController); ok {
		for _, vmID := range point.TargetVMs {
			if err := controller.PowerOffVM(ctx, vmID); err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}
	if remover, ok := m.targetConnector.(vmRemover); ok {
		for _, vmID := range point.TargetVMs {
			if err := remover.RemoveVM(ctx, vmID); err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.TargetVMsRemoved++
		}
	}

	for _, path := range point.ConvertedFiles {
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.FilesCleanedUp++
	}

	if controller, ok := m.sourceConnector.(vmPowerController); ok {
		for _, vm := range point.SourceVMs {
			if err := controller.RestoreVMPowerState(ctx, vm.VM.ID, vm.OriginalPowerState); err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.SourceVMsRestored++
		}
	}

	state.UpdatedAt = nowUTC()
	state.CompletedAt = state.UpdatedAt
	if len(result.Errors) > 0 {
		state.Phase = PhaseFailed
		for _, item := range result.Errors {
			state.Errors = append(state.Errors, "rollback: "+item)
		}
	} else {
		state.Phase = PhaseRolledBack
	}
	if err := persistMigrationState(ctx, m.store, state); err != nil {
		return nil, fmt.Errorf("rollback: %w", err)
	}

	result.Duration = time.Since(startedAt)
	if len(result.Errors) > 0 {
		log.Printf("component=rollback migration_id=%s restored=%d removed=%d files=%d errors=%q", migrationID, result.SourceVMsRestored, result.TargetVMsRemoved, result.FilesCleanedUp, strings.Join(result.Errors, "; "))
		return result, fmt.Errorf("rollback: completed with %d error(s): %s", len(result.Errors), strings.Join(result.Errors, "; "))
	}
	log.Printf("component=rollback migration_id=%s restored=%d removed=%d files=%d message=%q", migrationID, result.SourceVMsRestored, result.TargetVMsRemoved, result.FilesCleanedUp, "rollback complete")
	return result, nil
}
