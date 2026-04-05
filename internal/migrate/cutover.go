package migrate

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

// CutoverPlan describes the final warm-migration switchover inputs.
type CutoverPlan struct {
	// MigrationID is the migration identifier associated with the cutover.
	MigrationID string `json:"migration_id" yaml:"migration_id"`
	// SourceVM is the source workload being cut over.
	SourceVM models.VirtualMachine `json:"source_vm" yaml:"source_vm"`
	// TargetPlatform is the destination platform.
	TargetPlatform models.Platform `json:"target_platform" yaml:"target_platform"`
	// ReplicationState is the replication state to cut over from.
	ReplicationState *ReplicationState `json:"replication_state,omitempty" yaml:"replication_state,omitempty"`
	// NetworkMappings contains final NIC mappings for the target VM.
	NetworkMappings []MappedNIC `json:"network_mappings,omitempty" yaml:"network_mappings,omitempty"`
	// BootTimeout limits how long boot verification may take.
	BootTimeout time.Duration `json:"boot_timeout,omitempty" yaml:"boot_timeout,omitempty"`
	// AutoRollbackOnFailure controls whether failed cutovers are rolled back automatically.
	AutoRollbackOnFailure bool `json:"auto_rollback_on_failure,omitempty" yaml:"auto_rollback_on_failure,omitempty"`
}

// CutoverReport captures the outcome of a warm migration cutover.
type CutoverReport struct {
	// MigrationID is the migration identifier.
	MigrationID string `json:"migration_id" yaml:"migration_id"`
	// SourceVM is the source VM name.
	SourceVM string `json:"source_vm" yaml:"source_vm"`
	// TargetVMID is the created target VM identifier.
	TargetVMID string `json:"target_vm_id" yaml:"target_vm_id"`
	// TotalDowntime is the full measured downtime across power-off, final sync, and boot verification.
	TotalDowntime time.Duration `json:"total_downtime" yaml:"total_downtime"`
	// FinalSyncDuration is how long the last sync round took.
	FinalSyncDuration time.Duration `json:"final_sync_duration" yaml:"final_sync_duration"`
	// BootVerified reports whether boot verification succeeded.
	BootVerified bool `json:"boot_verified" yaml:"boot_verified"`
	// RolledBack reports whether rollback was triggered.
	RolledBack bool `json:"rolled_back" yaml:"rolled_back"`
	// Events contains all emitted migration events.
	Events []MigrationEvent `json:"events" yaml:"events"`
}

// CutoverCoordinator coordinates final sync, shutdown, and target boot verification.
type CutoverCoordinator struct {
	source     connectors.Connector
	target     connectors.Connector
	replicator *Replicator
	rollback   *RollbackManager
	onEvent    func(MigrationEvent)
}

// NewCutoverCoordinator creates a cutover coordinator.
func NewCutoverCoordinator(source, target connectors.Connector, replicator *Replicator, rollback *RollbackManager, onEvent func(MigrationEvent)) *CutoverCoordinator {
	return &CutoverCoordinator{
		source:     source,
		target:     target,
		replicator: replicator,
		rollback:   rollback,
		onEvent:    onEvent,
	}
}

// ExecuteCutover performs the warm-migration cutover.
func (c *CutoverCoordinator) ExecuteCutover(ctx context.Context, plan *CutoverPlan) (*CutoverReport, error) {
	if c == nil {
		return nil, fmt.Errorf("execute cutover: coordinator is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("execute cutover: plan is nil")
	}
	if plan.BootTimeout <= 0 {
		plan.BootTimeout = 5 * time.Minute
	}
	if c.replicator == nil {
		return nil, fmt.Errorf("execute cutover: replicator is required")
	}
	if plan.ReplicationState != nil {
		c.replicator.state = plan.ReplicationState
	}

	report := &CutoverReport{
		MigrationID: plan.MigrationID,
		SourceVM:    plan.SourceVM.Name,
		Events:      make([]MigrationEvent, 0),
	}

	emit := func(phase MigrationPhase, message string, progress int) {
		event := MigrationEvent{
			Timestamp: nowUTC(),
			Phase:     phase,
			VM:        plan.SourceVM.Name,
			Message:   message,
			Progress:  progress,
		}
		log.Printf("component=cutover migration_id=%s phase=%s vm=%s progress=%d message=%q", plan.MigrationID, phase, plan.SourceVM.Name, progress, message)
		report.Events = append(report.Events, event)
		if c.onEvent != nil {
			c.onEvent(event)
		}
	}

	if err := c.persistCutoverState(ctx, plan); err != nil {
		return nil, err
	}

	if c.rollback != nil {
		if _, err := c.rollback.CreateRecoveryPoint(ctx, &MigrationState{
			ID:        plan.MigrationID,
			SpecName:  "warm-cutover",
			Phase:     PhaseVerify,
			StartedAt: nowUTC(),
			UpdatedAt: nowUTC(),
			Workloads: []WorkloadMigration{{VM: plan.SourceVM}},
		}); err != nil {
			return nil, fmt.Errorf("execute cutover: create recovery point: %w", err)
		}
	}

	downtimeStart := time.Now()
	if controller, ok := c.source.(vmPowerController); ok {
		emit(PhaseConfigure, "Powering off source VM", 15)
		if err := controller.PowerOffVM(ctx, plan.SourceVM.ID); err != nil {
			return nil, fmt.Errorf("execute cutover: power off source VM: %w", err)
		}
	}

	emit(PhaseVerify, "Running final sync", 35)
	finalSyncStartedAt := time.Now()
	cutoverResult, err := c.replicator.ExecuteCutover(ctx)
	if err != nil {
		return nil, fmt.Errorf("execute cutover: final sync: %w", err)
	}
	report.FinalSyncDuration = time.Since(finalSyncStartedAt)

	targetVMID := plan.SourceVM.ID + "-warm-target"
	if importer, ok := c.target.(vmImporter); ok {
		targetVMID, err = importer.CreateVM(ctx, plan.SourceVM, []string{c.replicator.state.Config.TargetDisk}, "", "")
		if err != nil {
			return nil, fmt.Errorf("execute cutover: create target VM: %w", err)
		}
	}
	report.TargetVMID = targetVMID

	if configurer, ok := c.target.(vmNetworkConfigurer); ok && len(plan.NetworkMappings) > 0 {
		if err := configurer.ConfigureVMNetworks(ctx, targetVMID, plan.NetworkMappings); err != nil {
			return nil, fmt.Errorf("execute cutover: configure target networks: %w", err)
		}
	}

	if controller, ok := c.target.(vmPowerController); ok {
		emit(PhaseVerify, "Powering on target VM", 70)
		if err := controller.PowerOnVM(ctx, targetVMID); err != nil {
			return nil, fmt.Errorf("execute cutover: power on target VM: %w", err)
		}
	}

	emit(PhaseVerify, "Waiting for boot verification", 85)
	verifier := selectBootVerifier(c.target)
	bootErr := verifier.WaitForBoot(ctx, targetVMID, plan.BootTimeout)
	if bootErr != nil {
		if plan.AutoRollbackOnFailure && c.rollback != nil {
			if _, rollbackErr := c.rollback.Rollback(ctx, plan.MigrationID); rollbackErr != nil {
				return nil, fmt.Errorf("execute cutover: boot verification failed: %v; rollback failed: %w", bootErr, rollbackErr)
			}
			report.RolledBack = true
		}
		return report, fmt.Errorf("execute cutover: verify boot: %w", bootErr)
	}

	report.BootVerified = true
	report.TotalDowntime = time.Since(downtimeStart)
	if cutoverResult.DowntimeSeconds > 0 && report.TotalDowntime < time.Duration(cutoverResult.DowntimeSeconds)*time.Second {
		report.TotalDowntime = time.Duration(cutoverResult.DowntimeSeconds) * time.Second
	}
	emit(PhaseComplete, "Warm migration cutover complete", 100)
	return report, nil
}

func (c *CutoverCoordinator) persistCutoverState(ctx context.Context, plan *CutoverPlan) error {
	if c.rollback == nil || c.rollback.store == nil {
		return nil
	}

	state := &MigrationState{
		ID:             plan.MigrationID,
		SpecName:       "warm-cutover",
		SourcePlatform: plan.SourceVM.Platform,
		TargetPlatform: plan.TargetPlatform,
		Phase:          PhaseVerify,
		StartedAt:      nowUTC(),
		UpdatedAt:      nowUTC(),
		Workloads: []WorkloadMigration{
			{
				VM:              plan.SourceVM,
				NetworkMappings: mappedNICMap(plan.NetworkMappings),
			},
		},
	}
	if err := persistMigrationState(store.ContextWithTenantID(ctx, store.TenantIDFromContext(ctx)), c.rollback.store, state); err != nil {
		return fmt.Errorf("execute cutover: persist migration state: %w", err)
	}
	return nil
}

func mappedNICMap(items []MappedNIC) map[string]string {
	if len(items) == 0 {
		return nil
	}

	output := make(map[string]string, len(items))
	for _, item := range items {
		output[item.Original.ID] = item.TargetNetwork
	}
	return output
}

func selectBootVerifier(connector connectors.Connector) BootVerifier {
	switch connector.Platform() {
	case models.PlatformVMware:
		return NewVMwareBootVerifier(connector)
	case models.PlatformProxmox:
		return NewProxmoxBootVerifier(connector)
	default:
		return NewGenericBootVerifier(connector)
	}
}
