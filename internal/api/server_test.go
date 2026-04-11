package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	server := NewServer(nil, stateStore, 0, nil)
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
	server := NewServer(nil, stateStore, 0, nil)

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

func TestServer_HandleAdminTenants_ListRedactsSecrets_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0, nil)
	tenant := models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-secret",
		CreatedAt: time.Date(2026, time.April, 7, 14, 0, 0, 0, time.UTC),
		Active:    true,
		Quotas:    models.TenantQuota{RequestsPerMinute: 120},
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "svc-a",
				Name:      "automation",
				APIKey:    "svc-secret",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Date(2026, time.April, 7, 14, 5, 0, 0, time.UTC),
			},
		},
	}
	if err := stateStore.CreateTenant(context.Background(), tenant); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	recorder := httptest.NewRecorder()

	server.handleAdminTenants(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var listed []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	var listedTenant map[string]any
	for _, item := range listed {
		if item["id"] == tenant.ID {
			listedTenant = item
			break
		}
	}
	if listedTenant == nil {
		t.Fatalf("tenant %q not found in admin list: %#v", tenant.ID, listed)
	}
	if _, ok := listedTenant["api_key"]; ok {
		t.Fatalf("tenant list unexpectedly exposed api_key: %#v", listedTenant)
	}
	accounts, ok := listedTenant["service_accounts"].([]any)
	if !ok || len(accounts) != 1 {
		t.Fatalf("unexpected service_accounts payload: %#v", listedTenant["service_accounts"])
	}
	account, ok := accounts[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected service account payload: %#v", accounts[0])
	}
	if _, ok := account["api_key"]; ok {
		t.Fatalf("tenant list unexpectedly exposed service-account api_key: %#v", account)
	}
}

func TestServer_HandleAdminTenantByID_InvalidNestedPathRejected_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
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

	server := NewServer(nil, stateStore, 0, nil)
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
	server := NewServer(nil, stateStore, 0, nil)
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

	var response apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Error.Code != "approval_required" || response.Error.RequestID == "" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}

func TestServer_NewMigrationCommandResponse_UsesStoredPhaseAndRequestID_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0, nil)
	if err := stateStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
		ID:        "migration-2",
		TenantID:  store.DefaultTenantID,
		SpecName:  "execute",
		Phase:     string(migratepkg.PhasePlan),
		StartedAt: time.Date(2026, time.April, 8, 14, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.April, 8, 14, 1, 0, 0, time.UTC),
		RawJSON:   json.RawMessage(`{"id":"migration-2","phase":"plan"}`),
	}); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/migration-2/execute", bytes.NewBuffer(nil))
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	req = req.WithContext(context.WithValue(req.Context(), requestIDContextKey{}, "req-123"))

	response := server.newMigrationCommandResponse(req, store.DefaultTenantID, "migration-2", "execute", "executing")
	if response.Action != "execute" || response.OperationState != "accepted" || response.LifecycleState != "executing" {
		t.Fatalf("unexpected command response: %#v", response)
	}
	if response.Phase != migratepkg.PhasePlan || response.RequestID == "" || response.AcceptedAt.IsZero() {
		t.Fatalf("incomplete command response: %#v", response)
	}
	if response.RequestID != "req-123" {
		t.Fatalf("RequestID = %q, want req-123", response.RequestID)
	}
}

func TestServer_LatestInventory_MoreThanTwentySources_IncludesAllLatestSources(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0, nil)
	ctx := context.Background()

	for index := 0; index < 25; index++ {
		result := &models.DiscoveryResult{
			Source:       fmt.Sprintf("source-%02d", index),
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 7, 10, index, 0, 0, time.UTC),
			VMs: []models.VirtualMachine{
				{ID: fmt.Sprintf("vm-%02d", index), Name: fmt.Sprintf("vm-%02d", index), Platform: models.PlatformVMware},
			},
		}
		if _, err := stateStore.SaveDiscovery(ctx, store.DefaultTenantID, result); err != nil {
			t.Fatalf("SaveDiscovery(%s) error = %v", result.Source, err)
		}
	}

	inventory, err := server.latestInventory(store.ContextWithTenantID(ctx, store.DefaultTenantID), models.PlatformVMware)
	if err != nil {
		t.Fatalf("latestInventory() error = %v", err)
	}
	if len(inventory.VMs) != 25 {
		t.Fatalf("len(inventory.VMs) = %d, want 25", len(inventory.VMs))
	}
}

func TestServer_HandleSummary_PendingApprovalParsedFromRecord_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := NewServer(nil, stateStore, 0, nil)
	if err := stateStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
		ID:        "migration-pending-approval",
		SpecName:  "approval-test",
		Phase:     string(migratepkg.PhasePlan),
		StartedAt: time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.April, 7, 12, 1, 0, 0, time.UTC),
		RawJSON: json.RawMessage(`{
  "id": "migration-pending-approval",
  "phase": "plan",
  "pending_approval": true
}`),
	}); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/summary", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleSummary(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var summary tenantSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &summary); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if summary.PendingApprovals != 1 {
		t.Fatalf("PendingApprovals = %d, want 1", summary.PendingApprovals)
	}
}

func TestServer_HandleAbout_ReturnsBuildInfo_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	server.SetBuildInfo("v1.2.0", "deadbee", "2026-04-07T15:00:00Z")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/about", nil)
	recorder := httptest.NewRecorder()

	server.handleAbout(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response aboutResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Version != "v1.2.0" || response.Commit != "deadbee" || response.PluginProtocol == "" {
		t.Fatalf("unexpected about response: %#v", response)
	}
	if len(response.SupportedPlatforms) == 0 {
		t.Fatal("SupportedPlatforms is empty")
	}
	if response.StoreBackend != "memory" || response.PersistentStore {
		t.Fatalf("unexpected store diagnostics: %#v", response)
	}
	if len(response.SupportedPermissions) == 0 {
		t.Fatal("SupportedPermissions is empty")
	}
}

func TestServer_Handler_CORSAndSecurityHeaders_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want localhost dev origin", got)
	}
	for header, want := range map[string]string{
		"Cache-Control":           "no-store",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
		"Permissions-Policy":      "camera=(), geolocation=(), microphone=()",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	} {
		if got := recorder.Header().Get(header); got != want {
			t.Fatalf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestServer_Handler_DisallowedCORSOriginRejected_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	request.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestServer_HandlePreflight_InvalidSpecReturnsFieldErrors_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preflight", bytes.NewBufferString(`{"name":"bad-spec"}`))
	recorder := httptest.NewRecorder()

	server.handlePreflight(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}

	var response apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Error.Code != "invalid_spec" {
		t.Fatalf("error code = %q, want invalid_spec", response.Error.Code)
	}
	if len(response.Error.FieldErrors) == 0 {
		t.Fatalf("expected field errors, got %#v", response.Error)
	}
}
