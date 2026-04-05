package migrate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/google/uuid"
)

// MigrationPhase identifies a phase in the cold migration state machine.
type MigrationPhase string

const (
	// PhasePlan builds the migration plan from discovery data and selectors.
	PhasePlan MigrationPhase = "plan"
	// PhaseExport exports source VM disk data.
	PhaseExport MigrationPhase = "export"
	// PhaseConvert converts exported disks to the target format.
	PhaseConvert MigrationPhase = "convert"
	// PhaseImport creates target-side VMs and attaches disks.
	PhaseImport MigrationPhase = "import"
	// PhaseConfigure applies target-side network and placement settings.
	PhaseConfigure MigrationPhase = "configure"
	// PhaseVerify verifies that migrated VMs are healthy on the target.
	PhaseVerify MigrationPhase = "verify"
	// PhaseComplete marks a successful migration.
	PhaseComplete MigrationPhase = "complete"
	// PhaseFailed marks a failed migration.
	PhaseFailed MigrationPhase = "failed"
	// PhaseRolledBack marks a migration that has been rolled back.
	PhaseRolledBack MigrationPhase = "rolled_back"
)

// MigrationState captures the full persisted state of a migration execution.
type MigrationState struct {
	// ID is the migration identifier.
	ID string `json:"id"`
	// SpecName is the migration specification name.
	SpecName string `json:"spec_name"`
	// SourceAddress is the source endpoint address.
	SourceAddress string `json:"source_address,omitempty"`
	// SourcePlatform is the source platform.
	SourcePlatform models.Platform `json:"source_platform,omitempty"`
	// TargetAddress is the target endpoint address.
	TargetAddress string `json:"target_address,omitempty"`
	// TargetPlatform is the target platform.
	TargetPlatform models.Platform `json:"target_platform,omitempty"`
	// Phase is the current migration phase.
	Phase MigrationPhase `json:"phase"`
	// Workloads stores per-workload migration progress.
	Workloads []WorkloadMigration `json:"workloads"`
	// Plan is the persisted execution plan for the migration.
	Plan *MigrationPlan `json:"plan,omitempty"`
	// Checkpoints stores resumable phase progress.
	Checkpoints []MigrationCheckpoint `json:"checkpoints,omitempty"`
	// Window captures the execution schedule window.
	Window ExecutionWindow `json:"window,omitempty"`
	// Approval captures approval-gate state for the run.
	Approval ApprovalGate `json:"approval,omitempty"`
	// PendingApproval reports whether execution is blocked on approval.
	PendingApproval bool `json:"pending_approval,omitempty"`
	// StartedAt is when the migration began.
	StartedAt time.Time `json:"started_at"`
	// UpdatedAt is when state was last persisted.
	UpdatedAt time.Time `json:"updated_at"`
	// CompletedAt is when the migration finished.
	CompletedAt time.Time `json:"completed_at,omitempty"`
	// Errors stores execution errors and gating failures.
	Errors []string `json:"errors,omitempty"`
}

// WorkloadMigration captures the phase-specific state of a single VM migration.
type WorkloadMigration struct {
	VM                 models.VirtualMachine `json:"vm"`
	Phase              MigrationPhase        `json:"phase"`
	SourceDiskPaths    []string              `json:"source_disk_paths,omitempty"`
	ConvertedDiskPaths []string              `json:"converted_disk_paths,omitempty"`
	TargetVMID         string                `json:"target_vm_id,omitempty"`
	NetworkMappings    map[string]string     `json:"network_mappings,omitempty"`
	Error              string                `json:"error,omitempty"`
}

// MigrationEvent reports user-visible migration progress updates.
type MigrationEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Phase     MigrationPhase `json:"phase"`
	VM        string         `json:"vm,omitempty"`
	Message   string         `json:"message"`
	Progress  int            `json:"progress"`
}

// Orchestrator coordinates the end-to-end cold migration workflow.
type Orchestrator struct {
	sourceConnector connectors.Connector
	targetConnector connectors.Connector
	engine          *discovery.Engine
	store           store.Store
	onEvent         func(MigrationEvent)
	convertDisk     diskConvertFunc
	newID           func() string
}

// NewOrchestrator creates an orchestrator backed by source and target connectors.
func NewOrchestrator(source, target connectors.Connector, stateStore store.Store, onEvent func(MigrationEvent)) *Orchestrator {
	if stateStore == nil {
		stateStore = store.NewMemoryStore()
	}

	return &Orchestrator{
		sourceConnector: source,
		targetConnector: target,
		engine:          discovery.NewEngine(),
		store:           stateStore,
		onEvent:         onEvent,
		convertDisk:     ConvertDisk,
		newID:           uuid.NewString,
	}
}

// SetIDGenerator overrides migration ID creation for callers that need deterministic IDs.
func (o *Orchestrator) SetIDGenerator(generator func() string) {
	if generator == nil {
		return
	}

	o.newID = generator
}

// SetDiskConverter overrides the disk conversion implementation used during execution.
func (o *Orchestrator) SetDiskConverter(converter func(context.Context, ConversionRequest) (*ConversionResult, error)) {
	if converter == nil {
		return
	}

	o.convertDisk = converter
}

// Execute runs a migration spec through the cold migration state machine.
func (o *Orchestrator) Execute(ctx context.Context, spec *MigrationSpec) (*MigrationState, error) {
	if spec == nil {
		return nil, fmt.Errorf("execute migration: spec is nil")
	}
	if o.sourceConnector == nil || o.targetConnector == nil {
		return nil, fmt.Errorf("execute migration: source and target connectors are required")
	}

	state := o.newMigrationState(spec, o.newID())

	sourceResult, targetResult, err := o.discoverInventories(ctx)
	if err != nil {
		return o.failState(ctx, state, PhasePlan, err)
	}

	plan, err := BuildExecutionPlan(spec, sourceResult.VMs)
	if err != nil {
		return o.failState(ctx, state, PhasePlan, err)
	}

	state.Plan = plan
	state.Workloads = reorderWorkloadsByPlan(buildWorkloadMigrations(sourceResult.VMs, spec.Workloads), plan)
	if len(state.Workloads) == 0 {
		return o.failState(ctx, state, PhasePlan, fmt.Errorf("no workloads matched migration selectors"))
	}
	markCheckpointCompleted(state, PhasePlan, "migration plan created")

	if err := persistMigrationState(ctx, o.store, state); err != nil {
		return nil, fmt.Errorf("execute migration: %w", err)
	}
	o.emit(PhasePlan, "", "migration plan created", 10)

	if spec.Options.DryRun {
		return state, nil
	}
	if err := o.validateExecutionReadiness(state, spec, nowUTC()); err != nil {
		state.Errors = append(state.Errors, err.Error())
		state.PendingApproval = spec.Options.Approval.Required && !spec.Options.Approval.Approved()
		state.UpdatedAt = nowUTC()
		if persistErr := persistMigrationState(ctx, o.store, state); persistErr != nil {
			return nil, fmt.Errorf("execute migration: %w", persistErr)
		}
		return state, fmt.Errorf("execute migration: %w", err)
	}

	if _, err := NewRollbackManager(o.store, o.sourceConnector, o.targetConnector).CreateRecoveryPoint(ctx, state); err != nil {
		return o.failState(ctx, state, PhasePlan, err)
	}

	return o.runExecution(ctx, spec, state, sourceResult, targetResult, 0)
}

// Resume continues a previously persisted migration from the last incomplete checkpoint.
func (o *Orchestrator) Resume(ctx context.Context, migrationID string, spec *MigrationSpec) (*MigrationState, error) {
	if stringsTrimSpace(migrationID) == "" {
		return nil, fmt.Errorf("resume migration: migration ID is required")
	}
	if spec == nil {
		return nil, fmt.Errorf("resume migration: spec is nil")
	}
	if o.sourceConnector == nil || o.targetConnector == nil {
		return nil, fmt.Errorf("resume migration: source and target connectors are required")
	}

	state, err := loadMigrationState(ctx, o.store, migrationID)
	if err != nil {
		return nil, fmt.Errorf("resume migration: %w", err)
	}
	if state.Phase == PhaseComplete {
		return state, nil
	}
	if state.Phase == PhaseRolledBack {
		return nil, fmt.Errorf("resume migration: migration %s has already been rolled back", migrationID)
	}

	sourceResult, targetResult, err := o.discoverInventories(ctx)
	if err != nil {
		return o.failState(ctx, state, state.Phase, err)
	}

	if state.Plan == nil {
		plan, buildErr := BuildExecutionPlan(spec, sourceResult.VMs)
		if buildErr != nil {
			return o.failState(ctx, state, PhasePlan, buildErr)
		}
		state.Plan = plan
	}
	if len(state.Workloads) == 0 {
		state.Workloads = reorderWorkloadsByPlan(buildWorkloadMigrations(sourceResult.VMs, spec.Workloads), state.Plan)
	}
	if len(state.Checkpoints) == 0 {
		state.Checkpoints = initializeCheckpoints()
		markCheckpointCompleted(state, PhasePlan, "migration plan restored")
	}
	if err := o.validateExecutionReadiness(state, spec, nowUTC()); err != nil {
		state.Errors = append(state.Errors, err.Error())
		state.PendingApproval = spec.Options.Approval.Required && !spec.Options.Approval.Approved()
		state.UpdatedAt = nowUTC()
		if persistErr := persistMigrationState(ctx, o.store, state); persistErr != nil {
			return nil, fmt.Errorf("resume migration: %w", persistErr)
		}
		return state, fmt.Errorf("resume migration: %w", err)
	}

	startIndex := nextPhaseStartIndex(state)
	if startIndex < 0 {
		state.Phase = PhaseComplete
		state.PendingApproval = false
		state.UpdatedAt = nowUTC()
		state.CompletedAt = state.UpdatedAt
		if err := persistMigrationState(ctx, o.store, state); err != nil {
			return nil, fmt.Errorf("resume migration: %w", err)
		}
		return state, nil
	}

	return o.runExecution(ctx, spec, state, sourceResult, targetResult, startIndex)
}

func (o *Orchestrator) runExecution(ctx context.Context, spec *MigrationSpec, state *MigrationState, sourceResult, targetResult *models.DiscoveryResult, startIndex int) (*MigrationState, error) {
	phases := []struct {
		phase   MigrationPhase
		message string
		run     func(context.Context, *MigrationSpec, *MigrationState, *models.DiscoveryResult) error
	}{
		{phase: PhaseExport, message: "exporting source disks", run: o.runExport},
		{phase: PhaseConvert, message: "converting disks for target platform", run: o.runConvert},
		{phase: PhaseImport, message: "importing workloads on target platform", run: o.runImport},
		{phase: PhaseConfigure, message: "configuring target workloads", run: func(ctx context.Context, spec *MigrationSpec, state *MigrationState, _ *models.DiscoveryResult) error {
			return o.runConfigure(ctx, spec, state, targetResult)
		}},
		{phase: PhaseVerify, message: "verifying target workloads", run: o.runVerify},
	}

	for index, phase := range phases[startIndex:] {
		absoluteIndex := startIndex + index
		if err := o.setPhase(ctx, state, phase.phase); err != nil {
			return nil, fmt.Errorf("run migration: %w", err)
		}
		o.emit(phase.phase, "", phase.message, 20+(absoluteIndex*15))

		if _, err := NewRollbackManager(o.store, o.sourceConnector, o.targetConnector).CreateRecoveryPoint(ctx, state); err != nil {
			return o.failState(ctx, state, phase.phase, err)
		}

		if err := phase.run(ctx, spec, state, sourceResult); err != nil {
			return o.failState(ctx, state, phase.phase, err)
		}
		markCheckpointCompleted(state, phase.phase, phase.message)

		if err := persistMigrationState(ctx, o.store, state); err != nil {
			return nil, fmt.Errorf("run migration: %w", err)
		}
	}

	state.Phase = PhaseComplete
	state.PendingApproval = false
	state.UpdatedAt = nowUTC()
	state.CompletedAt = state.UpdatedAt
	for index := range state.Workloads {
		state.Workloads[index].Phase = PhaseComplete
	}
	if err := persistMigrationState(ctx, o.store, state); err != nil {
		return nil, fmt.Errorf("execute migration: %w", err)
	}
	o.emit(PhaseComplete, "", "migration complete", 100)

	return state, nil
}

func (o *Orchestrator) newMigrationState(spec *MigrationSpec, migrationID string) *MigrationState {
	return &MigrationState{
		ID:              migrationID,
		SpecName:        spec.Name,
		SourceAddress:   spec.Source.Address,
		SourcePlatform:  spec.Source.Platform,
		TargetAddress:   spec.Target.Address,
		TargetPlatform:  spec.Target.Platform,
		Phase:           PhasePlan,
		Checkpoints:     initializeCheckpoints(),
		Window:          spec.Options.Window,
		Approval:        spec.Options.Approval,
		PendingApproval: spec.Options.Approval.Required && !spec.Options.Approval.Approved(),
		StartedAt:       nowUTC(),
		UpdatedAt:       nowUTC(),
	}
}

func (o *Orchestrator) validateExecutionReadiness(state *MigrationState, spec *MigrationSpec, current time.Time) error {
	if state == nil {
		return fmt.Errorf("execution readiness: state is nil")
	}
	if spec == nil {
		return fmt.Errorf("execution readiness: spec is nil")
	}

	window := spec.Options.Window
	if !window.NotBefore.IsZero() && current.Before(window.NotBefore) {
		return fmt.Errorf("migration window opens at %s", window.NotBefore.Format(time.RFC3339))
	}
	if !window.NotAfter.IsZero() && current.After(window.NotAfter) {
		return fmt.Errorf("migration window closed at %s", window.NotAfter.Format(time.RFC3339))
	}
	if spec.Options.Approval.Required && !spec.Options.Approval.Approved() {
		return fmt.Errorf("migration requires approval before execution")
	}

	state.PendingApproval = false
	state.Approval = spec.Options.Approval
	return nil
}

func (o *Orchestrator) discoverInventories(ctx context.Context) (*models.DiscoveryResult, *models.DiscoveryResult, error) {
	if err := o.sourceConnector.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("connect source: %w", err)
	}
	defer func() { _ = o.sourceConnector.Close() }()

	if err := o.targetConnector.Connect(ctx); err != nil {
		return nil, nil, fmt.Errorf("connect target: %w", err)
	}
	defer func() { _ = o.targetConnector.Close() }()

	sourceResult, err := o.sourceConnector.Discover(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("discover source: %w", err)
	}
	targetResult, err := o.targetConnector.Discover(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("discover target: %w", err)
	}

	return sourceResult, targetResult, nil
}

func (o *Orchestrator) runExport(ctx context.Context, spec *MigrationSpec, state *MigrationState, _ *models.DiscoveryResult) error {
	for index := range state.Workloads {
		workload := &state.Workloads[index]
		workload.Phase = PhaseExport

		if spec.Options.ShutdownSource {
			if controller, ok := o.sourceConnector.(vmPowerController); ok && workload.VM.PowerState == models.PowerOn {
				if err := controller.PowerOffVM(ctx, workload.VM.ID); err != nil {
					workload.Error = err.Error()
					return fmt.Errorf("export %s: power off source: %w", workload.VM.Name, err)
				}
			}
		}

		if exporter, ok := o.sourceConnector.(diskExporter); ok {
			paths, err := exporter.ExportVMDisks(ctx, workload.VM)
			if err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("export %s: %w", workload.VM.Name, err)
			}
			workload.SourceDiskPaths = append([]string(nil), paths...)
		} else {
			workload.SourceDiskPaths = defaultSourceDiskPaths(state.ID, workload.VM)
		}

		o.emit(PhaseExport, workload.VM.Name, "exported source workload", 25)
	}

	return nil
}

func (o *Orchestrator) runConvert(ctx context.Context, spec *MigrationSpec, state *MigrationState, _ *models.DiscoveryResult) error {
	targetFormat := targetDiskFormat(spec.Target.Platform)

	for index := range state.Workloads {
		workload := &state.Workloads[index]
		workload.Phase = PhaseConvert
		workload.ConvertedDiskPaths = workload.ConvertedDiskPaths[:0]

		for _, path := range workload.SourceDiskPaths {
			sourceFormat := inferDiskFormat(path, workload.VM.Platform)
			targetPath := deriveTargetDiskPath(path, targetFormat)
			if filepath.Clean(targetPath) == filepath.Clean(path) {
				targetPath = path + ".converted"
			}

			result, err := o.convertDisk(ctx, ConversionRequest{
				SourcePath:   path,
				SourceFormat: sourceFormat,
				TargetPath:   targetPath,
				TargetFormat: targetFormat,
				Thin:         true,
			})
			if err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("convert %s disk %s: %w", workload.VM.Name, path, err)
			}

			workload.ConvertedDiskPaths = append(workload.ConvertedDiskPaths, result.TargetPath)
		}

		o.emit(PhaseConvert, workload.VM.Name, "converted exported disks", 45)
	}

	return nil
}

func (o *Orchestrator) runImport(ctx context.Context, spec *MigrationSpec, state *MigrationState, _ *models.DiscoveryResult) error {
	for index := range state.Workloads {
		workload := &state.Workloads[index]
		workload.Phase = PhaseImport

		overrides, _ := mergedOverridesForVM(workload.VM, spec.Workloads)
		targetHost := overrides.TargetHost
		if targetHost == "" {
			targetHost = spec.Target.DefaultHost
		}
		targetStorage := overrides.TargetStorage
		if targetStorage == "" {
			targetStorage = spec.Target.DefaultStorage
		}

		if importer, ok := o.targetConnector.(vmImporter); ok {
			vmID, err := importer.CreateVM(ctx, workload.VM, workload.ConvertedDiskPaths, targetHost, targetStorage)
			if err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("import %s: %w", workload.VM.Name, err)
			}
			workload.TargetVMID = vmID
		} else {
			workload.TargetVMID = workload.VM.ID + "-target"
		}

		o.emit(PhaseImport, workload.VM.Name, "created target workload", 60)
	}

	return nil
}

func (o *Orchestrator) runConfigure(ctx context.Context, spec *MigrationSpec, state *MigrationState, targetResult *models.DiscoveryResult) error {
	for index := range state.Workloads {
		workload := &state.Workloads[index]
		workload.Phase = PhaseConfigure

		overrides, _ := mergedOverridesForVM(workload.VM, spec.Workloads)
		mapper := NewNetworkMapper(overrides.NetworkMap, targetResult.Networks)
		mappedNICs, errs := mapper.MapAllNICs(workload.VM.NICs)
		if len(errs) > 0 {
			workload.Error = errs[0].Error()
			return fmt.Errorf("configure %s: %w", workload.VM.Name, errors.Join(errs...))
		}

		if configurer, ok := o.targetConnector.(vmNetworkConfigurer); ok {
			if err := configurer.ConfigureVMNetworks(ctx, workload.TargetVMID, mappedNICs); err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("configure %s: %w", workload.VM.Name, err)
			}
		}

		o.emit(PhaseConfigure, workload.VM.Name, "applied network mappings", 75)
	}

	return nil
}

func (o *Orchestrator) runVerify(ctx context.Context, spec *MigrationSpec, state *MigrationState, _ *models.DiscoveryResult) error {
	if !spec.Options.VerifyBoot {
		for index := range state.Workloads {
			state.Workloads[index].Phase = PhaseVerify
		}
		return nil
	}

	for index := range state.Workloads {
		workload := &state.Workloads[index]
		workload.Phase = PhaseVerify

		controller, hasPower := o.targetConnector.(vmPowerController)
		if hasPower {
			if err := controller.PowerOnVM(ctx, workload.TargetVMID); err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("verify %s: power on: %w", workload.VM.Name, err)
			}
		}
		if verifier, ok := o.targetConnector.(vmVerifier); ok {
			if err := verifier.VerifyVM(ctx, workload.TargetVMID); err != nil {
				workload.Error = err.Error()
				return fmt.Errorf("verify %s: %w", workload.VM.Name, err)
			}
		}

		o.emit(PhaseVerify, workload.VM.Name, "verified target workload", 90)
	}

	return nil
}

func (o *Orchestrator) setPhase(ctx context.Context, state *MigrationState, phase MigrationPhase) error {
	state.Phase = phase
	state.UpdatedAt = nowUTC()
	markCheckpointRunning(state, phase, "phase started")
	return persistMigrationState(ctx, o.store, state)
}

func (o *Orchestrator) failState(ctx context.Context, state *MigrationState, phase MigrationPhase, err error) (*MigrationState, error) {
	state.Phase = PhaseFailed
	state.UpdatedAt = nowUTC()
	if err != nil {
		state.Errors = append(state.Errors, err.Error())
	}
	markCheckpointFailed(state, phase, err)
	_ = persistMigrationState(ctx, o.store, state)
	o.emit(phase, "", "migration failed", 100)
	return state, err
}

func (o *Orchestrator) emit(phase MigrationPhase, vm, message string, progress int) {
	log.Printf("component=migrate phase=%s vm=%s progress=%d message=%q", phase, vm, progress, message)
	if o.onEvent == nil {
		return
	}

	o.onEvent(MigrationEvent{
		Timestamp: nowUTC(),
		Phase:     phase,
		VM:        vm,
		Message:   message,
		Progress:  progress,
	})
}

func defaultSourceDiskPaths(migrationID string, vm models.VirtualMachine) []string {
	paths := make([]string, 0, len(vm.Disks))
	for index := range vm.Disks {
		paths = append(paths, filepath.Join("artifacts", migrationID, vm.ID, fmt.Sprintf("disk-%d%s", index+1, diskExtensionForPlatform(vm.Platform))))
	}
	return paths
}

func diskExtensionForPlatform(platform models.Platform) string {
	switch platform {
	case models.PlatformVMware:
		return ".vmdk"
	case models.PlatformHyperV:
		return ".vhdx"
	case models.PlatformProxmox, models.PlatformKVM:
		return ".qcow2"
	default:
		return ".raw"
	}
}
