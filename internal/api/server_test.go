package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

type paginationProbeStore struct {
	store.Store

	listSnapshotsCalls      int
	listSnapshotsPageCalls  int
	listMigrationsCalls     int
	listMigrationsPageCalls int
}

func (s *paginationProbeStore) ListSnapshots(ctx context.Context, tenantID string, platform models.Platform, limit int) ([]store.SnapshotMeta, error) {
	s.listSnapshotsCalls++
	return s.Store.ListSnapshots(ctx, tenantID, platform, limit)
}

func (s *paginationProbeStore) ListSnapshotsPage(ctx context.Context, tenantID string, platform models.Platform, page, perPage int) ([]store.SnapshotMeta, int, error) {
	s.listSnapshotsPageCalls++
	return s.Store.ListSnapshotsPage(ctx, tenantID, platform, page, perPage)
}

func (s *paginationProbeStore) ListMigrations(ctx context.Context, tenantID string, limit int) ([]store.MigrationMeta, error) {
	s.listMigrationsCalls++
	return s.Store.ListMigrations(ctx, tenantID, limit)
}

func (s *paginationProbeStore) ListMigrationsPage(ctx context.Context, tenantID string, page, perPage int) ([]store.MigrationMeta, int, error) {
	s.listMigrationsPageCalls++
	return s.Store.ListMigrationsPage(ctx, tenantID, page, perPage)
}

func TestServer_LatestInventory_UsesLatestSnapshotPerSourcePlatform_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
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
	server := mustNewServer(t, stateStore)

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

func TestServer_HandleAdminTenants_TrimsOneTimeTenantKeyResponse_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)

	body := bytes.NewBufferString(`{"name":"Trimmed Tenant","api_key":"  tenant-key  "}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tenants", body)
	recorder := httptest.NewRecorder()

	server.handleAdminTenants(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var created adminTenantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if created.APIKey != "tenant-key" {
		t.Fatalf("created tenant APIKey = %q, want trimmed value", created.APIKey)
	}

	persisted, err := stateStore.GetTenant(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	if !store.APIKeyMatches("tenant-key", persisted.APIKeyHash, persisted.APIKey) {
		t.Fatal("persisted tenant credential did not match trimmed one-time response")
	}
}

func TestServer_HandleAdminTenants_ListRedactsSecrets_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
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

	server := mustNewServer(t, store.NewMemoryStore())
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

	server := mustNewServer(t, stateStore)
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
	server := mustNewServer(t, stateStore)
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
	server := mustNewServer(t, stateStore)
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
	server := mustNewServer(t, stateStore)
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
	server := mustNewServer(t, stateStore)
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

	server := mustNewServer(t, store.NewMemoryStore())
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
	if response.LocalOperatorSession {
		t.Fatal("LocalOperatorSession = true, want false by default")
	}
}

func TestServer_Handler_OpenAPIDocsRedirectAndJSON_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	redirectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	redirectRecorder := httptest.NewRecorder()
	handler.ServeHTTP(redirectRecorder, redirectRequest)
	if redirectRecorder.Code != http.StatusPermanentRedirect {
		t.Fatalf("redirect status = %d, want %d", redirectRecorder.Code, http.StatusPermanentRedirect)
	}
	if location := redirectRecorder.Header().Get("Location"); location != "/api/v1/docs/index.html" {
		t.Fatalf("redirect location = %q, want /api/v1/docs/index.html", location)
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/api/v1/docs/swagger.json", nil)
	jsonRecorder := httptest.NewRecorder()
	handler.ServeHTTP(jsonRecorder, jsonRequest)
	if jsonRecorder.Code != http.StatusOK {
		t.Fatalf("swagger status = %d, want %d: %s", jsonRecorder.Code, http.StatusOK, jsonRecorder.Body.String())
	}
	if contentType := jsonRecorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	var document map[string]any
	if err := json.Unmarshal(jsonRecorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if document["openapi"] == nil {
		t.Fatalf("openapi field missing from swagger document: %#v", document)
	}
}

func TestServer_Handler_MetricsRouteRequiresAdmin_Expected(t *testing.T) {
	t.Setenv("VIADUCT_ADMIN_KEY", "admin-key")
	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	unauthorizedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedRecorder, unauthorizedRequest)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d: %s", unauthorizedRecorder.Code, http.StatusUnauthorized, unauthorizedRecorder.Body.String())
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	authorizedRequest.Header.Set(adminCredentialHeader, "admin-key")
	authorizedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(authorizedRecorder, authorizedRequest)
	if authorizedRecorder.Code != http.StatusOK {
		t.Fatalf("authorized status = %d, want %d: %s", authorizedRecorder.Code, http.StatusOK, authorizedRecorder.Body.String())
	}
	if body := authorizedRecorder.Body.String(); !strings.Contains(body, "viaduct_http_requests_total") {
		t.Fatalf("metrics response missing expected counter: %s", body)
	}
}

func TestServer_HandleHealthzAndReadyz_ReturnsExpectedStatus_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	healthRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRecorder := httptest.NewRecorder()
	handler.ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d: %s", healthRecorder.Code, http.StatusOK, healthRecorder.Body.String())
	}

	readinessRequest := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readinessRecorder := httptest.NewRecorder()
	handler.ServeHTTP(readinessRecorder, readinessRequest)
	if readinessRecorder.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, want %d: %s", readinessRecorder.Code, http.StatusOK, readinessRecorder.Body.String())
	}

	pingRequest := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	pingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(pingRecorder, pingRequest)
	if pingRecorder.Code != http.StatusOK {
		t.Fatalf("ping status = %d, want %d: %s", pingRecorder.Code, http.StatusOK, pingRecorder.Body.String())
	}

	var readiness readinessResponse
	if err := json.Unmarshal(readinessRecorder.Body.Bytes(), &readiness); err != nil {
		t.Fatalf("Unmarshal(readiness) error = %v", err)
	}
	if readiness.Status != "ready" {
		t.Fatalf("readiness.Status = %q, want ready", readiness.Status)
	}
	if !readiness.PoliciesLoaded {
		t.Fatal("PoliciesLoaded = false, want true")
	}
}

func TestServer_Handler_AuthSessionRouteRateLimited_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.authRateLimiter = newTenantRateLimiter(1, time.Minute)
	handler := server.Handler()

	for index, expectedStatus := range []int{http.StatusCreated, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
		request.RemoteAddr = "203.0.113.10:41000"
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		if recorder.Code != expectedStatus {
			t.Fatalf("request %d status = %d, want %d: %s", index, recorder.Code, expectedStatus, recorder.Body.String())
		}
	}
}

func TestServer_Handler_AuthSessionRouteRateLimited_TwentyFirstRequestRejected_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.authRateLimiter = newTenantRateLimiter(20, time.Minute)
	handler := server.Handler()

	for index := 0; index < 21; index++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
		request.RemoteAddr = "203.0.113.10:41000"
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		expectedStatus := http.StatusCreated
		if index == 20 {
			expectedStatus = http.StatusTooManyRequests
		}
		if recorder.Code != expectedStatus {
			t.Fatalf("request %d status = %d, want %d: %s", index+1, recorder.Code, expectedStatus, recorder.Body.String())
		}
	}
}

func TestServer_Handler_LocalRuntimeAuthSession_IssuesOperatorSession_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetLocalRuntimeMode(true)
	server.SetBindHost("127.0.0.1")
	handler := server.Handler()

	createRequest := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/auth/session", bytes.NewBufferString(`{"mode":"local"}`))
	createRequest.RemoteAddr = "127.0.0.1:41000"
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Origin", "http://127.0.0.1")
	createRecorder := httptest.NewRecorder()

	handler.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	var createResponse authSessionResponse
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("Unmarshal(create auth session) error = %v", err)
	}
	if createResponse.Mode != "local" || createResponse.SessionID == "" {
		t.Fatalf("unexpected local auth session response: %#v", createResponse)
	}

	currentTenantRequest := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/tenants/current", nil)
	currentTenantRequest.RemoteAddr = "127.0.0.1:41000"
	for _, cookie := range createRecorder.Result().Cookies() {
		currentTenantRequest.AddCookie(cookie)
	}
	currentTenantRecorder := httptest.NewRecorder()

	handler.ServeHTTP(currentTenantRecorder, currentTenantRequest)
	if currentTenantRecorder.Code != http.StatusOK {
		t.Fatalf("current tenant status = %d, want %d: %s", currentTenantRecorder.Code, http.StatusOK, currentTenantRecorder.Body.String())
	}

	var currentTenant currentTenantResponse
	if err := json.Unmarshal(currentTenantRecorder.Body.Bytes(), &currentTenant); err != nil {
		t.Fatalf("Unmarshal(current tenant) error = %v", err)
	}
	if currentTenant.Role != models.TenantRoleOperator {
		t.Fatalf("currentTenant.Role = %q, want %q", currentTenant.Role, models.TenantRoleOperator)
	}
	if currentTenant.AuthMethod != "local-runtime-session" {
		t.Fatalf("currentTenant.AuthMethod = %q, want local-runtime-session", currentTenant.AuthMethod)
	}
}

func TestServer_Handler_LocalRuntimeProtectedRoute_RejectsAmbientLoopbackRequest_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetLocalRuntimeMode(true)
	server.SetBindHost("127.0.0.1")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/tenants/current", nil)
	request.RemoteAddr = "127.0.0.1:41000"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
}

func TestServer_Handler_LocalRuntimeAuthSession_RejectsNonLoopback_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetLocalRuntimeMode(true)
	server.SetBindHost("127.0.0.1")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/auth/session", bytes.NewBufferString(`{"mode":"local"}`))
	request.RemoteAddr = "203.0.113.10:41000"
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
}

func TestServer_Handler_LocalRuntimeAuthSession_RejectsForwardedProxyRequest_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetLocalRuntimeMode(true)
	server.SetBindHost("127.0.0.1")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPost, "https://viaduct.example.com/api/v1/auth/session", bytes.NewBufferString(`{"mode":"local"}`))
	request.RemoteAddr = "127.0.0.1:41000"
	request.Host = "viaduct.example.com"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("Origin", "https://viaduct.example.com")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
}

func TestServer_Handler_AuthSessionCookieSecure_TrustedForwardedProtoOnlyWhenProxyTrusted_Expected(t *testing.T) {
	t.Setenv("VIADUCT_TRUSTED_PROXIES", "10.0.0.0/8")

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.SetBindHost("0.0.0.0")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPost, "http://viaduct.example.com/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
	request.RemoteAddr = "10.10.10.10:41000"
	request.Host = "viaduct.example.com"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].Secure {
		t.Fatalf("session cookie secure = %t, want true", len(cookies) > 0 && cookies[0].Secure)
	}
}

func TestServer_Handler_AuthSessionCookieSecure_IgnoresForwardedProtoWithoutTrustedProxy_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.SetBindHost("0.0.0.0")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPost, "http://viaduct.example.com/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
	request.RemoteAddr = "10.10.10.10:41000"
	request.Host = "viaduct.example.com"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Secure {
		t.Fatalf("session cookie secure = %t, want false", len(cookies) > 0 && cookies[0].Secure)
	}
}

func TestServer_Handler_AuthSessionCookieSecure_IgnoresForwardedProtoOnLoopbackBind_Expected(t *testing.T) {
	t.Setenv("VIADUCT_TRUSTED_PROXIES", "10.0.0.0/8")

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.SetBindHost("127.0.0.1")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
	request.RemoteAddr = "10.10.10.10:41000"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	request.Header.Set("X-Forwarded-Proto", "https")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Secure {
		t.Fatalf("session cookie secure = %t, want false", len(cookies) > 0 && cookies[0].Secure)
	}
}

func TestServer_Handler_DeleteAuthSession_RevokesCurrentSession_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	handler := server.Handler()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
	createRequest.RemoteAddr = "203.0.113.10:41000"
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	handler.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/session", nil)
	deleteRequest.RemoteAddr = "203.0.113.10:41000"
	for _, cookie := range createRecorder.Result().Cookies() {
		deleteRequest.AddCookie(cookie)
	}
	deleteRecorder := httptest.NewRecorder()
	handler.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d: %s", deleteRecorder.Code, http.StatusOK, deleteRecorder.Body.String())
	}

	currentTenantRequest := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/current", nil)
	currentTenantRequest.RemoteAddr = "203.0.113.10:41000"
	for _, cookie := range createRecorder.Result().Cookies() {
		currentTenantRequest.AddCookie(cookie)
	}
	currentTenantRecorder := httptest.NewRecorder()
	handler.ServeHTTP(currentTenantRecorder, currentTenantRequest)
	if currentTenantRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("current tenant status = %d, want %d: %s", currentTenantRecorder.Code, http.StatusUnauthorized, currentTenantRecorder.Body.String())
	}
}

func TestServer_Handler_AdminAuthSessionRevoke_BlocksSessionLookup_Expected(t *testing.T) {
	t.Setenv("VIADUCT_ADMIN_KEY", "admin-secret")

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	handler := server.Handler()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session", bytes.NewBufferString(`{"mode":"tenant","api_key":"tenant-a-key"}`))
	createRequest.RemoteAddr = "203.0.113.10:41000"
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	handler.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	var sessionResponse authSessionResponse
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &sessionResponse); err != nil {
		t.Fatalf("Unmarshal(create auth session) error = %v", err)
	}

	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session/revoke", bytes.NewBufferString(fmt.Sprintf(`{"session_id":%q}`, sessionResponse.SessionID)))
	revokeRequest.RemoteAddr = "203.0.113.10:41000"
	revokeRequest.Header.Set("Content-Type", "application/json")
	revokeRequest.Header.Set(adminCredentialHeader, "admin-secret")
	revokeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(revokeRecorder, revokeRequest)
	if revokeRecorder.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want %d: %s", revokeRecorder.Code, http.StatusOK, revokeRecorder.Body.String())
	}

	currentTenantRequest := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/current", nil)
	currentTenantRequest.RemoteAddr = "203.0.113.10:41000"
	for _, cookie := range createRecorder.Result().Cookies() {
		currentTenantRequest.AddCookie(cookie)
	}
	currentTenantRecorder := httptest.NewRecorder()
	handler.ServeHTTP(currentTenantRecorder, currentTenantRequest)
	if currentTenantRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("current tenant status = %d, want %d: %s", currentTenantRecorder.Code, http.StatusUnauthorized, currentTenantRecorder.Body.String())
	}
}

func TestNewServer_PersistentAuthSessionTTL_DefaultAndOverride_Expected(t *testing.T) {
	stateStore := store.NewMemoryStore()

	defaultServer := mustNewServer(t, stateStore)
	if defaultServer.authSessions.persistentTTL != 7*24*time.Hour {
		t.Fatalf("default persistentTTL = %s, want %s", defaultServer.authSessions.persistentTTL, 7*24*time.Hour)
	}

	t.Setenv("VIADUCT_LONG_SESSION_DAYS", "21")
	overrideServer := mustNewServer(t, stateStore)
	if overrideServer.authSessions.persistentTTL != 21*24*time.Hour {
		t.Fatalf("override persistentTTL = %s, want %s", overrideServer.authSessions.persistentTTL, 21*24*time.Hour)
	}
}

func TestServer_HandleAbout_HidesLocalOperatorSessionForNonLocalRequest_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetLocalRuntimeMode(true)
	server.SetBindHost("127.0.0.1")

	request := httptest.NewRequest(http.MethodGet, "https://viaduct.example.com/api/v1/about", nil)
	request.RemoteAddr = "127.0.0.1:41000"
	request.Host = "viaduct.example.com"
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	recorder := httptest.NewRecorder()

	server.handleAbout(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response aboutResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal(about) error = %v", err)
	}
	if response.LocalOperatorSession {
		t.Fatal("LocalOperatorSession = true, want false for forwarded non-local request")
	}
}

func TestServer_Handler_TenantRoutesUseClientLimiter_NotAuthLimiter(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.clientRateLimiter = newTenantRateLimiter(5, time.Minute)
	server.authRateLimiter = newTenantRateLimiter(1, time.Minute)
	handler := server.Handler()

	for index := 0; index < 2; index++ {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/current", nil)
		request.RemoteAddr = "203.0.113.10:41000"
		request.Header.Set("X-API-Key", "tenant-a-key")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d: %s", index, recorder.Code, http.StatusOK, recorder.Body.String())
		}
	}
}

func TestServer_HandleInventory_V1ShapePreserved_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)

	for index := 0; index < 2; index++ {
		_, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, &models.DiscoveryResult{
			Source:       fmt.Sprintf("source-%d", index),
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 10, 11, index, 0, 0, time.UTC),
			VMs: []models.VirtualMachine{
				{ID: fmt.Sprintf("vm-%d", index), Name: fmt.Sprintf("vm-%d", index), Platform: models.PlatformVMware},
			},
		})
		if err != nil {
			t.Fatalf("SaveDiscovery(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleInventory(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response models.DiscoveryResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response.VMs) != 2 {
		t.Fatalf("len(response.VMs) = %d, want 2", len(response.VMs))
	}
}

func TestServer_HandleInventory_V2PaginationEnvelope_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)

	for index := 0; index < 3; index++ {
		_, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, &models.DiscoveryResult{
			Source:       fmt.Sprintf("source-%d", index),
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 10, 11, index, 0, 0, time.UTC),
			VMs: []models.VirtualMachine{
				{ID: fmt.Sprintf("vm-%d", index), Name: fmt.Sprintf("vm-%d", index), Platform: models.PlatformVMware},
			},
		})
		if err != nil {
			t.Fatalf("SaveDiscovery(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/inventory?page=2&per_page=1", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleInventory(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Inventory  models.DiscoveryResult `json:"inventory"`
		Pagination paginationResponse     `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Pagination.Total != 3 || response.Pagination.Page != 2 || response.Pagination.PerPage != 1 || response.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination: %#v", response.Pagination)
	}
	if len(response.Inventory.VMs) != 1 {
		t.Fatalf("len(response.Inventory.VMs) = %d, want 1", len(response.Inventory.VMs))
	}
}

func TestServer_HandleSnapshots_UsesStorePagination_Expected(t *testing.T) {
	t.Parallel()

	baseStore := store.NewMemoryStore()
	probeStore := &paginationProbeStore{Store: baseStore}
	server := mustNewServer(t, probeStore)

	for index := 0; index < 3; index++ {
		_, err := baseStore.SaveDiscovery(context.Background(), store.DefaultTenantID, &models.DiscoveryResult{
			Source:       "source",
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 10, 12, index, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("SaveDiscovery(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/snapshots?page=2&per_page=1", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleSnapshots(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	if probeStore.listSnapshotsPageCalls != 1 {
		t.Fatalf("ListSnapshotsPage() calls = %d, want 1", probeStore.listSnapshotsPageCalls)
	}
	if probeStore.listSnapshotsCalls != 0 {
		t.Fatalf("ListSnapshots() calls = %d, want 0", probeStore.listSnapshotsCalls)
	}

	var response pagedItemsResponse[store.SnapshotMeta]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Pagination.Total != 3 || response.Pagination.Page != 2 || response.Pagination.PerPage != 1 || response.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination: %#v", response.Pagination)
	}
	if len(response.Items) != 1 {
		t.Fatalf("len(response.Items) = %d, want 1", len(response.Items))
	}
	if got, want := response.Items[0].DiscoveredAt, time.Date(2026, time.April, 10, 12, 1, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("response.Items[0].DiscoveredAt = %s, want %s", got, want)
	}
}

func TestServer_HandleSnapshots_V1ShapePreserved_Expected(t *testing.T) {
	t.Parallel()

	baseStore := store.NewMemoryStore()
	probeStore := &paginationProbeStore{Store: baseStore}
	server := mustNewServer(t, probeStore)

	for index := 0; index < 2; index++ {
		_, err := baseStore.SaveDiscovery(context.Background(), store.DefaultTenantID, &models.DiscoveryResult{
			Source:       fmt.Sprintf("source-%d", index),
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Date(2026, time.April, 10, 12, index, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("SaveDiscovery(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleSnapshots(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	if probeStore.listSnapshotsCalls != 1 {
		t.Fatalf("ListSnapshots() calls = %d, want 1", probeStore.listSnapshotsCalls)
	}
	if probeStore.listSnapshotsPageCalls != 0 {
		t.Fatalf("ListSnapshotsPage() calls = %d, want 0", probeStore.listSnapshotsPageCalls)
	}

	var response []store.SnapshotMeta
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response) != 2 {
		t.Fatalf("len(response) = %d, want 2", len(response))
	}
}

func TestServer_HandleMigrations_UsesStorePagination_Expected(t *testing.T) {
	t.Parallel()

	baseStore := store.NewMemoryStore()
	probeStore := &paginationProbeStore{Store: baseStore}
	server := mustNewServer(t, probeStore)

	for index := 0; index < 3; index++ {
		err := baseStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
			ID:        fmt.Sprintf("migration-%d", index),
			TenantID:  store.DefaultTenantID,
			SpecName:  "phase-test",
			Phase:     string(migratepkg.PhasePlan),
			StartedAt: time.Date(2026, time.April, 10, 14, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, time.April, 10, 14, index, 0, 0, time.UTC),
			RawJSON:   json.RawMessage(`{"phase":"plan"}`),
		})
		if err != nil {
			t.Fatalf("SaveMigration(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/migrations?page=2&per_page=1", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleMigrations(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	if probeStore.listMigrationsPageCalls != 1 {
		t.Fatalf("ListMigrationsPage() calls = %d, want 1", probeStore.listMigrationsPageCalls)
	}
	if probeStore.listMigrationsCalls != 0 {
		t.Fatalf("ListMigrations() calls = %d, want 0", probeStore.listMigrationsCalls)
	}

	var response pagedItemsResponse[store.MigrationMeta]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Pagination.Total != 3 || response.Pagination.Page != 2 || response.Pagination.PerPage != 1 || response.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination: %#v", response.Pagination)
	}
	if len(response.Items) != 1 {
		t.Fatalf("len(response.Items) = %d, want 1", len(response.Items))
	}
	if got, want := response.Items[0].UpdatedAt, time.Date(2026, time.April, 10, 14, 1, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("response.Items[0].UpdatedAt = %s, want %s", got, want)
	}
}

func TestServer_HandleMigrations_V1ShapePreserved_Expected(t *testing.T) {
	t.Parallel()

	baseStore := store.NewMemoryStore()
	probeStore := &paginationProbeStore{Store: baseStore}
	server := mustNewServer(t, probeStore)

	for index := 0; index < 2; index++ {
		err := baseStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
			ID:        fmt.Sprintf("migration-v1-%d", index),
			TenantID:  store.DefaultTenantID,
			SpecName:  "phase-test",
			Phase:     string(migratepkg.PhasePlan),
			StartedAt: time.Date(2026, time.April, 10, 14, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, time.April, 10, 14, index, 0, 0, time.UTC),
			RawJSON:   json.RawMessage(`{"phase":"plan"}`),
		})
		if err != nil {
			t.Fatalf("SaveMigration(%d) error = %v", index, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/migrations", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleMigrations(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	if probeStore.listMigrationsCalls != 1 {
		t.Fatalf("ListMigrations() calls = %d, want 1", probeStore.listMigrationsCalls)
	}
	if probeStore.listMigrationsPageCalls != 0 {
		t.Fatalf("ListMigrationsPage() calls = %d, want 0", probeStore.listMigrationsPageCalls)
	}

	var response []store.MigrationMeta
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(response) != 2 {
		t.Fatalf("len(response) = %d, want 2", len(response))
	}
}

func TestServer_Handler_CORSAndSecurityHeaders_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	for header, want := range map[string]string{
		"Cache-Control":           "no-store",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
		"Permissions-Policy":      "camera=(), geolocation=(), microphone=()",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
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

	server := mustNewServer(t, store.NewMemoryStore())
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

func TestServer_Handler_ExplicitAllowedCORSOrigin_Expected(t *testing.T) {
	t.Setenv("VIADUCT_ALLOWED_ORIGINS", "http://localhost:5173")
	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want explicit configured origin", got)
	}
}

func TestServer_Handler_ExplicitAllowedCORSOrigin_CaseInsensitive_Expected(t *testing.T) {
	t.Setenv("VIADUCT_ALLOWED_ORIGINS", "HTTP://LOCALHOST:5173")
	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want lowercase request origin", got)
	}
}

func TestNewServer_WildcardAllowedOriginsWithAuthEnabled_ReturnsError(t *testing.T) {
	t.Setenv("VIADUCT_ADMIN_KEY", "admin-key")
	t.Setenv("VIADUCT_ALLOWED_ORIGINS", "*")

	_, err := NewServer(nil, store.NewMemoryStore(), 0, nil)
	if err == nil {
		t.Fatal("NewServer() error = nil, want wildcard CORS validation failure")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "VIADUCT_ALLOWED_ORIGINS") || !strings.Contains(got, "*") {
		t.Fatalf("NewServer() error = %q, want wildcard CORS validation context", got)
	}
}

func TestServer_HandlePreflight_InvalidSpecReturnsFieldErrors_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
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

func TestServer_BackgroundTaskContext_ServerLifetimeAndMetadata_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	parentCtx, parentCancel := context.WithCancel(
		ContextWithRequestID(
			store.ContextWithTenantID(context.Background(), "tenant-background"),
			"req-background",
		),
	)
	defer parentCancel()

	ctx, cancel := server.backgroundTaskContext(parentCtx, "", "")
	defer cancel()

	parentCancel()
	select {
	case <-ctx.Done():
		t.Fatal("background task context canceled with parent request")
	default:
	}
	if got := store.TenantIDFromContext(ctx); got != "tenant-background" {
		t.Fatalf("TenantIDFromContext() = %q, want tenant-background", got)
	}
	if got := RequestIDFromContext(ctx); got != "req-background" {
		t.Fatalf("RequestIDFromContext() = %q, want req-background", got)
	}

	server.cancelBackgroundTasks()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("background task context not canceled by server shutdown")
	}
}

func TestServer_RunMigrationAsync_RecoversPanicsAndLogs_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	var logBuffer bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))

	server.runMigrationAsync(
		ContextWithRequestID(store.ContextWithTenantID(context.Background(), "tenant-panic"), "req-panic"),
		"",
		"",
		"execute",
		"migration-panic",
		func(ctx context.Context) error {
			if got := store.TenantIDFromContext(ctx); got != "tenant-panic" {
				t.Fatalf("TenantIDFromContext() = %q, want tenant-panic", got)
			}
			if got := RequestIDFromContext(ctx); got != "req-panic" {
				t.Fatalf("RequestIDFromContext() = %q, want req-panic", got)
			}
			panic("boom")
		},
	)

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"msg":"migration background task panicked"`,
		`"action":"execute"`,
		`"migration_id":"migration-panic"`,
		`"tenant_id":"tenant-panic"`,
		`"request_id":"req-panic"`,
		`"panic":"boom"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("log output missing %s: %s", fragment, logOutput)
		}
	}
}

func TestServer_ValidateBindSecurity_RemoteBindWithoutCredentialsRejected_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetBindHost("0.0.0.0")

	err := server.validateBindSecurity(context.Background())
	if err == nil {
		t.Fatal("validateBindSecurity() error = nil, want remote bind rejection")
	}
	if !strings.Contains(err.Error(), "remote bind without configured authentication is refused") {
		t.Fatalf("validateBindSecurity() error = %q, want remote bind refusal", err)
	}
}

func TestServer_ValidateBindSecurity_LoopbackWithoutCredentialsAllowed_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetBindHost("127.0.0.1")

	if err := server.validateBindSecurity(context.Background()); err != nil {
		t.Fatalf("validateBindSecurity() error = %v", err)
	}
}

func TestServer_ValidateBindSecurity_RemoteBindAllowedWithConfiguredCredentials_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	server.SetBindHost("0.0.0.0")

	if err := server.validateBindSecurity(context.Background()); err != nil {
		t.Fatalf("validateBindSecurity() error = %v", err)
	}
}

func TestServer_ValidateBindSecurity_RemoteBindAllowsExplicitDangerousOverride_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.SetBindHost("0.0.0.0")
	server.SetAllowUnauthenticatedRemote(true)

	if err := server.validateBindSecurity(context.Background()); err != nil {
		t.Fatalf("validateBindSecurity() error = %v", err)
	}
}
