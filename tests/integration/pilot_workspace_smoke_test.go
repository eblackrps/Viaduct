package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/api"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestPilotWorkspace_LabFlow_CreateDiscoverGraphSimulatePlanReport_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-pilot",
		Name:   "Pilot Tenant",
		APIKey: "tenant-pilot-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:          "sa-pilot",
				Name:        "Pilot Automation",
				APIKey:      "sa-pilot-key",
				Role:        models.TenantRoleOperator,
				Permissions: []models.TenantPermission{models.TenantPermissionMigrationManage},
				Active:      true,
				CreatedAt:   time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := api.NewServer(nil, stateStore, 0, nil)
	handler := server.Handler()

	createRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodPost, "/api/v1/workspaces", `{
		"name":"Examples Lab Assessment",
		"description":"Smoke flow workspace",
		"source_connections":[{"id":"src-lab","name":"Lab KVM","platform":"kvm","address":"examples/lab/kvm","credential_ref":"lab-kvm"}],
		"target_assumptions":{"platform":"proxmox","address":"https://pilot-proxmox.local:8006/api2/json","default_host":"pve-node-01","default_storage":"local-lvm","default_network":"vmbr0"},
		"plan_settings":{"name":"examples-lab-plan","parallel":2,"verify_boot":true,"approval_required":true,"wave_size":2,"dependency_aware":true}
	}`)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	var workspace models.PilotWorkspace
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &workspace); err != nil {
		t.Fatalf("Unmarshal(workspace) error = %v", err)
	}

	for _, jobType := range []string{"discovery", "graph", "simulation", "plan"} {
		jobRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodPost, "/api/v1/workspaces/"+workspace.ID+"/jobs", `{"type":"`+jobType+`"}`)
		if jobRecorder.Code != http.StatusAccepted {
			t.Fatalf("%s job status = %d, want %d: %s", jobType, jobRecorder.Code, http.StatusAccepted, jobRecorder.Body.String())
		}

		var job models.WorkspaceJob
		if err := json.Unmarshal(jobRecorder.Body.Bytes(), &job); err != nil {
			t.Fatalf("Unmarshal(job) error = %v", err)
		}
		waitForWorkspaceJob(t, handler, workspace.ID, job.ID)
	}

	reportRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodPost, "/api/v1/workspaces/"+workspace.ID+"/reports/export", `{"format":"markdown"}`)
	if reportRecorder.Code != http.StatusOK {
		t.Fatalf("report status = %d, want %d: %s", reportRecorder.Code, http.StatusOK, reportRecorder.Body.String())
	}
	if !strings.Contains(reportRecorder.Body.String(), "# Pilot Workspace Report") {
		t.Fatalf("unexpected report body: %s", reportRecorder.Body.String())
	}

	workspaceRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodGet, "/api/v1/workspaces/"+workspace.ID, "")
	if workspaceRecorder.Code != http.StatusOK {
		t.Fatalf("workspace get status = %d, want %d: %s", workspaceRecorder.Code, http.StatusOK, workspaceRecorder.Body.String())
	}
	if err := json.Unmarshal(workspaceRecorder.Body.Bytes(), &workspace); err != nil {
		t.Fatalf("Unmarshal(updated workspace) error = %v", err)
	}
	if workspace.Graph == nil || workspace.Simulation == nil || workspace.SavedPlan == nil || len(workspace.Reports) == 0 {
		t.Fatalf("workspace flow incomplete: %#v", workspace)
	}

	deleteRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodDelete, "/api/v1/workspaces/"+workspace.ID, "")
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d: %s", deleteRecorder.Code, http.StatusNoContent, deleteRecorder.Body.String())
	}

	missingRecorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodGet, "/api/v1/workspaces/"+workspace.ID, "")
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("deleted workspace get status = %d, want %d: %s", missingRecorder.Code, http.StatusNotFound, missingRecorder.Body.String())
	}
}

func sendWorkspaceRequest(t *testing.T, handler http.Handler, serviceAccountKey, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("X-Service-Account-Key", serviceAccountKey)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func waitForWorkspaceJob(t *testing.T, handler http.Handler, workspaceID, jobID string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		recorder := sendWorkspaceRequest(t, handler, "sa-pilot-key", http.MethodGet, "/api/v1/workspaces/"+workspaceID+"/jobs/"+jobID, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("job poll status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var job models.WorkspaceJob
		if err := json.Unmarshal(recorder.Body.Bytes(), &job); err != nil {
			t.Fatalf("Unmarshal(polled job) error = %v", err)
		}
		if job.Status == models.WorkspaceJobStatusSucceeded {
			return
		}
		if job.Status == models.WorkspaceJobStatusFailed {
			t.Fatalf("workspace job %s failed: %#v", job.Type, job)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("workspace job %s did not complete before timeout", jobID)
}
