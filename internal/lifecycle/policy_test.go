package lifecycle

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestPolicyEngine_Evaluate_PlacementEnforceWarn(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(newLifecycleCostEngine())
	engine.AddPolicy(Policy{
		Name:     "cluster-required",
		Type:     PlacementPolicy,
		Severity: PolicySeverityEnforce,
		Rules: []PolicyRule{
			{Field: "cluster", Operator: "equals", Value: "Cluster-A"},
		},
	})
	engine.AddPolicy(Policy{
		Name:     "folder-warning",
		Type:     PlacementPolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "folder", Operator: "matches", Value: "^/Prod/"},
		},
	})

	report, err := engine.Evaluate(sampleLifecycleInventory())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if report.NonCompliantVMs != 1 || len(report.Violations) != 2 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestPolicyEngine_Evaluate_ComplianceTagCheck(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(nil)
	engine.AddPolicy(Policy{
		Name:     "environment-tag",
		Type:     CompliancePolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "tag:env", Operator: "equals", Value: "production"},
		},
	})

	report, err := engine.Evaluate(sampleLifecycleInventory())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(report.Violations) != 1 || report.Violations[0].VM.Name != "db-01" {
		t.Fatalf("unexpected violations: %#v", report.Violations)
	}
}

func TestPolicyEngine_Evaluate_CostThreshold(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(newLifecycleCostEngine())
	engine.AddPolicy(Policy{
		Name:     "monthly-cost-cap",
		Type:     CostPolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "cost", Operator: "less-than", Value: 150},
		},
	})

	report, err := engine.Evaluate(sampleLifecycleInventory())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(report.Violations) != 1 || report.Violations[0].VM.Name != "db-01" {
		t.Fatalf("unexpected violations: %#v", report.Violations)
	}
}

func TestPolicyEngine_Evaluate_AllCompliant(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(nil)
	engine.AddPolicy(Policy{
		Name:     "platform-allowed",
		Type:     PlacementPolicy,
		Severity: PolicySeverityInfo,
		Rules: []PolicyRule{
			{Field: "platform", Operator: "in", Value: []string{"vmware", "proxmox"}},
		},
	})

	report, err := engine.Evaluate(&models.DiscoveryResult{
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
		VMs: []models.VirtualMachine{
			sampleLifecycleVM(),
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if report.NonCompliantVMs != 0 || report.CompliantVMs != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestPolicyEngine_Simulate_AddsCandidatePolicy(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(nil)
	engine.AddPolicy(Policy{
		Name:     "platform-allowed",
		Type:     PlacementPolicy,
		Severity: PolicySeverityInfo,
		Rules: []PolicyRule{
			{Field: "platform", Operator: "equals", Value: "vmware"},
		},
	})

	report, err := engine.Simulate(sampleLifecycleInventory(), Policy{
		Name:     "folder-required",
		Type:     CompliancePolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "folder", Operator: "matches", Value: "^/Prod/"},
		},
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if len(report.Policies) != 2 || report.NonCompliantVMs == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestPolicyEngine_LoadPolicies_Directory_Expected(t *testing.T) {
	t.Parallel()

	engine := NewPolicyEngine(newLifecycleCostEngine())
	if err := engine.LoadPolicies(filepath.Join("..", "..", "configs", "policies")); err != nil {
		t.Fatalf("LoadPolicies() error = %v", err)
	}
	if len(engine.policies) < 4 {
		t.Fatalf("len(policies) = %d, want >= 4", len(engine.policies))
	}
}

func sampleLifecycleInventory() *models.DiscoveryResult {
	return &models.DiscoveryResult{
		Source:       "lab",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
		VMs: []models.VirtualMachine{
			sampleLifecycleVM(),
			{
				ID:         "vm-2",
				Name:       "db-01",
				Platform:   models.PlatformVMware,
				PowerState: models.PowerOn,
				CPUCount:   16,
				MemoryMB:   32768,
				Cluster:    "Cluster-B",
				Folder:     "/Legacy/DB",
				Tags: map[string]string{
					"env": "staging",
				},
				Disks: []models.Disk{
					{ID: "disk-2", SizeMB: 102400},
				},
			},
		},
	}
}
