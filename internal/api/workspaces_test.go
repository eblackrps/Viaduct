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
	server := NewServer(nil, stateStore, 0, nil)
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
	server := NewServer(nil, stateStore, 0, nil)
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
}
