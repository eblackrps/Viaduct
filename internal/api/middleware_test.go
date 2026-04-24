package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestTenantAuthMiddleware_RotatedTenantKey_RejectsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	tenant, err := stateStore.GetTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	tenant.APIKey = "tenant-a-key-rotated"
	tenant.APIKeyHash = ""
	if err := stateStore.UpdateTenant(context.Background(), *tenant); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(tenantCredentialHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTenantAuthMiddleware_RotatedServiceAccountKey_RejectsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-operator",
				Name:      "Operator",
				APIKey:    "service-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	tenant, err := stateStore.GetTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	for index := range tenant.ServiceAccounts {
		if tenant.ServiceAccounts[index].ID == "sa-operator" {
			tenant.ServiceAccounts[index].APIKey = "service-key-rotated"
			tenant.ServiceAccounts[index].APIKeyHash = ""
		}
	}
	if err := stateStore.UpdateTenant(context.Background(), *tenant); err != nil {
		t.Fatalf("UpdateTenant(rotate service account) error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(serviceAccountCredentialHeader, "service-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTenantAuthMiddleware_DeletedServiceAccountSession_RejectsRequest(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-operator",
				Name:      "Operator",
				APIKey:    "service-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	principal, ok, err := findServiceAccountPrincipalByAPIKey(context.Background(), stateStore, "service-key")
	if err != nil {
		t.Fatalf("findServiceAccountPrincipalByAPIKey() error = %v", err)
	}
	if !ok {
		t.Fatal("findServiceAccountPrincipalByAPIKey() = false, want true")
	}

	sessions := newAuthSessionManager(time.Hour, 24*time.Hour)
	record, err := sessions.CreateCredential("service-account", principal, hashCredential(context.Background(), "service-key"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:              "tenant-a",
		Name:            "Tenant A",
		APIKey:          "tenant-a-key",
		Active:          true,
		ServiceAccounts: nil,
	}); err != nil {
		t.Fatalf("UpdateTenant(delete service account) error = %v", err)
	}

	server := &Server{store: stateStore, authSessions: sessions}
	handler := tenantAuthMiddleware(stateStore, sessions, server, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected access with deleted service-account session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookieName, Value: record.Secret})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if _, ok := sessions.Lookup(record.Secret); ok {
		t.Fatal("Lookup() = true, want deleted service-account session to be revoked")
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

func TestLocalRuntimeRequestAllowed_RejectionLogsAuditForMutatingRequests_Expected(t *testing.T) {
	var logBuffer bytes.Buffer
	originalLogger := packageLogger
	packageLogger = slog.New(slog.NewTextHandler(&logBuffer, nil))
	t.Cleanup(func() {
		packageLogger = originalLogger
	})

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/auth/session", nil)
	req.RemoteAddr = "127.0.0.1:41000"
	req.Header.Set("Origin", "http://evil.example")

	if localRuntimeRequestAllowed(req, "127.0.0.1") {
		t.Fatal("localRuntimeRequestAllowed() = true, want false for mismatched origin")
	}

	logLine := logBuffer.String()
	if !strings.Contains(logLine, "loopback_rejection") || !strings.Contains(logLine, "origin_mismatch") {
		t.Fatalf("rejection audit log = %q, want loopback rejection details", logLine)
	}
}

func TestLocalRuntimeRequestAllowed_RejectionLogsAuditForGetRequests_Expected(t *testing.T) {
	var logBuffer bytes.Buffer
	originalLogger := packageLogger
	packageLogger = slog.New(slog.NewTextHandler(&logBuffer, nil))
	t.Cleanup(func() {
		packageLogger = originalLogger
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/about", nil)
	req.RemoteAddr = "127.0.0.1:41000"
	req.Header.Set("Origin", "http://evil.example")

	if localRuntimeRequestAllowed(req, "127.0.0.1") {
		t.Fatal("localRuntimeRequestAllowed() = true, want false for mismatched GET origin")
	}

	logLine := logBuffer.String()
	if !strings.Contains(logLine, "loopback_rejection") || !strings.Contains(logLine, "method=GET") || !strings.Contains(logLine, "origin_mismatch") {
		t.Fatalf("rejection audit log = %q, want GET loopback rejection details", logLine)
	}
}

func TestRequestFromLoopback_NormalizesRemoteAddr_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{name: "ipv4 loopback", remoteAddr: "127.0.0.1:41000", want: true},
		{name: "ipv6 loopback", remoteAddr: "[::1]:41000", want: true},
		{name: "ipv4 mapped loopback", remoteAddr: "[::ffff:127.0.0.1]:41000", want: true},
		{name: "ipv4 mapped remote", remoteAddr: "[::ffff:8.8.8.8]:41000", want: false},
		{name: "ipv6 link local zone", remoteAddr: "[fe80::1%eth0]:41000", want: false},
		{name: "ipv6 loopback zone", remoteAddr: "[::1%lo]:41000", want: false},
		{name: "ipv4 zone rejected", remoteAddr: "127.0.0.1%0:41000", want: false},
		{name: "mapped ipv4 zone rejected", remoteAddr: "[::ffff:127.0.0.1%0]:41000", want: false},
		{name: "private remote", remoteAddr: "10.0.0.1:41000", want: false},
		{name: "empty", remoteAddr: "", want: false},
		{name: "garbage", remoteAddr: "not-an-ip", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/v1/about", nil)
			req.RemoteAddr = tc.remoteAddr

			if got := requestFromLoopback(req); got != tc.want {
				t.Fatalf("requestFromLoopback(%q) = %t, want %t", tc.remoteAddr, got, tc.want)
			}
		})
	}
}

func TestLocalRuntimeRequestAllowed_MutatingRequestsRequireSameOriginSource_Expected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		method  string
		origin  string
		referer string
		want    bool
	}{
		{name: "post same origin", method: http.MethodPost, origin: "http://127.0.0.1", want: true},
		{name: "post same origin explicit port", method: http.MethodPost, origin: "http://127.0.0.1:80", want: true},
		{name: "post referer fallback", method: http.MethodPost, referer: "http://127.0.0.1/settings", want: true},
		{name: "post missing source", method: http.MethodPost, want: false},
		{name: "post empty origin", method: http.MethodPost, origin: "   ", want: false},
		{name: "post wrong port", method: http.MethodPost, origin: "http://127.0.0.1:8080", want: false},
		{name: "post wrong host", method: http.MethodPost, origin: "http://localhost", want: false},
		{name: "get no source allowed", method: http.MethodGet, want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, "http://127.0.0.1/api/v1/auth/session", nil)
			req.RemoteAddr = "127.0.0.1:41000"
			if tc.origin != "" || strings.Contains(tc.name, "empty origin") {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}

			if got := localRuntimeRequestAllowed(req, "127.0.0.1"); got != tc.want {
				t.Fatalf("localRuntimeRequestAllowed(%s, origin=%q, referer=%q) = %t, want %t", tc.method, tc.origin, tc.referer, got, tc.want)
			}
		})
	}
}

func TestStoredCredentialHashMatches_HashedAndLegacySources_Expected(t *testing.T) {
	t.Parallel()

	digest := hashCredential(context.Background(), "tenant-secret")
	if !storedCredentialHashMatches(context.Background(), store.HashAPIKey("tenant-secret"), "", digest) {
		t.Fatal("storedCredentialHashMatches(hashed) = false, want true")
	}
	if !storedCredentialHashMatches(context.Background(), "", "tenant-secret", digest) {
		t.Fatal("storedCredentialHashMatches(legacy) = false, want true")
	}
}

func TestStoredCredentialHashMatches_InvalidOrZeroDigestRejected_Expected(t *testing.T) {
	t.Parallel()

	digest := hashCredential(context.Background(), "tenant-secret")
	if storedCredentialHashMatches(context.Background(), "invalid-hex", "", digest) {
		t.Fatal("storedCredentialHashMatches(invalid stored hash) = true, want false")
	}
	if storedCredentialHashMatches(context.Background(), store.HashAPIKey("tenant-secret"), "", [32]byte{}) {
		t.Fatal("storedCredentialHashMatches(zero expected hash) = true, want false")
	}
}

func TestConstantTimeEqual_EmptyInputsRejected_Expected(t *testing.T) {
	t.Parallel()

	if constantTimeEqual("", "") {
		t.Fatal("constantTimeEqual(empty, empty) = true, want false")
	}
	if constantTimeEqual("tenant-secret", "") {
		t.Fatal("constantTimeEqual(non-empty, empty) = true, want false")
	}
}

func TestAdminAuthMiddleware_PlaintextKeyAcceptedAndWarnsOnce_Expected(t *testing.T) {
	originalLogger := packageLogger
	var logBuffer bytes.Buffer
	packageLogger = slog.New(slog.NewTextHandler(&logBuffer, nil))
	defer func() {
		packageLogger = originalLogger
	}()

	handler := AdminAuthMiddleware("admin-secret", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
		req.Header.Set(adminCredentialHeader, "admin-secret")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", requestIndex, recorder.Code, http.StatusNoContent)
		}
	}

	logOutput := logBuffer.String()
	if strings.Count(logOutput, "legacy plaintext VIADUCT_ADMIN_KEY accepted") != 1 {
		t.Fatalf("warning count = %d, want 1; log=%q", strings.Count(logOutput, "legacy plaintext VIADUCT_ADMIN_KEY accepted"), logOutput)
	}
	if !strings.Contains(logOutput, "docs/operations/admin-key.md") {
		t.Fatalf("log output = %q, want admin-key migration guidance", logOutput)
	}
}

func TestAdminAuthMiddleware_HashedKeyAcceptedWithoutPlaintextWarning_Expected(t *testing.T) {
	originalLogger := packageLogger
	var logBuffer bytes.Buffer
	packageLogger = slog.New(slog.NewTextHandler(&logBuffer, nil))
	defer func() {
		packageLogger = originalLogger
	}()

	handler := AdminAuthMiddleware(store.HashAPIKey("admin-secret"), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/tenants", nil)
	req.Header.Set(adminCredentialHeader, "admin-secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if strings.Contains(logBuffer.String(), "legacy plaintext VIADUCT_ADMIN_KEY accepted") {
		t.Fatalf("log output = %q, want no plaintext warning for hashed key", logBuffer.String())
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

func TestTenantAuthMiddleware_ServiceAccountTenantMismatch_ReturnsForbidden(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	if err := stateStore.UpdateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-operator",
				Name:      "Operator",
				APIKey:    "service-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), "tenant-b"))
	req.Header.Set(serviceAccountCredentialHeader, "service-key")
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
