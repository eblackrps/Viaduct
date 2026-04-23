package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_WithObservability_AddsRequestIDAndMetrics_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.withObservability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("request ID missing from context")
		}
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/migration-1/execute", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if strings.TrimSpace(recorder.Header().Get(requestIDHeader)) == "" {
		t.Fatal("X-Request-ID header is empty")
	}

	metrics := server.metrics.render()
	if !strings.Contains(metrics, `path="/api/v1/migrations/:id/execute"`) {
		t.Fatalf("metrics output missing normalized route: %s", metrics)
	}
	if !strings.Contains(metrics, `status="201"`) {
		t.Fatalf("metrics output missing created status: %s", metrics)
	}
}

func TestServer_WithObservability_ExtractsTraceparent_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	handler := server.withObservability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if TraceIDFromContext(r.Context()) != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Fatalf("TraceIDFromContext() = %q, want propagated trace ID", TraceIDFromContext(r.Context()))
		}
		scope := requestScopeFromContext(r.Context())
		if scope == nil {
			t.Fatal("request scope missing from context")
		}
		if scope.traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Fatalf("scope.traceID = %q, want propagated trace ID", scope.traceID)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/about", nil)
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if recorder.Header().Get(traceIDHeader) != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("X-Trace-ID header = %q, want propagated trace ID", recorder.Header().Get(traceIDHeader))
	}
}

func TestServer_WithObservability_CapturesAuthenticatedTenantScope_Expected(t *testing.T) {
	t.Parallel()

	stateStore := newTenantTestStore(t)
	server := mustNewServer(t, stateStore)
	handler := server.withObservability(TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := requestScopeFromContext(r.Context())
		if scope == nil {
			t.Fatal("request scope missing from context")
		}
		if scope.tenantID != "tenant-a" {
			t.Fatalf("scope.tenantID = %q, want tenant-a", scope.tenantID)
		}
		if scope.authMethod != "tenant-api-key" {
			t.Fatalf("scope.authMethod = %q, want tenant-api-key", scope.authMethod)
		}
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	req.Header.Set(tenantCredentialHeader, "tenant-a-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestTenantRateLimitMiddleware_LimitExceeded_ReturnsTooManyRequests(t *testing.T) {
	t.Parallel()

	handler := TenantRateLimitMiddleware(newTenantRateLimiter(1, time.Minute), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for idx, expectedStatus := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
		req = req.WithContext(store.ContextWithTenantID(req.Context(), "tenant-a"))
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
		if recorder.Code != expectedStatus {
			t.Fatalf("request %d status = %d, want %d", idx, recorder.Code, expectedStatus)
		}
		if expectedStatus == http.StatusTooManyRequests {
			var response apiErrorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if response.Error.Code != "rate_limit_exceeded" || response.Error.RequestID == "" {
				t.Fatalf("unexpected rate-limit error response: %#v", response)
			}
		}
	}
}

func TestTenantRateLimitMiddleware_UsesTenantQuotaOverride_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		Quotas: models.TenantQuota{
			RequestsPerMinute: 1,
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	handler := TenantAuthMiddleware(stateStore, TenantRateLimitMiddleware(newTenantRateLimiter(5, time.Minute), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	for idx, expectedStatus := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
		req.Header.Set(tenantCredentialHeader, "tenant-a-key")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
		if recorder.Code != expectedStatus {
			t.Fatalf("request %d status = %d, want %d", idx, recorder.Code, expectedStatus)
		}
	}
}

func TestClientRateLimitMiddleware_LimitExceeded_ReturnsTooManyRequests(t *testing.T) {
	t.Parallel()

	handler := ClientRateLimitMiddleware(newTenantRateLimiter(1, time.Minute), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for idx, expectedStatus := range []int{http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/about", nil)
		req.RemoteAddr = "203.0.113.10:41000"
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
		if recorder.Code != expectedStatus {
			t.Fatalf("request %d status = %d, want %d", idx, recorder.Code, expectedStatus)
		}
	}
}

func TestServer_HandleMetrics_OperationalMetricsIncluded_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		Quotas: models.TenantQuota{
			MaxSnapshots:  3,
			MaxMigrations: 4,
		},
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
	if _, err := stateStore.SaveDiscovery(context.Background(), "tenant-a", &models.DiscoveryResult{
		Source:       "tenant-a-source",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}
	if err := stateStore.SaveMigration(context.Background(), "tenant-a", store.MigrationRecord{
		ID:        "migration-1",
		TenantID:  "tenant-a",
		SpecName:  "metrics",
		Phase:     "complete",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	recorder := httptest.NewRecorder()

	server.handleMetrics(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `viaduct_store_info{backend="memory",persistent="false"} 1`) {
		t.Fatalf("metrics missing store info: %s", body)
	}
	if !strings.Contains(body, `viaduct_tenant_service_accounts{tenant="tenant-a"} 1`) {
		t.Fatalf("metrics missing service-account gauge: %s", body)
	}
	if !strings.Contains(body, `viaduct_tenant_quota_remaining{tenant="tenant-a",resource="snapshots"} 2`) {
		t.Fatalf("metrics missing snapshot quota gauge: %s", body)
	}
	if !strings.Contains(body, `viaduct_tenant_migrations{tenant="tenant-a",phase="complete"} 1`) {
		t.Fatalf("metrics missing migration phase gauge: %s", body)
	}
}

func TestServer_HandleMetrics_IncludesWorkspaceQueueDepth_Expected(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	server.workspaceJobExecutor = workspaceQueueDepthStub{
		depths: map[string]int{
			"tenant-a": 2,
			"tenant-b": 1,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	recorder := httptest.NewRecorder()

	server.handleMetrics(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `viaduct_workspace_queue_depth{tenant="tenant-a"} 2`) {
		t.Fatalf("metrics missing tenant-a queue depth: %s", body)
	}
	if !strings.Contains(body, `viaduct_workspace_queue_depth{tenant="tenant-b"} 1`) {
		t.Fatalf("metrics missing tenant-b queue depth: %s", body)
	}
}

type workspaceQueueDepthStub struct {
	depths map[string]int
}

func (s workspaceQueueDepthStub) Enqueue(context.Context, workspaceJobTask) error {
	return nil
}

func (s workspaceQueueDepthStub) QueueDepthByTenant() map[string]int {
	return s.depths
}
