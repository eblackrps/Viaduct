package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors/veeam"
	"github.com/eblackrps/viaduct/internal/lifecycle"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestCostModeling_FleetComparison(t *testing.T) {
	t.Parallel()

	engine := lifecycle.NewCostEngine()
	engine.AddProfile(lifecycle.CostProfile{
		Platform:                models.PlatformVMware,
		Name:                    "VMware",
		CPUCostPerCoreMonth:     12,
		MemoryCostPerGBMonth:    4,
		StorageCostPerGBMonth:   0.5,
		LicenseCostPerCoreMonth: 8,
		LicenseModel:            lifecycle.LicenseModelPerCore,
	})
	engine.AddProfile(lifecycle.CostProfile{
		Platform:              models.PlatformProxmox,
		Name:                  "Proxmox",
		CPUCostPerCoreMonth:   6,
		MemoryCostPerGBMonth:  2,
		StorageCostPerGBMonth: 0.4,
		LicenseModel:          lifecycle.LicenseModelNone,
	})

	fleet, err := engine.CalculateFleetCost(models.PlatformProxmox, lifecycleInventory().VMs)
	if err != nil {
		t.Fatalf("CalculateFleetCost() error = %v", err)
	}
	if fleet.TotalMonthly <= 0 || len(fleet.VMCosts) != len(lifecycleInventory().VMs) {
		t.Fatalf("unexpected fleet result: %#v", fleet)
	}
}

func TestPolicyEnforcement_FullWorkflow(t *testing.T) {
	t.Parallel()

	engine := lifecycle.NewPolicyEngine(nil)
	engine.AddPolicy(lifecycle.Policy{
		Name:     "env-tag",
		Type:     lifecycle.CompliancePolicy,
		Severity: lifecycle.PolicySeverityWarn,
		Rules: []lifecycle.PolicyRule{
			{Field: "tag:env", Operator: "equals", Value: "production"},
		},
	})

	report, err := engine.Evaluate(lifecycleInventory())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if report.NonCompliantVMs != 1 {
		t.Fatalf("NonCompliantVMs = %d, want 1", report.NonCompliantVMs)
	}
}

func TestDriftDetection_BaselineComparison(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	baseline := lifecycleInventory()
	baselineID, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, baseline)
	if err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	current := lifecycleInventory()
	current.VMs[0].CPUCount = current.VMs[0].CPUCount + 4

	detector := lifecycle.NewDriftDetector(stateStore, nil, lifecycle.DriftConfig{})
	report, err := detector.Compare(context.Background(), baselineID, current)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if report.ModifiedVMs != 1 {
		t.Fatalf("ModifiedVMs = %d, want 1", report.ModifiedVMs)
	}
}

func TestBackupJobPortability_FullWorkflow(t *testing.T) {
	t.Parallel()

	manager, cleanup := newLifecyclePortabilityManager(t)
	defer cleanup()

	plan, err := manager.PlanJobMigration(context.Background(), lifecycleInventory().VMs[0], models.VirtualMachine{
		ID:   "vm-target",
		Name: "web-01-target",
	}, map[string]string{"primary-repo": "target-repo"})
	if err != nil {
		t.Fatalf("PlanJobMigration() error = %v", err)
	}
	if len(plan.Jobs) != 1 {
		t.Fatalf("len(plan.Jobs) = %d, want 1", len(plan.Jobs))
	}

	result, err := manager.ExecuteJobMigration(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecuteJobMigration() error = %v", err)
	}
	if len(result.CreatedJobs) != 1 {
		t.Fatalf("len(result.CreatedJobs) = %d, want 1", len(result.CreatedJobs))
	}
}

func TestLifecycle_CombinedWorkflow(t *testing.T) {
	t.Parallel()

	inventory := lifecycleInventory()
	stateStore := store.NewMemoryStore()
	baselineID, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, inventory)
	if err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	costEngine := lifecycle.NewCostEngine()
	costEngine.AddProfile(lifecycle.CostProfile{
		Platform:                models.PlatformVMware,
		Name:                    "VMware",
		CPUCostPerCoreMonth:     12,
		MemoryCostPerGBMonth:    4,
		StorageCostPerGBMonth:   0.5,
		LicenseCostPerCoreMonth: 8,
		LicenseModel:            lifecycle.LicenseModelPerCore,
	})
	costEngine.AddProfile(lifecycle.CostProfile{
		Platform:              models.PlatformProxmox,
		Name:                  "Proxmox",
		CPUCostPerCoreMonth:   6,
		MemoryCostPerGBMonth:  2,
		StorageCostPerGBMonth: 0.4,
		LicenseModel:          lifecycle.LicenseModelNone,
	})

	policyEngine := lifecycle.NewPolicyEngine(costEngine)
	policyEngine.AddPolicy(lifecycle.Policy{
		Name:     "cost-cap",
		Type:     lifecycle.CostPolicy,
		Severity: lifecycle.PolicySeverityWarn,
		Rules: []lifecycle.PolicyRule{
			{Field: "cost", Operator: "less-than", Value: 100},
		},
	})

	driftDetector := lifecycle.NewDriftDetector(stateStore, policyEngine, lifecycle.DriftConfig{})
	current := lifecycleInventory()
	current.VMs[1].MemoryMB = current.VMs[1].MemoryMB + 1024

	driftReport, err := driftDetector.Compare(context.Background(), baselineID, current)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if driftReport.ModifiedVMs == 0 {
		t.Fatalf("ModifiedVMs = %d, want > 0", driftReport.ModifiedVMs)
	}
	if driftReport.PolicyDrifts == 0 {
		t.Fatalf("PolicyDrifts = %d, want > 0", driftReport.PolicyDrifts)
	}
}

func TestLifecycle_RecommendationAndSimulation_Workflow(t *testing.T) {
	t.Parallel()

	inventory := lifecycleInventory()
	costEngine := lifecycle.NewCostEngine()
	costEngine.AddProfile(lifecycle.CostProfile{
		Platform:                models.PlatformVMware,
		Name:                    "VMware",
		CPUCostPerCoreMonth:     12,
		MemoryCostPerGBMonth:    4,
		StorageCostPerGBMonth:   0.5,
		LicenseCostPerCoreMonth: 8,
		LicenseModel:            lifecycle.LicenseModelPerCore,
	})
	costEngine.AddProfile(lifecycle.CostProfile{
		Platform:              models.PlatformProxmox,
		Name:                  "Proxmox",
		CPUCostPerCoreMonth:   6,
		MemoryCostPerGBMonth:  2,
		StorageCostPerGBMonth: 0.4,
		LicenseModel:          lifecycle.LicenseModelNone,
	})

	policyEngine := lifecycle.NewPolicyEngine(costEngine)
	policyEngine.AddPolicy(lifecycle.Policy{
		Name:     "environment-tag",
		Type:     lifecycle.CompliancePolicy,
		Severity: lifecycle.PolicySeverityWarn,
		Rules: []lifecycle.PolicyRule{
			{Field: "tag:env", Operator: "equals", Value: "production"},
		},
	})

	recommendationEngine := lifecycle.NewRecommendationEngine(costEngine, policyEngine)
	recommendations, err := recommendationEngine.Generate(inventory, nil, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(recommendations.Recommendations) == 0 {
		t.Fatal("len(Recommendations) = 0, want actionable guidance")
	}

	simulation, err := recommendationEngine.Simulate(inventory, lifecycle.SimulationRequest{
		TargetPlatform: models.PlatformProxmox,
		IncludeAll:     true,
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if simulation.MovedVMs != len(inventory.VMs) {
		t.Fatalf("MovedVMs = %d, want %d", simulation.MovedVMs, len(inventory.VMs))
	}
	if simulation.RecommendationReport == nil {
		t.Fatal("RecommendationReport = nil, want populated simulation guidance")
	}
}

func lifecycleInventory() *models.DiscoveryResult {
	return &models.DiscoveryResult{
		Source:       "lab",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
		VMs: []models.VirtualMachine{
			{
				ID:         "vm-1",
				Name:       "web-01",
				Platform:   models.PlatformVMware,
				PowerState: models.PowerOn,
				CPUCount:   4,
				MemoryMB:   4096,
				Cluster:    "Cluster-A",
				Folder:     "/Prod/Web",
				Tags: map[string]string{
					"env": "production",
				},
				Disks: []models.Disk{
					{ID: "disk-1", SizeMB: 10240},
				},
			},
			{
				ID:         "vm-2",
				Name:       "db-01",
				Platform:   models.PlatformVMware,
				PowerState: models.PowerOn,
				CPUCount:   12,
				MemoryMB:   16384,
				Cluster:    "Cluster-B",
				Folder:     "/Prod/DB",
				Tags: map[string]string{
					"env": "staging",
				},
				Disks: []models.Disk{
					{ID: "disk-2", SizeMB: 51200},
				},
			},
		},
	}
}

func newLifecyclePortabilityManager(t *testing.T) (*veeam.PortabilityManager, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs":
			writeLifecycleJSON(t, w, []map[string]interface{}{
				{"id": "job-1", "name": "Daily web-01", "targetRepo": "primary-repo", "retentionDays": 14, "protectedVMs": []string{"web-01"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/backupInfrastructure/repositories":
			writeLifecycleJSON(t, w, []map[string]interface{}{
				{"id": "repo-1", "name": "primary-repo", "type": "xfs", "capacityMB": 100000, "freeMB": 50000, "usedMB": 50000},
				{"id": "repo-2", "name": "target-repo", "type": "xfs", "capacityMB": 100000, "freeMB": 80000, "usedMB": 20000},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			writeLifecycleJSON(t, w, map[string]string{"id": "job-created-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/job-created-1/start":
			writeLifecycleJSON(t, w, map[string]string{"status": "started"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/jobs/job-created-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))

	client := veeam.NewVeeamClient(server.URL, true)
	return veeam.NewPortabilityManager(client), server.Close
}

func writeLifecycleJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
