package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_HandleCurrentTenant_ServiceAccountResponseRedactsKeys_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		Quotas: models.TenantQuota{RequestsPerMinute: 42},
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-1",
				Name:      "Automation",
				APIKey:    "sa-1-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(server.handleCurrentTenant))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/current", nil)
	req.Header.Set(serviceAccountCredentialHeader, "sa-1-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response currentTenantResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.ServiceAccountID != "sa-1" || response.Role != models.TenantRoleOperator {
		t.Fatalf("unexpected current tenant response: %#v", response)
	}
	if len(response.Permissions) == 0 {
		t.Fatalf("Permissions is empty: %#v", response)
	}
	if response.Quotas.RequestsPerMinute != 42 {
		t.Fatalf("RequestsPerMinute = %d, want 42", response.Quotas.RequestsPerMinute)
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("sa-1-key")) {
		t.Fatal("current tenant response leaked service-account API key")
	}
}

func TestServer_HandleServiceAccounts_CreateAndRotate_Expected(t *testing.T) {
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
	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/service-accounts":
			RequireTenantRole(models.TenantRoleAdmin, http.HandlerFunc(server.handleServiceAccounts)).ServeHTTP(w, r)
		case "/api/v1/service-accounts/sa-1/rotate":
			RequireTenantRole(models.TenantRoleAdmin, http.HandlerFunc(server.handleServiceAccountByID)).ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/service-accounts", bytes.NewBufferString(`{"id":"sa-1","name":"Automation","role":"operator","permissions":["migration.manage","tenant.read"]}`))
	createRequest.Header.Set(tenantCredentialHeader, "tenant-a-key")
	createRecorder := httptest.NewRecorder()

	handler.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createRecorder.Code, http.StatusCreated, createRecorder.Body.String())
	}

	var created serviceAccountResponse
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("Unmarshal(created) error = %v", err)
	}
	if created.APIKey == "" || created.Role != models.TenantRoleOperator {
		t.Fatalf("unexpected created service account: %#v", created)
	}
	if len(created.Permissions) != 2 {
		t.Fatalf("unexpected created permissions: %#v", created)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/service-accounts", nil)
	listRequest.Header.Set(tenantCredentialHeader, "tenant-a-key")
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", listRecorder.Code, http.StatusOK, listRecorder.Body.String())
	}
	if bytes.Contains(listRecorder.Body.Bytes(), []byte(created.APIKey)) {
		t.Fatal("list response leaked service-account API key")
	}

	rotateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/service-accounts/sa-1/rotate", bytes.NewBufferString(`{"api_key":"rotated-key"}`))
	rotateRequest.Header.Set(tenantCredentialHeader, "tenant-a-key")
	rotateRecorder := httptest.NewRecorder()

	handler.ServeHTTP(rotateRecorder, rotateRequest)
	if rotateRecorder.Code != http.StatusOK {
		t.Fatalf("rotate status = %d, want %d: %s", rotateRecorder.Code, http.StatusOK, rotateRecorder.Body.String())
	}

	var rotated serviceAccountResponse
	if err := json.Unmarshal(rotateRecorder.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("Unmarshal(rotated) error = %v", err)
	}
	if rotated.APIKey != "rotated-key" || rotated.LastRotatedAt.IsZero() {
		t.Fatalf("unexpected rotated service account: %#v", rotated)
	}
}
