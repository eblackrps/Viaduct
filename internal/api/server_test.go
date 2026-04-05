package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_LatestInventory_UsesLatestSnapshotPerSourcePlatform_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0)
	ctx := context.Background()

	for _, result := range []*models.DiscoveryResult{
		{
			Source:       "vcsa-a",
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 4, 9, 0, 0, 0, time.UTC),
			VMs:          []models.VirtualMachine{{ID: "vm-old", Name: "old-a", Platform: models.PlatformVMware}},
		},
		{
			Source:       "vcsa-a",
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 4, 10, 0, 0, 0, time.UTC),
			VMs:          []models.VirtualMachine{{ID: "vm-new", Name: "new-a", Platform: models.PlatformVMware}},
		},
		{
			Source:       "pve-b",
			Platform:     models.PlatformProxmox,
			DiscoveredAt: time.Date(2026, time.April, 4, 10, 5, 0, 0, time.UTC),
			VMs:          []models.VirtualMachine{{ID: "vm-b", Name: "new-b", Platform: models.PlatformProxmox}},
		},
	} {
		if _, err := stateStore.SaveDiscovery(ctx, store.DefaultTenantID, result); err != nil {
			t.Fatalf("SaveDiscovery(%s) error = %v", result.Source, err)
		}
	}

	inventory, err := server.latestInventory(store.ContextWithTenantID(ctx, store.DefaultTenantID), "")
	if err != nil {
		t.Fatalf("latestInventory() error = %v", err)
	}
	if len(inventory.VMs) != 2 {
		t.Fatalf("len(inventory.VMs) = %d, want 2", len(inventory.VMs))
	}
	for _, vm := range inventory.VMs {
		if vm.Name == "old-a" {
			t.Fatalf("inventory unexpectedly included stale snapshot VM: %#v", inventory.VMs)
		}
	}
}

func TestServer_HandleAdminTenants_ExplicitInactiveTenantPreserved_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0)

	body := bytes.NewBufferString(`{"name":"Inactive Tenant","active":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", body)
	recorder := httptest.NewRecorder()

	server.handleAdminTenants(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created models.Tenant
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if created.Active {
		t.Fatalf("created tenant Active = %t, want false", created.Active)
	}

	persisted, err := stateStore.GetTenant(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	if persisted.Active {
		t.Fatalf("persisted tenant Active = %t, want false", persisted.Active)
	}
}

func TestServer_HandleAdminTenantByID_InvalidNestedPathRejected_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0)
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/tenants/tenant-a/extra", nil)
	recorder := httptest.NewRecorder()

	server.handleAdminTenantByID(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestServer_HandleSummary_TenantScopedCounts_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	for _, tenant := range []models.Tenant{
		{ID: "tenant-a", Name: "Tenant A", APIKey: "tenant-a-key", Active: true},
		{ID: "tenant-b", Name: "Tenant B", APIKey: "tenant-b-key", Active: true},
	} {
		if err := stateStore.CreateTenant(context.Background(), tenant); err != nil {
			t.Fatalf("CreateTenant(%s) error = %v", tenant.ID, err)
		}
	}

	server := NewServer(nil, stateStore, 0)
	_, _ = stateStore.SaveDiscovery(context.Background(), "tenant-a", &models.DiscoveryResult{
		Source:       "tenant-a-source",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
		VMs:          []models.VirtualMachine{{ID: "vm-a", Name: "tenant-a-vm", Platform: models.PlatformVMware}},
	})
	_, _ = stateStore.SaveDiscovery(context.Background(), "tenant-b", &models.DiscoveryResult{
		Source:       "tenant-b-source",
		Platform:     models.PlatformProxmox,
		DiscoveredAt: time.Now().UTC(),
		VMs:          []models.VirtualMachine{{ID: "vm-b", Name: "tenant-b-vm", Platform: models.PlatformProxmox}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	recorder := httptest.NewRecorder()

	server.handleSummary(recorder, req.WithContext(store.ContextWithTenantID(req.Context(), "tenant-a")))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var summary tenantSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &summary); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if summary.TenantID != "tenant-a" || summary.WorkloadCount != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.PlatformCounts["vmware"] != 1 || summary.PlatformCounts["proxmox"] != 0 {
		t.Fatalf("unexpected platform counts: %#v", summary.PlatformCounts)
	}
}

func TestServer_HandleMigrationByID_ExecuteApprovalRequiredConflict_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0)
	server.mu.Lock()
	server.specs[server.specKey(store.DefaultTenantID, "migration-1")] = &migratepkg.MigrationSpec{
		Name: "approval-required",
		Source: migratepkg.SourceSpec{
			Address:  "vcsa.lab.local",
			Platform: models.PlatformVMware,
		},
		Target: migratepkg.TargetSpec{
			Address:  "pve.lab.local",
			Platform: models.PlatformProxmox,
		},
		Workloads: []migratepkg.WorkloadSelector{{Match: migratepkg.MatchCriteria{NamePattern: "web-*"}}},
		Options: migratepkg.MigrationOptions{
			Parallel: 1,
			Approval: migratepkg.ApprovalGate{Required: true},
		},
	}
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/migration-1/execute", bytes.NewBuffer(nil))
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleMigrationByID(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
}
