package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_WithObservability_AddsRequestIDAndMetrics_Expected(t *testing.T) {
	t.Parallel()

	server := NewServer(nil, store.NewMemoryStore(), 0, nil)
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
	}
}
