package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
	req.Header.Set(tenantAPIKeyHeader, "tenant-a-key")
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
	req.Header.Set(tenantAPIKeyHeader, "bad-key")
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
	req.Header.Set(tenantAPIKeyHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
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
