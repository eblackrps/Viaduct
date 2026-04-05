package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestDriftDetector_Compare_NoChanges(t *testing.T) {
	t.Parallel()

	report := compareDrift(t, sampleLifecycleInventory(), sampleLifecycleInventory(), DriftConfig{})
	if report.AddedVMs != 0 || report.RemovedVMs != 0 || report.ModifiedVMs != 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestDriftDetector_Compare_NewVM(t *testing.T) {
	t.Parallel()

	current := sampleLifecycleInventory()
	current.VMs = append(current.VMs, models.VirtualMachine{
		ID:         "vm-3",
		Name:       "app-01",
		Platform:   models.PlatformVMware,
		PowerState: models.PowerOn,
	})

	report := compareDrift(t, sampleLifecycleInventory(), current, DriftConfig{})
	if report.AddedVMs != 1 {
		t.Fatalf("AddedVMs = %d, want 1", report.AddedVMs)
	}
}

func TestDriftDetector_Compare_RemovedVM(t *testing.T) {
	t.Parallel()

	current := sampleLifecycleInventory()
	current.VMs = current.VMs[:1]

	report := compareDrift(t, sampleLifecycleInventory(), current, DriftConfig{})
	if report.RemovedVMs != 1 {
		t.Fatalf("RemovedVMs = %d, want 1", report.RemovedVMs)
	}
}

func TestDriftDetector_Compare_ModifiedCPU(t *testing.T) {
	t.Parallel()

	current := sampleLifecycleInventory()
	current.VMs[0].CPUCount = current.VMs[0].CPUCount + 4

	report := compareDrift(t, sampleLifecycleInventory(), current, DriftConfig{})
	if report.ModifiedVMs != 1 {
		t.Fatalf("ModifiedVMs = %d, want 1", report.ModifiedVMs)
	}
}

func TestDriftDetector_Compare_MemoryThresholdIgnored(t *testing.T) {
	t.Parallel()

	current := sampleLifecycleInventory()
	current.VMs[0].MemoryMB = current.VMs[0].MemoryMB + 128

	report := compareDrift(t, sampleLifecycleInventory(), current, DriftConfig{MemoryThresholdMB: 256})
	if report.ModifiedVMs != 0 {
		t.Fatalf("ModifiedVMs = %d, want 0", report.ModifiedVMs)
	}
}

func TestDriftDetector_Compare_PolicyDrift(t *testing.T) {
	t.Parallel()

	policyEngine := NewPolicyEngine(nil)
	policyEngine.AddPolicy(Policy{
		Name:     "environment-tag",
		Type:     CompliancePolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "tag:env", Operator: "equals", Value: "production"},
		},
	})

	stateStore := store.NewMemoryStore()
	ctx := context.Background()
	baselineID, err := stateStore.SaveDiscovery(ctx, store.DefaultTenantID, sampleLifecycleInventory())
	if err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	detector := NewDriftDetector(stateStore, policyEngine, DriftConfig{})
	report, err := detector.Compare(ctx, baselineID, sampleLifecycleInventory())
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if report.PolicyDrifts != 1 {
		t.Fatalf("PolicyDrifts = %d, want 1", report.PolicyDrifts)
	}
}

func TestDriftDetector_Compare_MultipleChanges(t *testing.T) {
	t.Parallel()

	current := sampleLifecycleInventory()
	current.VMs[0].CPUCount = current.VMs[0].CPUCount + 6
	current.VMs[1].Host = "esx-02"
	current.VMs = current.VMs[:1]

	report := compareDrift(t, sampleLifecycleInventory(), current, DriftConfig{})
	if report.RemovedVMs != 1 || report.ModifiedVMs != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func compareDrift(t *testing.T, baseline, current *models.DiscoveryResult, cfg DriftConfig) *DriftReport {
	t.Helper()

	baseline.DiscoveredAt = time.Date(2026, time.April, 4, 9, 0, 0, 0, time.UTC)
	current.DiscoveredAt = time.Date(2026, time.April, 4, 10, 0, 0, 0, time.UTC)

	stateStore := store.NewMemoryStore()
	ctx := context.Background()
	baselineID, err := stateStore.SaveDiscovery(ctx, store.DefaultTenantID, baseline)
	if err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	detector := NewDriftDetector(stateStore, nil, cfg)
	report, err := detector.Compare(ctx, baselineID, current)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	return report
}
