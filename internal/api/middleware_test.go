package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestTenantAuthMiddleware_ValidKey_AllowsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, err := RequireTenant(r.Context())
		if err != nil {
			t.Fatalf("RequireTenant() error = %v", err)
		}
		if tenant.ID != "tenant-a" {
			t.Fatalf("tenant ID = %q, want tenant-a", tenant.ID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(tenantCredentialHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestTenantAuthMiddleware_InvalidKey_RejectsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(tenantCredentialHeader, "bad-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTenantAuthMiddleware_MissingKey_RejectsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	var response apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Error.Code != "missing_credentials" || response.Error.RequestID == "" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}

func TestTenantAuthMiddleware_TenantIsolation_ScopesStoreAccess(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	ctx := context.Background()
	if _, err := stateStore.SaveDiscovery(ctx, "tenant-a", &models.DiscoveryResult{
		Source:   "tenant-a-source",
		Platform: models.PlatformVMware,
	}); err != nil {
		t.Fatalf("SaveDiscovery(tenant-a) error = %v", err)
	}
	if _, err := stateStore.SaveDiscovery(ctx, "tenant-b", &models.DiscoveryResult{
		Source:   "tenant-b-source",
		Platform: models.PlatformProxmox,
	}); err != nil {
		t.Fatalf("SaveDiscovery(tenant-b) error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := stateStore.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 10)
		if err != nil {
			t.Fatalf("ListSnapshots() error = %v", err)
		}
		if len(items) != 1 || items[0].Source != "tenant-a-source" {
			t.Fatalf("unexpected tenant-scoped snapshots: %#v", items)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
	req.Header.Set(tenantCredentialHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestTenantAuthMiddleware_ServiceAccountKey_AllowsScopedRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-viewer",
				Name:      "Viewer",
				APIKey:    "service-key",
				Role:      models.TenantRoleViewer,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := RequirePrincipal(r.Context())
		if err != nil {
			t.Fatalf("RequirePrincipal() error = %v", err)
		}
		if principal.ServiceAccount == nil || principal.ServiceAccount.ID != "sa-viewer" {
			t.Fatalf("unexpected principal: %#v", principal)
		}
		if principal.Role != models.TenantRoleViewer {
			t.Fatalf("principal.Role = %q, want viewer", principal.Role)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(serviceAccountCredentialHeader, "service-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRequireTenantRole_ViewerDeniedOperatorRoute_Expected(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-viewer",
				Name:      "Viewer",
				APIKey:    "service-key",
				Role:      models.TenantRoleViewer,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, RequireTenantRole(models.TenantRoleOperator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", nil)
	req.Header.Set(serviceAccountCredentialHeader, "service-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestRequireTenantPermission_ServiceAccountExplicitScopeDenied_Expected(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:          "sa-inventory",
				Name:        "Inventory Only",
				APIKey:      "inventory-key",
				Role:        models.TenantRoleAdmin,
				Permissions: []models.TenantPermission{models.TenantPermissionInventoryRead},
				Active:      true,
				CreatedAt:   time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, RequireTenantPermission(models.TenantPermissionMigrationManage, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations", nil)
	req.Header.Set(serviceAccountCredentialHeader, "inventory-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestRequireTenantPermission_ServiceAccountExplicitScopeAllows_Expected(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:          "sa-reports",
				Name:        "Reports",
				APIKey:      "reports-key",
				Role:        models.TenantRoleViewer,
				Permissions: []models.TenantPermission{models.TenantPermissionReportsRead},
				Active:      true,
				CreatedAt:   time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, RequireTenantPermission(models.TenantPermissionReportsRead, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/audit", nil)
	req.Header.Set(serviceAccountCredentialHeader, "reports-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRequireAnyTenantPermission_ServiceAccountWithAlternatePermissionAllows_Expected(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:          "sa-operator",
				Name:        "Operator",
				APIKey:      "operator-key",
				Role:        models.TenantRoleOperator,
				Permissions: []models.TenantPermission{models.TenantPermissionMigrationManage},
				Active:      true,
				CreatedAt:   time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, RequireTenantRole(models.TenantRoleViewer, RequireAnyTenantPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), models.TenantPermissionReportsRead, models.TenantPermissionMigrationManage)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set(serviceAccountCredentialHeader, "operator-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestTenantAuthMiddleware_RejectsLoopbackWithoutExplicitCredentials_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected unauthenticated access through tenant middleware")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.RemoteAddr = "127.0.0.1:41000"
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
}

func TestLocalRuntimeRequestAllowed_RejectsForwardedProxyRequest_Expected(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "http://viaduct.example.local/api/v1/auth/session", nil)
	req.RemoteAddr = "127.0.0.1:41000"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")

	if localRuntimeRequestAllowed(req, "127.0.0.1") {
		t.Fatal("localRuntimeRequestAllowed() = true, want false for forwarded proxy request")
	}
}

func TestTenantAuthMiddleware_ContextTenantMismatch_ReturnsForbidden(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), "tenant-b"))
	req.Header.Set(tenantCredentialHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
}

func newTenantTestStore(t *testing.T) store.Store {
	t.Helper()

	stateStore := store.NewMemoryStore()
	for _, tenant := range []models.Tenant{
		{ID: "tenant-a", Name: "Tenant A", APIKey: "tenant-a-key", Active: true},
		{ID: "tenant-b", Name: "Tenant B", APIKey: "tenant-b-key", Active: true},
	} {
		if err := stateStore.CreateTenant(context.Background(), tenant); err != nil {
			t.Fatalf("CreateTenant(%s) error = %v", tenant.ID, err)
		}
	}
	return stateStore
}
