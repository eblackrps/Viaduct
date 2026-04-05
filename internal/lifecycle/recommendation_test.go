package lifecycle

import (
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestPolicyEngine_EvaluateWithWaivers_SuppressesMatchingViolation(t *testing.T) {
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

	report, err := engine.EvaluateWithWaivers(sampleLifecycleInventory(), []PolicyWaiver{
		{
			PolicyName: "environment-tag",
			VMID:       "vm-2",
			Reason:     "approved staging exception",
			ExpiresAt:  time.Now().UTC().Add(24 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("EvaluateWithWaivers() error = %v", err)
	}
	if len(report.Violations) != 0 || report.WaivedViolations != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestRecommendationEngine_Generate_ReturnsPlacementAndPolicyGuidance(t *testing.T) {
	t.Parallel()

	engine := NewRecommendationEngine(newLifecycleCostEngine(), func() *PolicyEngine {
		policyEngine := NewPolicyEngine(newLifecycleCostEngine())
		policyEngine.AddPolicy(Policy{
			Name:     "environment-tag",
			Type:     CompliancePolicy,
			Severity: PolicySeverityWarn,
			Rules: []PolicyRule{
				{Field: "tag:env", Operator: "equals", Value: "production"},
			},
		})
		return policyEngine
	}())

	report, err := engine.Generate(sampleLifecycleInventory(), &DriftReport{
		Events: []DriftEvent{
			{
				Type:     DriftModified,
				VM:       sampleLifecycleVM(),
				Field:    "cpu_count",
				Severity: DriftSeverityWarning,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(report.Recommendations) < 3 {
		t.Fatalf("len(Recommendations) = %d, want at least 3", len(report.Recommendations))
	}
}

func TestRecommendationEngine_Simulate_TargetPlatformShift_Expected(t *testing.T) {
	t.Parallel()

	policyEngine := NewPolicyEngine(newLifecycleCostEngine())
	policyEngine.AddPolicy(Policy{
		Name:     "platform-allowed",
		Type:     PlacementPolicy,
		Severity: PolicySeverityWarn,
		Rules: []PolicyRule{
			{Field: "platform", Operator: "in", Value: []string{"vmware", "proxmox"}},
		},
	})

	engine := NewRecommendationEngine(newLifecycleCostEngine(), policyEngine)
	result, err := engine.Simulate(sampleLifecycleInventory(), SimulationRequest{
		TargetPlatform: models.PlatformProxmox,
		VMIDs:          []string{"vm-1"},
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if result.MovedVMs != 1 {
		t.Fatalf("MovedVMs = %d, want 1", result.MovedVMs)
	}
	if result.SimulatedInventory == nil || result.SimulatedInventory.VMs[0].Platform != models.PlatformProxmox {
		t.Fatalf("unexpected simulated inventory: %#v", result.SimulatedInventory)
	}
	if result.RecommendationReport == nil {
		t.Fatal("RecommendationReport = nil, want populated report")
	}
}
