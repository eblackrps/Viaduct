package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestOrchestrator_DryRun(t *testing.T) {
	t.Parallel()

	source := &mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()}
	target := &mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()}
	orchestrator := NewOrchestrator(source, target, store.NewMemoryStore(), nil)

	spec := sampleSpec()
	spec.Options.DryRun = true

	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Phase != PhasePlan {
		t.Fatalf("Phase = %s, want %s", state.Phase, PhasePlan)
	}
	if len(state.Workloads) != 2 {
		t.Fatalf("len(Workloads) = %d, want 2", len(state.Workloads))
	}
}

func TestOrchestrator_MatchWorkloads_GlobPattern(t *testing.T) {
	t.Parallel()

	vms := sampleVirtualMachines()
	selectors := []WorkloadSelector{{Match: MatchCriteria{NamePattern: "web-*"}}}
	matched := MatchWorkloads(vms, selectors)
	if len(matched) != 2 {
		t.Fatalf("len(MatchWorkloads()) = %d, want 2", len(matched))
	}
}

func TestOrchestrator_MatchWorkloads_TagFilter(t *testing.T) {
	t.Parallel()

	vms := sampleVirtualMachines()
	selectors := []WorkloadSelector{{Match: MatchCriteria{Tags: map[string]string{"env": "production"}}}}
	matched := MatchWorkloads(vms, selectors)
	if len(matched) != 2 {
		t.Fatalf("len(MatchWorkloads()) = %d, want 2", len(matched))
	}
}

func TestOrchestrator_MatchWorkloads_Exclude(t *testing.T) {
	t.Parallel()

	vms := sampleVirtualMachines()
	selectors := []WorkloadSelector{{Match: MatchCriteria{NamePattern: "web-*", Exclude: []string{"web-02"}}}}
	matched := MatchWorkloads(vms, selectors)
	if len(matched) != 1 || matched[0].Name != "web-01" {
		t.Fatalf("unexpected matched workloads: %#v", matched)
	}
}

func TestOrchestrator_PhaseTransitions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourceResult := sampleSourceResult()
	source := &mockMigrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: map[string][]string{},
	}
	for _, vm := range sourceResult.VMs[:2] {
		path := filepath.Join(tempDir, vm.ID+".vmdk")
		if err := os.WriteFile(path, []byte("source"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		source.exportedDisks[vm.ID] = []string{path}
	}

	target := &mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()}
	stateStore := store.NewMemoryStore()
	orchestrator := NewOrchestrator(source, target, stateStore, nil)
	orchestrator.convertDisk = fakeConvertDisk

	state, err := orchestrator.Execute(context.Background(), sampleSpec())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Phase != PhaseComplete {
		t.Fatalf("Phase = %s, want %s", state.Phase, PhaseComplete)
	}
	for _, workload := range state.Workloads {
		if workload.Phase != PhaseComplete {
			t.Fatalf("workload phase = %s, want %s", workload.Phase, PhaseComplete)
		}
	}
}

func TestOrchestrator_Execute_ApprovalRequiredBlocksExecution(t *testing.T) {
	t.Parallel()

	source := &mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()}
	target := &mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()}
	orchestrator := NewOrchestrator(source, target, store.NewMemoryStore(), nil)

	spec := sampleSpec()
	spec.Options.Approval = ApprovalGate{Required: true}

	state, err := orchestrator.Execute(context.Background(), spec)
	if err == nil {
		t.Fatal("Execute() error = nil, want approval gate failure")
	}
	if state == nil {
		t.Fatal("Execute() state = nil, want planned state")
	}
	if !state.PendingApproval {
		t.Fatal("PendingApproval = false, want true")
	}
	if state.Phase != PhasePlan {
		t.Fatalf("Phase = %s, want %s", state.Phase, PhasePlan)
	}
}

func TestOrchestrator_Execute_PersistsPlanAndCheckpoints(t *testing.T) {
	t.Parallel()

	source := &mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()}
	target := &mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()}
	orchestrator := NewOrchestrator(source, target, store.NewMemoryStore(), nil)

	spec := sampleSpec()
	spec.Options.DryRun = true
	spec.Workloads[0].Overrides.Dependencies = []string{"db-01"}
	spec.Workloads = append(spec.Workloads, WorkloadSelector{
		Match: MatchCriteria{NamePattern: "db-*"},
	})

	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Plan == nil || len(state.Plan.Waves) == 0 {
		t.Fatalf("Plan = %#v, want populated execution plan", state.Plan)
	}
	if checkpointStatus(state, PhasePlan) != CheckpointCompleted {
		t.Fatalf("plan checkpoint = %s, want %s", checkpointStatus(state, PhasePlan), CheckpointCompleted)
	}
}

func TestOrchestrator_Resume_SkipsCompletedPhases(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourceResult := sampleSourceResult()
	source := &mockMigrationConnector{
		platform:      sourceResult.Platform,
		result:        sourceResult,
		exportedDisks: map[string][]string{},
	}
	for _, vm := range sourceResult.VMs[:2] {
		path := filepath.Join(tempDir, vm.ID+".vmdk")
		if err := os.WriteFile(path, []byte("source"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		source.exportedDisks[vm.ID] = []string{path}
	}

	target := &mockMigrationConnector{
		platform:  sampleTargetResult().Platform,
		result:    sampleTargetResult(),
		createErr: fmt.Errorf("target create unavailable"),
	}
	stateStore := store.NewMemoryStore()
	orchestrator := NewOrchestrator(source, target, stateStore, nil)
	orchestrator.SetIDGenerator(func() string { return "resume-test" })
	orchestrator.SetDiskConverter(fakeConvertDisk)

	spec := sampleSpec()
	state, err := orchestrator.Execute(context.Background(), spec)
	if err == nil {
		t.Fatal("Execute() error = nil, want import failure")
	}
	if state == nil {
		t.Fatal("Execute() state = nil, want failed state")
	}
	if checkpointStatus(state, PhaseExport) != CheckpointCompleted || checkpointStatus(state, PhaseConvert) != CheckpointCompleted {
		t.Fatalf("unexpected checkpoints: %#v", state.Checkpoints)
	}

	target.createErr = nil
	resumed, err := orchestrator.Resume(context.Background(), state.ID, spec)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if resumed.Phase != PhaseComplete {
		t.Fatalf("Phase = %s, want %s", resumed.Phase, PhaseComplete)
	}
	if len(source.poweredOff) != len(resumed.Workloads) {
		t.Fatalf("power off calls = %d, want %d", len(source.poweredOff), len(resumed.Workloads))
	}
}

func TestOrchestrator_Execute_WindowNotOpenBlocksExecution(t *testing.T) {
	t.Parallel()

	source := &mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()}
	target := &mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()}
	orchestrator := NewOrchestrator(source, target, store.NewMemoryStore(), nil)

	spec := sampleSpec()
	spec.Options.Window = ExecutionWindow{NotBefore: time.Now().UTC().Add(90 * time.Minute)}

	state, err := orchestrator.Execute(context.Background(), spec)
	if err == nil {
		t.Fatal("Execute() error = nil, want execution window failure")
	}
	if state == nil || state.Phase != PhasePlan {
		t.Fatalf("state = %#v, want planned blocked state", state)
	}
}
