package migrate

import (
	"fmt"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

// BuildStateFromInventory derives a persisted plan-state payload from inventory that has already been discovered.
func BuildStateFromInventory(spec *MigrationSpec, inventory *models.DiscoveryResult, migrationID string, generatedAt time.Time) (*MigrationState, error) {
	if spec == nil {
		return nil, fmt.Errorf("build migration state: spec is nil")
	}
	if inventory == nil {
		return nil, fmt.Errorf("build migration state: inventory is nil")
	}
	if stringsTrimSpace(migrationID) == "" {
		return nil, fmt.Errorf("build migration state: migration ID is required")
	}
	if generatedAt.IsZero() {
		generatedAt = nowUTC()
	}

	state := &MigrationState{
		ID:              migrationID,
		SpecName:        spec.Name,
		SourceAddress:   spec.Source.Address,
		SourcePlatform:  spec.Source.Platform,
		TargetAddress:   spec.Target.Address,
		TargetPlatform:  spec.Target.Platform,
		Phase:           PhasePlan,
		Window:          spec.Options.Window,
		Approval:        spec.Options.Approval,
		PendingApproval: spec.Options.Approval.Required && !spec.Options.Approval.Approved(),
		StartedAt:       generatedAt,
		UpdatedAt:       generatedAt,
		Checkpoints:     initializeCheckpoints(),
	}

	plan, err := BuildExecutionPlan(spec, inventory.VMs)
	if err != nil {
		return nil, fmt.Errorf("build migration state: %w", err)
	}

	state.Plan = plan
	state.Workloads = reorderWorkloadsByPlan(buildWorkloadMigrations(inventory.VMs, spec.Workloads), plan)
	if len(state.Workloads) == 0 {
		return nil, fmt.Errorf("build migration state: no workloads matched migration selectors")
	}

	markCheckpointCompleted(state, PhasePlan, "migration plan created")
	return state, nil
}
