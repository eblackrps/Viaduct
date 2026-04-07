package veeam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestPortabilityManager_PlanJobMigration_SingleJob(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{
			"id":            "job-1",
			"name":          "Daily web-01",
			"type":          "vmware",
			"schedule":      "0 2 * * *",
			"targetRepo":    "primary-repo",
			"retentionDays": 14,
			"protectedVMs":  []string{"web-01"},
			"enabled":       true,
		},
	})
	defer cleanup()

	plan, err := manager.PlanJobMigration(context.Background(), sampleBackupSourceVM(), sampleBackupTargetVM(), map[string]string{
		"primary-repo": "target-repo",
	})
	if err != nil {
		t.Fatalf("PlanJobMigration() error = %v", err)
	}
	if len(plan.Jobs) != 1 || plan.Jobs[0].TargetRepo != "target-repo" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if len(plan.RepositoryCompatibility) != 1 || !plan.RepositoryCompatibility[0].Compatible {
		t.Fatalf("unexpected repository compatibility: %#v", plan.RepositoryCompatibility)
	}
}

func TestPortabilityManager_PlanJobMigration_MultipleJobs(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{"id": "job-1", "name": "Daily web-01", "targetRepo": "primary-repo", "retentionDays": 7, "protectedVMs": []string{"web-01"}},
		{"id": "job-2", "name": "Weekly web-01", "targetRepo": "primary-repo", "retentionDays": 30, "protectedVMs": []string{"web-01"}},
	})
	defer cleanup()

	plan, err := manager.PlanJobMigration(context.Background(), sampleBackupSourceVM(), sampleBackupTargetVM(), nil)
	if err != nil {
		t.Fatalf("PlanJobMigration() error = %v", err)
	}
	if len(plan.Jobs) != 2 {
		t.Fatalf("len(plan.Jobs) = %d, want 2", len(plan.Jobs))
	}
}

func TestPortabilityManager_PlanJobMigration_NoExistingJob(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{"id": "job-1", "name": "Daily db-01", "targetRepo": "primary-repo", "retentionDays": 7, "protectedVMs": []string{"db-01"}},
	})
	defer cleanup()

	plan, err := manager.PlanJobMigration(context.Background(), sampleBackupSourceVM(), sampleBackupTargetVM(), nil)
	if err != nil {
		t.Fatalf("PlanJobMigration() error = %v", err)
	}
	if len(plan.Jobs) != 0 {
		t.Fatalf("len(plan.Jobs) = %d, want 0", len(plan.Jobs))
	}
}

func TestPortabilityManager_ExecuteJobMigration_Success(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{"id": "job-1", "name": "Daily web-01", "targetRepo": "primary-repo", "retentionDays": 7, "protectedVMs": []string{"web-01"}},
	})
	defer cleanup()

	result, err := manager.ExecuteJobMigration(context.Background(), &JobMigrationPlan{
		Jobs: []BackupJobTemplate{
			{Name: "Daily web-01-target", TargetRepo: "primary-repo", ProtectedVMs: []string{"web-01-target"}},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteJobMigration() error = %v", err)
	}
	if len(result.CreatedJobs) != 1 || result.VerificationStatus[result.CreatedJobs[0]] != "verified" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestPortabilityManager_ExecuteJobMigration_VerificationFailureReturnsError_Expected(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			writePortabilityJSON(t, w, map[string]string{"id": "job-created-1"})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") && strings.HasSuffix(r.URL.Path, "/start"):
			http.Error(w, "verification failed", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewVeeamClient(server.URL, true)
	client.accessToken = "token"
	manager := NewPortabilityManager(client)

	result, err := manager.ExecuteJobMigration(context.Background(), &JobMigrationPlan{
		Jobs: []BackupJobTemplate{
			{Name: "Daily web-01-target", TargetRepo: "primary-repo", ProtectedVMs: []string{"web-01-target"}},
		},
	})
	if err == nil {
		t.Fatal("ExecuteJobMigration() error = nil, want verification failure")
	}
	if result == nil || len(result.CreatedJobs) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.VerificationStatus["job-created-1"] != "failed" {
		t.Fatalf("VerificationStatus = %#v, want failed job status", result.VerificationStatus)
	}
}

func TestPortabilityManager_RollbackJobMigration_DeletesCreatedJobs(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{})
	defer cleanup()

	result := &JobMigrationResult{CreatedJobs: []string{"job-created-1"}}
	if err := manager.RollbackJobMigration(context.Background(), result); err != nil {
		t.Fatalf("RollbackJobMigration() error = %v", err)
	}
}

func TestPortabilityManager_PlanJobMigration_RepositoryWarning(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{"id": "job-1", "name": "Daily web-01", "targetRepo": "archive-repo", "retentionDays": 7, "protectedVMs": []string{"web-01"}},
	})
	defer cleanup()

	plan, err := manager.PlanJobMigration(context.Background(), sampleBackupSourceVM(), sampleBackupTargetVM(), nil)
	if err != nil {
		t.Fatalf("PlanJobMigration() error = %v", err)
	}
	if len(plan.Warnings) != 1 {
		t.Fatalf("len(plan.Warnings) = %d, want 1", len(plan.Warnings))
	}
	if len(plan.RepositoryCompatibility) != 1 || plan.RepositoryCompatibility[0].Compatible {
		t.Fatalf("unexpected repository compatibility: %#v", plan.RepositoryCompatibility)
	}
}

func TestPortabilityManager_PlanFleetMigration_MultipleWorkloads(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{
		{"id": "job-1", "name": "Daily web-01", "targetRepo": "primary-repo", "retentionDays": 7, "protectedVMs": []string{"web-01"}},
		{"id": "job-2", "name": "Daily app-02", "targetRepo": "primary-repo", "retentionDays": 7, "protectedVMs": []string{"app-02"}},
	})
	defer cleanup()

	plan, err := manager.PlanFleetMigration(context.Background(), []WorkloadMapping{
		{SourceVM: sampleBackupSourceVM(), TargetVM: sampleBackupTargetVM()},
		{SourceVM: models.VirtualMachine{ID: "vm-app", Name: "app-02"}, TargetVM: models.VirtualMachine{ID: "vm-app-target", Name: "app-02-target"}},
	}, nil)
	if err != nil {
		t.Fatalf("PlanFleetMigration() error = %v", err)
	}
	if len(plan.Plans) != 2 {
		t.Fatalf("len(plan.Plans) = %d, want 2", len(plan.Plans))
	}
}

func TestPortabilityManager_ExecuteFleetMigration_AggregatesResults(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServer(t, []map[string]interface{}{})
	defer cleanup()

	result, err := manager.ExecuteFleetMigration(context.Background(), &FleetJobMigrationPlan{
		Plans: []JobMigrationPlan{
			{
				TargetVM: models.VirtualMachine{Name: "web-01-target"},
				Jobs: []BackupJobTemplate{
					{Name: "Daily web-01-target", TargetRepo: "primary-repo", ProtectedVMs: []string{"web-01-target"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteFleetMigration() error = %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(result.Results) = %d, want 1", len(result.Results))
	}
}

func TestPortabilityManager_ValidateBackupContinuity_Healthy_Expected(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServerWithInventory(
		t,
		[]map[string]interface{}{
			{
				"id":            "job-1",
				"name":          "Daily web-01-target",
				"schedule":      "0 2 * * *",
				"targetRepo":    "target-repo",
				"retentionDays": 14,
				"protectedVMs":  []string{"web-01-target"},
				"enabled":       true,
			},
		},
		[]map[string]interface{}{
			{
				"id":        "rp-1",
				"vmId":      "vm-target",
				"vmName":    "web-01-target",
				"jobName":   "Daily web-01-target",
				"createdAt": "2026-04-07T03:00:00Z",
				"sizeMB":    1024,
				"type":      "incremental",
			},
		},
		nil,
	)
	defer cleanup()

	report, err := manager.ValidateBackupContinuity(context.Background(), &JobMigrationPlan{
		TargetVM: sampleBackupTargetVM(),
		Jobs: []BackupJobTemplate{
			{
				Name:          "Daily web-01-target",
				Schedule:      "0 2 * * *",
				TargetRepo:    "target-repo",
				RetentionDays: 14,
				ProtectedVMs:  []string{"web-01-target"},
				Enabled:       true,
			},
		},
	}, &JobMigrationResult{
		VerificationStatus: map[string]string{"job-created-1": "verified"},
	})
	if err != nil {
		t.Fatalf("ValidateBackupContinuity() error = %v", err)
	}
	if report.Status != "healthy" || report.RestorePointCount != 1 {
		t.Fatalf("unexpected continuity report: %#v", report)
	}
	if len(report.PolicyDrifts) != 0 {
		t.Fatalf("PolicyDrifts = %#v, want none", report.PolicyDrifts)
	}
}

func TestPortabilityManager_ValidateBackupContinuity_DetectsPolicyDrift_Expected(t *testing.T) {
	t.Parallel()

	manager, cleanup := newPortabilityManagerTestServerWithInventory(
		t,
		[]map[string]interface{}{
			{
				"id":            "job-1",
				"name":          "Daily web-01-target",
				"schedule":      "0 4 * * *",
				"targetRepo":    "archive-repo",
				"retentionDays": 7,
				"protectedVMs":  []string{"web-01-target"},
				"enabled":       false,
			},
		},
		[]map[string]interface{}{},
		nil,
	)
	defer cleanup()

	report, err := manager.ValidateBackupContinuity(context.Background(), &JobMigrationPlan{
		TargetVM: sampleBackupTargetVM(),
		Jobs: []BackupJobTemplate{
			{
				Name:          "Daily web-01-target",
				Schedule:      "0 2 * * *",
				TargetRepo:    "target-repo",
				RetentionDays: 14,
				ProtectedVMs:  []string{"web-01-target"},
				Enabled:       true,
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("ValidateBackupContinuity() error = %v", err)
	}
	if report.Status != "degraded" {
		t.Fatalf("Status = %q, want degraded", report.Status)
	}
	if len(report.PolicyDrifts) == 0 {
		t.Fatal("PolicyDrifts is empty, want detected drift")
	}
	if report.RestorePointCount != 0 {
		t.Fatalf("RestorePointCount = %d, want 0", report.RestorePointCount)
	}
}

func newPortabilityManagerTestServer(t *testing.T, jobs []map[string]interface{}) (*PortabilityManager, func()) {
	t.Helper()
	return newPortabilityManagerTestServerWithInventory(t, jobs, nil, nil)
}

func newPortabilityManagerTestServerWithInventory(t *testing.T, jobs, restorePoints, repositories []map[string]interface{}) (*PortabilityManager, func()) {
	t.Helper()

	if repositories == nil {
		repositories = []map[string]interface{}{
			{"id": "repo-1", "name": "primary-repo", "type": "xfs", "capacityMB": 102400, "freeMB": 51200, "usedMB": 51200},
			{"id": "repo-2", "name": "target-repo", "type": "xfs", "capacityMB": 102400, "freeMB": 90000, "usedMB": 12400},
		}
	}
	if restorePoints == nil {
		restorePoints = []map[string]interface{}{}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs":
			writePortabilityJSON(t, w, jobs)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/objectRestorePoints":
			writePortabilityJSON(t, w, restorePoints)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/backupInfrastructure/repositories":
			writePortabilityJSON(t, w, repositories)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			writePortabilityJSON(t, w, map[string]string{"id": "job-created-1"})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") && strings.HasSuffix(r.URL.Path, "/start"):
			writePortabilityJSON(t, w, map[string]string{"status": "started"})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/jobs/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))

	client := NewVeeamClient(server.URL, true)
	client.accessToken = "token"
	return NewPortabilityManager(client), server.Close
}

func writePortabilityJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}

func sampleBackupSourceVM() models.VirtualMachine {
	return models.VirtualMachine{ID: "vm-source", Name: "web-01"}
}

func sampleBackupTargetVM() models.VirtualMachine {
	return models.VirtualMachine{ID: "vm-target", Name: "web-01-target"}
}
