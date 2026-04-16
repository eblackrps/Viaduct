package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_HandleWorkspaces_CreateUpdateAndList_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(`{
		"name":"Lab Assessment",
		"description":"Pilot workspace",
		"source_connections":[{"id":"src-1","name":"Lab KVM","platform":"kvm","address":"examples/lab/kvm","credential_ref":"lab-kvm"}],
		"target_assumptions":{"platform":"proxmox","address":"https://pilot-proxmox.local:8006/api2/json","default_host":"pve-node-01"}
	}`))
	createRecorder := httptest.NewRecorder()
	server.handleWorkspaces(createRecorder, createRequest.WithContext(ctx))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	var created models.PilotWorkspace
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal(created) error = %v", err)
	}
	if created.ID == "" || created.Status != models.PilotWorkspaceStatusDraft {
		t.Fatalf("unexpected created workspace: %#v", created)
	}

	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+created.ID, bytes.NewBufferString(`{
		"description":"Updated pilot workspace",
		"plan_settings":{"name":"lab-plan","parallel":2,"verify_boot":true}
	}`))
	updateRecorder := httptest.NewRecorder()
	server.handleWorkspaceByID(updateRecorder, updateRequest.WithContext(ctx))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updateRecorder.Code, http.StatusOK, updateRecorder.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	listRecorder := httptest.NewRecorder()
	server.handleWorkspaces(listRecorder, listRequest.WithContext(ctx))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", listRecorder.Code, http.StatusOK, listRecorder.Body.String())
	}

	var listed []models.PilotWorkspace
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("Unmarshal(listed) error = %v", err)
	}
	if len(listed) != 1 || listed[0].PlanSettings.Name != "lab-plan" {
		t.Fatalf("unexpected listed workspaces: %#v", listed)
	}
}

func TestServer_HandleWorkspaceJobs_DiscoveryAndReport_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	if err := stateStore.CreateWorkspace(ctx, store.DefaultTenantID, models.PilotWorkspace{
		ID:          "workspace-1",
		Name:        "Lab Workspace",
		Status:      models.PilotWorkspaceStatusDraft,
		Description: "Assessment",
		SourceConnections: []models.WorkspaceSourceConnection{
			{
				ID:            "src-1",
				Name:          "Lab KVM",
				Platform:      models.PlatformKVM,
				Address:       filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
				CredentialRef: "lab-kvm",
			},
		},
		TargetAssumptions: models.WorkspaceTargetAssumptions{
			Platform:       models.PlatformProxmox,
			Address:        "https://pilot-proxmox.local:8006/api2/json",
			DefaultHost:    "pve-node-01",
			DefaultStorage: "local-lvm",
			DefaultNetwork: "vmbr0",
		},
		PlanSettings: models.WorkspacePlanSettings{
			Name:             "lab-plan",
			Parallel:         2,
			VerifyBoot:       true,
			ApprovalRequired: true,
			DependencyAware:  true,
			WaveSize:         2,
		},
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	jobRequest := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/workspace-1/jobs", bytes.NewBufferString(`{"type":"discovery"}`))
	jobRecorder := httptest.NewRecorder()
	server.handleWorkspaceByID(jobRecorder, jobRequest.WithContext(ctx))
	if jobRecorder.Code != http.StatusAccepted {
		t.Fatalf("job status = %d, want %d: %s", jobRecorder.Code, http.StatusAccepted, jobRecorder.Body.String())
	}

	var job models.WorkspaceJob
	if err := json.Unmarshal(jobRecorder.Body.Bytes(), &job); err != nil {
		t.Fatalf("Unmarshal(job) error = %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current, err := stateStore.GetWorkspaceJob(ctx, store.DefaultTenantID, "workspace-1", job.ID)
		if err != nil {
			t.Fatalf("GetWorkspaceJob() error = %v", err)
		}
		if current.Status == models.WorkspaceJobStatusSucceeded {
			break
		}
		if current.Status == models.WorkspaceJobStatusFailed {
			t.Fatalf("workspace discovery job failed: %#v", current)
		}
		time.Sleep(100 * time.Millisecond)
	}

	workspace, err := stateStore.GetWorkspace(ctx, store.DefaultTenantID, "workspace-1")
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if len(workspace.Snapshots) == 0 || workspace.Status != models.PilotWorkspaceStatusDiscovered {
		t.Fatalf("workspace = %#v, want discovered snapshot state", workspace)
	}

	reportRequest := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/workspace-1/reports/export", bytes.NewBufferString(`{"format":"markdown"}`))
	reportRecorder := httptest.NewRecorder()
	server.handleWorkspaceByID(reportRecorder, reportRequest.WithContext(ctx))
	if reportRecorder.Code != http.StatusOK {
		t.Fatalf("report status = %d, want %d: %s", reportRecorder.Code, http.StatusOK, reportRecorder.Body.String())
	}
	if contentType := reportRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("Content-Type = %q, want markdown", contentType)
	}
	if !strings.Contains(reportRecorder.Body.String(), "# Pilot Workspace Report") {
		t.Fatalf("unexpected report payload: %s", reportRecorder.Body.String())
	}
	if !strings.Contains(reportRecorder.Body.String(), "## Background Jobs") || !strings.Contains(reportRecorder.Body.String(), "## Report History") {
		t.Fatalf("report is missing operator handoff sections: %s", reportRecorder.Body.String())
	}
}

func TestServer_HandleWorkspaces_CreateValidation_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBufferString(`{
		"name":"  ",
		"source_connections":[{"id":"src-1","name":"","platform":"invalid","address":""}],
		"notes":[{"kind":"operator","author":"","body":""}]
	}`))
	recorder := httptest.NewRecorder()

	server.handleWorkspaces(recorder, request.WithContext(ctx))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	var payload apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(payload.Error.FieldErrors) < 4 {
		t.Fatalf("expected multiple field errors, got %#v", payload.Error.FieldErrors)
	}
}

func TestServer_HandleWorkspaceDocument_DeleteWorkspace_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	if err := stateStore.CreateWorkspace(ctx, store.DefaultTenantID, models.PilotWorkspace{
		ID:     "workspace-delete",
		Name:   "Delete Me",
		Status: models.PilotWorkspaceStatusDraft,
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if err := stateStore.SaveWorkspaceJob(ctx, store.DefaultTenantID, models.WorkspaceJob{
		ID:          "job-delete",
		WorkspaceID: "workspace-delete",
		Type:        models.WorkspaceJobTypeDiscovery,
		Status:      models.WorkspaceJobStatusSucceeded,
		RequestedAt: time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveWorkspaceJob() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/workspace-delete", nil)
	recorder := httptest.NewRecorder()

	server.handleWorkspaceByID(recorder, request.WithContext(ctx))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}

	if _, err := stateStore.GetWorkspace(ctx, store.DefaultTenantID, "workspace-delete"); err == nil {
		t.Fatal("GetWorkspace() error = nil, want not found")
	}
	if _, err := stateStore.GetWorkspaceJob(ctx, store.DefaultTenantID, "workspace-delete", "job-delete"); err == nil {
		t.Fatal("GetWorkspaceJob() error = nil, want not found")
	}
}

func TestServer_HandleWorkspaceJobs_CreateValidation_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	if err := stateStore.CreateWorkspace(ctx, store.DefaultTenantID, models.PilotWorkspace{
		ID:     "workspace-validation",
		Name:   "Validation Workspace",
		Status: models.PilotWorkspaceStatusDraft,
		SourceConnections: []models.WorkspaceSourceConnection{
			{
				ID:       "src-validation",
				Name:     "Lab KVM",
				Platform: models.PlatformKVM,
				Address:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
			},
		},
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/workspace-validation/jobs", bytes.NewBufferString(`{
		"type":"plan",
		"source_connection_ids":["missing"],
		"selected_workload_ids":["","vm-1"]
	}`))
	recorder := httptest.NewRecorder()

	server.handleWorkspaceByID(recorder, request.WithContext(ctx))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	var payload apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(payload.Error.FieldErrors) < 3 {
		t.Fatalf("expected multiple field errors, got %#v", payload.Error.FieldErrors)
	}
}

func TestServer_WorkspaceRoutes_ViewerCanReadButNotMutate_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-view",
		Name:   "Viewer Tenant",
		APIKey: "tenant-view-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "viewer",
				Name:      "Viewer",
				APIKey:    "viewer-key",
				Role:      models.TenantRoleViewer,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
			{
				ID:        "operator",
				Name:      "Operator",
				APIKey:    "operator-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	tenantCtx := store.ContextWithTenantID(context.Background(), "tenant-view")
	if err := stateStore.CreateWorkspace(tenantCtx, "tenant-view", models.PilotWorkspace{
		ID:     "workspace-view",
		Name:   "Viewer Workspace",
		Status: models.PilotWorkspaceStatusDraft,
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	handler := server.Handler()

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/workspace-view", nil)
	getRequest.Header.Set("X-Service-Account-Key", "viewer-key")
	getRecorder := httptest.NewRecorder()
	handler.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("viewer get status = %d, want %d: %s", getRecorder.Code, http.StatusOK, getRecorder.Body.String())
	}

	reportRequest := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/workspace-view/reports/export", bytes.NewBufferString(`{"format":"json"}`))
	reportRequest.Header.Set("X-Service-Account-Key", "viewer-key")
	reportRecorder := httptest.NewRecorder()
	handler.ServeHTTP(reportRecorder, reportRequest)
	if reportRecorder.Code != http.StatusOK {
		t.Fatalf("viewer report export status = %d, want %d: %s", reportRecorder.Code, http.StatusOK, reportRecorder.Body.String())
	}

	patchRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/workspace-view", bytes.NewBufferString(`{"description":"mutated"}`))
	patchRequest.Header.Set("X-Service-Account-Key", "viewer-key")
	patchRecorder := httptest.NewRecorder()
	handler.ServeHTTP(patchRecorder, patchRequest)
	if patchRecorder.Code != http.StatusForbidden {
		t.Fatalf("viewer patch status = %d, want %d: %s", patchRecorder.Code, http.StatusForbidden, patchRecorder.Body.String())
	}

	jobRequest := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/workspace-view/jobs", bytes.NewBufferString(`{"type":"discovery"}`))
	jobRequest.Header.Set("X-Service-Account-Key", "viewer-key")
	jobRecorder := httptest.NewRecorder()
	handler.ServeHTTP(jobRecorder, jobRequest)
	if jobRecorder.Code != http.StatusForbidden {
		t.Fatalf("viewer job create status = %d, want %d: %s", jobRecorder.Code, http.StatusForbidden, jobRecorder.Body.String())
	}
}

func TestServer_RecoverWorkspaceJobs_RerunsQueuedDiscoveryJob_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	ctx := store.ContextWithTenantID(context.Background(), store.DefaultTenantID)

	if err := stateStore.CreateWorkspace(ctx, store.DefaultTenantID, models.PilotWorkspace{
		ID:     "workspace-recover",
		Name:   "Recover Workspace",
		Status: models.PilotWorkspaceStatusDraft,
		SourceConnections: []models.WorkspaceSourceConnection{
			{
				ID:       "src-recover",
				Name:     "Lab KVM",
				Platform: models.PlatformKVM,
				Address:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
			},
		},
	}); err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	if err := stateStore.SaveWorkspaceJob(ctx, store.DefaultTenantID, models.WorkspaceJob{
		ID:            "job-recover",
		WorkspaceID:   "workspace-recover",
		Type:          models.WorkspaceJobTypeDiscovery,
		Status:        models.WorkspaceJobStatusQueued,
		RequestedAt:   time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		CorrelationID: "req-recover",
		InputJSON:     json.RawMessage(`{"type":"discovery"}`),
	}); err != nil {
		t.Fatalf("SaveWorkspaceJob() error = %v", err)
	}

	if err := server.recoverWorkspaceJobs(ctx); err != nil {
		t.Fatalf("recoverWorkspaceJobs() error = %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		job, err := stateStore.GetWorkspaceJob(ctx, store.DefaultTenantID, "workspace-recover", "job-recover")
		if err != nil {
			t.Fatalf("GetWorkspaceJob() error = %v", err)
		}
		if job.Status == models.WorkspaceJobStatusSucceeded {
			workspace, err := stateStore.GetWorkspace(ctx, store.DefaultTenantID, "workspace-recover")
			if err != nil {
				t.Fatalf("GetWorkspace() error = %v", err)
			}
			if len(workspace.Snapshots) == 0 {
				t.Fatalf("workspace snapshots = %#v, want discovered snapshots", workspace.Snapshots)
			}
			return
		}
		if job.Status == models.WorkspaceJobStatusFailed {
			t.Fatalf("recovered job failed: %#v", job)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("recovered workspace job did not complete before timeout")
}
