package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestServer_HandleAudit_TenantScoped_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	for _, event := range []models.AuditEvent{
		{
			ID:        "audit-tenant-a",
			TenantID:  "tenant-a",
			Actor:     "tenant:tenant-a",
			Category:  "migration",
			Action:    "execute",
			Resource:  "migration-1",
			Outcome:   models.AuditOutcomeSuccess,
			Message:   "started",
			CreatedAt: time.Date(2026, time.April, 5, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:        "audit-default",
			TenantID:  store.DefaultTenantID,
			Actor:     "tenant:default",
			Category:  "migration",
			Action:    "execute",
			Resource:  "migration-default",
			Outcome:   models.AuditOutcomeSuccess,
			Message:   "started",
			CreatedAt: time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC),
		},
	} {
		if err := stateStore.SaveAuditEvent(ctx, event); err != nil {
			t.Fatalf("SaveAuditEvent(%s) error = %v", event.ID, err)
		}
	}

	server := mustNewServer(t, stateStore)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), "tenant-a"))
	recorder := httptest.NewRecorder()

	server.handleAudit(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var events []models.AuditEvent
	if err := json.Unmarshal(recorder.Body.Bytes(), &events); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(events) != 1 || events[0].ID != "audit-tenant-a" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestServer_HandleReports_AuditCSV_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	server := mustNewServer(t, stateStore)
	if err := stateStore.SaveAuditEvent(context.Background(), models.AuditEvent{
		ID:        "audit-1",
		TenantID:  store.DefaultTenantID,
		Actor:     "tenant:default",
		Category:  "migration",
		Action:    "rollback",
		Resource:  "migration-1",
		Outcome:   models.AuditOutcomeFailure,
		Message:   "rollback failed",
		RequestID: "req-1",
		CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("SaveAuditEvent() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/audit?format=csv", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleReports(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/csv") {
		t.Fatalf("content type = %q, want CSV", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "id,created_at,actor,category,action,resource,outcome,message,request_id") {
		t.Fatalf("CSV header missing: %s", body)
	}
	if !strings.Contains(body, "audit-1") || !strings.Contains(body, "rollback failed") {
		t.Fatalf("CSV body missing audit event: %s", body)
	}
}

func TestServer_HandleReports_MigrationsCSV_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if err := stateStore.SaveMigration(context.Background(), store.DefaultTenantID, store.MigrationRecord{
		ID:          "migration-1",
		TenantID:    store.DefaultTenantID,
		SpecName:    "wave-one",
		Phase:       string(migratepkg.PhaseComplete),
		StartedAt:   time.Date(2026, time.April, 5, 9, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, time.April, 5, 9, 5, 0, 0, time.UTC),
		CompletedAt: time.Date(2026, time.April, 5, 9, 10, 0, 0, time.UTC),
		RawJSON:     json.RawMessage(`{"id":"migration-1","phase":"complete"}`),
	}); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	server := mustNewServer(t, stateStore)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/migrations?format=csv", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleReports(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "id,spec_name,phase,started_at,updated_at,completed_at") {
		t.Fatalf("CSV header missing: %s", body)
	}
	if !strings.Contains(body, "migration-1,wave-one,complete") {
		t.Fatalf("CSV body missing migration event: %s", body)
	}
}

func TestServer_HandleReports_UnknownReport_ReturnsStructuredError(t *testing.T) {
	t.Parallel()

	server := mustNewServer(t, store.NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/unknown", nil)
	req = req.WithContext(store.ContextWithTenantID(req.Context(), store.DefaultTenantID))
	recorder := httptest.NewRecorder()

	server.handleReports(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var response apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if response.Error.Code != "report_not_found" || response.Error.RequestID == "" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}
