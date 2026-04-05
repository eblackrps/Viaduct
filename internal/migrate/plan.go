package migrate

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

// CheckpointStatus identifies the lifecycle of a persisted migration checkpoint.
type CheckpointStatus string

const (
	// CheckpointPending indicates the phase has not started yet.
	CheckpointPending CheckpointStatus = "pending"
	// CheckpointRunning indicates the phase is currently executing.
	CheckpointRunning CheckpointStatus = "running"
	// CheckpointCompleted indicates the phase finished successfully.
	CheckpointCompleted CheckpointStatus = "completed"
	// CheckpointFailed indicates the phase ended in failure.
	CheckpointFailed CheckpointStatus = "failed"
)

// PlannedWorkload captures wave-planning metadata for a matched VM.
type PlannedWorkload struct {
	VMID          string            `json:"vm_id" yaml:"vm_id"`
	Name          string            `json:"name" yaml:"name"`
	Dependencies  []string          `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	TargetHost    string            `json:"target_host,omitempty" yaml:"target_host,omitempty"`
	TargetStorage string            `json:"target_storage,omitempty" yaml:"target_storage,omitempty"`
	NetworkMap    map[string]string `json:"network_map,omitempty" yaml:"network_map,omitempty"`
}

// MigrationWave defines a dependency-safe execution batch.
type MigrationWave struct {
	Index      int               `json:"index" yaml:"index"`
	Reason     string            `json:"reason" yaml:"reason"`
	Workloads  []PlannedWorkload `json:"workloads" yaml:"workloads"`
	Dependency bool              `json:"dependency_aware" yaml:"dependency_aware"`
}

// MigrationPlan describes the planned execution order for a migration.
type MigrationPlan struct {
	GeneratedAt       time.Time       `json:"generated_at" yaml:"generated_at"`
	TotalWorkloads    int             `json:"total_workloads" yaml:"total_workloads"`
	Window            ExecutionWindow `json:"window,omitempty" yaml:"window,omitempty"`
	RequiresApproval  bool            `json:"requires_approval,omitempty" yaml:"requires_approval,omitempty"`
	ApprovalSatisfied bool            `json:"approval_satisfied" yaml:"approval_satisfied"`
	WaveStrategy      WaveStrategy    `json:"wave_strategy" yaml:"wave_strategy"`
	Waves             []MigrationWave `json:"waves" yaml:"waves"`
}

// MigrationCheckpoint stores resumable progress for a single migration phase.
type MigrationCheckpoint struct {
	Phase       MigrationPhase   `json:"phase" yaml:"phase"`
	Status      CheckpointStatus `json:"status" yaml:"status"`
	StartedAt   time.Time        `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	CompletedAt time.Time        `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Message     string           `json:"message,omitempty" yaml:"message,omitempty"`
	Diagnostics []string         `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
}

// BuildExecutionPlan builds a dependency-aware execution plan from matched workloads.
func BuildExecutionPlan(spec *MigrationSpec, sourceVMs []models.VirtualMachine) (*MigrationPlan, error) {
	if spec == nil {
		return nil, fmt.Errorf("build execution plan: spec is nil")
	}

	workloads := make([]PlannedWorkload, 0)
	for _, vm := range MatchWorkloads(sourceVMs, spec.Workloads) {
		overrides, ok := mergedOverridesForVM(vm, spec.Workloads)
		if !ok {
			continue
		}

		targetHost := overrides.TargetHost
		if targetHost == "" {
			targetHost = spec.Target.DefaultHost
		}
		targetStorage := overrides.TargetStorage
		if targetStorage == "" {
			targetStorage = spec.Target.DefaultStorage
		}

		workloads = append(workloads, PlannedWorkload{
			VMID:          workloadKey(vm),
			Name:          vm.Name,
			Dependencies:  append([]string(nil), overrides.Dependencies...),
			TargetHost:    targetHost,
			TargetStorage: targetStorage,
			NetworkMap:    copyStringMap(overrides.NetworkMap),
		})
	}

	plan := &MigrationPlan{
		GeneratedAt:       nowUTC(),
		TotalWorkloads:    len(workloads),
		Window:            spec.Options.Window,
		RequiresApproval:  spec.Options.Approval.Required,
		ApprovalSatisfied: spec.Options.Approval.Approved(),
		WaveStrategy:      normalizedWaveStrategy(spec.Options),
		Waves:             make([]MigrationWave, 0),
	}
	if !plan.WaveStrategy.DependencyAware {
		for _, workload := range workloads {
			if len(workload.Dependencies) > 0 {
				plan.WaveStrategy.DependencyAware = true
				break
			}
		}
	}
	if len(workloads) == 0 {
		return plan, nil
	}

	sort.Slice(workloads, func(i, j int) bool {
		return strings.ToLower(workloads[i].Name) < strings.ToLower(workloads[j].Name)
	})

	if !plan.WaveStrategy.DependencyAware {
		for index, batch := range chunkWorkloads(workloads, plan.WaveStrategy.Size) {
			plan.Waves = append(plan.Waves, MigrationWave{
				Index:      index + 1,
				Reason:     "batch_size",
				Workloads:  batch,
				Dependency: false,
			})
		}
		return plan, nil
	}

	waves, err := buildDependencyAwareWaves(workloads, plan.WaveStrategy.Size)
	if err != nil {
		return nil, fmt.Errorf("build execution plan: %w", err)
	}
	plan.Waves = waves
	return plan, nil
}

func buildDependencyAwareWaves(workloads []PlannedWorkload, waveSize int) ([]MigrationWave, error) {
	nodes := make(map[string]PlannedWorkload, len(workloads))
	nameIndex := make(map[string]string, len(workloads)*2)
	indegree := make(map[string]int, len(workloads))
	reverse := make(map[string][]string, len(workloads))

	for _, workload := range workloads {
		nodes[workload.VMID] = workload
		nameIndex[strings.ToLower(workload.VMID)] = workload.VMID
		nameIndex[strings.ToLower(workload.Name)] = workload.VMID
		indegree[workload.VMID] = 0
	}

	for _, workload := range workloads {
		for _, dependency := range workload.Dependencies {
			dependencyKey, ok := nameIndex[strings.ToLower(strings.TrimSpace(dependency))]
			if !ok {
				return nil, fmt.Errorf("workload %s depends on unknown workload %q", workload.Name, dependency)
			}
			if dependencyKey == workload.VMID {
				return nil, fmt.Errorf("workload %s depends on itself", workload.Name)
			}
			indegree[workload.VMID]++
			reverse[dependencyKey] = append(reverse[dependencyKey], workload.VMID)
		}
	}

	available := make([]string, 0)
	for key, degree := range indegree {
		if degree == 0 {
			available = append(available, key)
		}
	}
	sort.Slice(available, func(i, j int) bool {
		return strings.ToLower(nodes[available[i]].Name) < strings.ToLower(nodes[available[j]].Name)
	})

	waves := make([]MigrationWave, 0)
	processed := 0

	for len(available) > 0 {
		count := waveSize
		if count <= 0 || count > len(available) {
			count = len(available)
		}

		current := append([]string(nil), available[:count]...)
		available = append([]string(nil), available[count:]...)

		wave := MigrationWave{
			Index:      len(waves) + 1,
			Reason:     "dependency_wave",
			Workloads:  make([]PlannedWorkload, 0, len(current)),
			Dependency: true,
		}
		for _, key := range current {
			wave.Workloads = append(wave.Workloads, nodes[key])
			processed++
		}
		waves = append(waves, wave)

		newlyReady := make([]string, 0)
		for _, key := range current {
			for _, dependent := range reverse[key] {
				indegree[dependent]--
				if indegree[dependent] == 0 {
					newlyReady = append(newlyReady, dependent)
				}
			}
		}
		sort.Slice(newlyReady, func(i, j int) bool {
			return strings.ToLower(nodes[newlyReady[i]].Name) < strings.ToLower(nodes[newlyReady[j]].Name)
		})
		available = append(available, newlyReady...)
		sort.Slice(available, func(i, j int) bool {
			return strings.ToLower(nodes[available[i]].Name) < strings.ToLower(nodes[available[j]].Name)
		})
	}

	if processed != len(workloads) {
		return nil, fmt.Errorf("dependency cycle detected in migration wave planning")
	}

	return waves, nil
}

func chunkWorkloads(workloads []PlannedWorkload, waveSize int) [][]PlannedWorkload {
	if waveSize <= 0 {
		waveSize = len(workloads)
	}

	items := make([][]PlannedWorkload, 0, (len(workloads)+waveSize-1)/waveSize)
	for start := 0; start < len(workloads); start += waveSize {
		end := start + waveSize
		if end > len(workloads) {
			end = len(workloads)
		}
		batch := append([]PlannedWorkload(nil), workloads[start:end]...)
		items = append(items, batch)
	}
	return items
}

func initializeCheckpoints() []MigrationCheckpoint {
	phases := []MigrationPhase{PhasePlan, PhaseExport, PhaseConvert, PhaseImport, PhaseConfigure, PhaseVerify}
	checkpoints := make([]MigrationCheckpoint, 0, len(phases))
	for _, phase := range phases {
		checkpoints = append(checkpoints, MigrationCheckpoint{
			Phase:  phase,
			Status: CheckpointPending,
		})
	}
	return checkpoints
}

func markCheckpointRunning(state *MigrationState, phase MigrationPhase, message string) {
	updateCheckpoint(state, phase, func(checkpoint *MigrationCheckpoint) {
		checkpoint.Status = CheckpointRunning
		if checkpoint.StartedAt.IsZero() {
			checkpoint.StartedAt = nowUTC()
		}
		checkpoint.Message = message
	})
}

func markCheckpointCompleted(state *MigrationState, phase MigrationPhase, message string) {
	updateCheckpoint(state, phase, func(checkpoint *MigrationCheckpoint) {
		if checkpoint.StartedAt.IsZero() {
			checkpoint.StartedAt = nowUTC()
		}
		checkpoint.Status = CheckpointCompleted
		checkpoint.CompletedAt = nowUTC()
		checkpoint.Message = message
	})
}

func markCheckpointFailed(state *MigrationState, phase MigrationPhase, err error) {
	updateCheckpoint(state, phase, func(checkpoint *MigrationCheckpoint) {
		if checkpoint.StartedAt.IsZero() {
			checkpoint.StartedAt = nowUTC()
		}
		checkpoint.Status = CheckpointFailed
		checkpoint.CompletedAt = nowUTC()
		if err != nil {
			checkpoint.Diagnostics = append(checkpoint.Diagnostics, err.Error())
			checkpoint.Message = err.Error()
		}
	})
}

func checkpointStatus(state *MigrationState, phase MigrationPhase) CheckpointStatus {
	for _, checkpoint := range state.Checkpoints {
		if checkpoint.Phase == phase {
			return checkpoint.Status
		}
	}
	return CheckpointPending
}

func nextPhaseStartIndex(state *MigrationState) int {
	for index, phase := range []MigrationPhase{PhaseExport, PhaseConvert, PhaseImport, PhaseConfigure, PhaseVerify} {
		switch checkpointStatus(state, phase) {
		case CheckpointCompleted:
			continue
		default:
			return index
		}
	}
	return -1
}

func updateCheckpoint(state *MigrationState, phase MigrationPhase, update func(*MigrationCheckpoint)) {
	if state == nil {
		return
	}
	if len(state.Checkpoints) == 0 {
		state.Checkpoints = initializeCheckpoints()
	}
	for index := range state.Checkpoints {
		if state.Checkpoints[index].Phase != phase {
			continue
		}
		update(&state.Checkpoints[index])
		return
	}
	state.Checkpoints = append(state.Checkpoints, MigrationCheckpoint{Phase: phase, Status: CheckpointPending})
	update(&state.Checkpoints[len(state.Checkpoints)-1])
}

func normalizedWaveStrategy(options MigrationOptions) WaveStrategy {
	strategy := options.Waves
	if strategy.Size <= 0 {
		strategy.Size = options.Parallel
	}
	if strategy.Size <= 0 {
		strategy.Size = 1
	}
	return strategy
}

func reorderWorkloadsByPlan(workloads []WorkloadMigration, plan *MigrationPlan) []WorkloadMigration {
	if len(workloads) == 0 || plan == nil || len(plan.Waves) == 0 {
		return workloads
	}

	workloadIndex := make(map[string]WorkloadMigration, len(workloads))
	for _, workload := range workloads {
		workloadIndex[workloadKey(workload.VM)] = workload
	}

	ordered := make([]WorkloadMigration, 0, len(workloads))
	seen := make(map[string]struct{}, len(workloads))
	for _, wave := range plan.Waves {
		for _, item := range wave.Workloads {
			workload, ok := workloadIndex[item.VMID]
			if !ok {
				continue
			}
			if _, ok := seen[item.VMID]; ok {
				continue
			}
			seen[item.VMID] = struct{}{}
			ordered = append(ordered, workload)
		}
	}
	for _, workload := range workloads {
		key := workloadKey(workload.VM)
		if _, ok := seen[key]; ok {
			continue
		}
		ordered = append(ordered, workload)
	}
	return ordered
}

func workloadKey(vm models.VirtualMachine) string {
	if strings.TrimSpace(vm.ID) != "" {
		return vm.ID
	}
	return vm.Name
}
