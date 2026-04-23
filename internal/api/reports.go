package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/eblackrps/viaduct/internal/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type awaitAuditContextKey struct{}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	events, err := s.store.ListAuditEvents(r.Context(), store.TenantIDFromContext(r.Context()), 200)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	if name == "" || strings.Contains(name, "/") {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "report name is required", apiErrorOptions{})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	switch name {
	case "summary":
		s.writeSummaryReport(w, r, format)
	case "migrations":
		s.writeMigrationsReport(w, r, format)
	case "audit":
		s.writeAuditReport(w, r, format)
	default:
		writeAPIError(w, r, http.StatusNotFound, "report_not_found", "report not found", apiErrorOptions{
			Details: map[string]any{
				"report_name": name,
			},
		})
	}
}

func (s *Server) writeSummaryReport(w http.ResponseWriter, r *http.Request, format string) {
	ctx, span := telemetry.StartSpan(r.Context(), telemetry.ScopeAPI, "report.summary", withTenantSpanAttributes(r.Context(),
		attribute.String("viaduct.report.name", "summary"),
		attribute.String("viaduct.report.format", format),
	)...)
	defer span.End()
	r = r.WithContext(ctx)

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		telemetry.RecordError(span, err)
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	snapshots, err := s.store.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 100)
	if err != nil {
		telemetry.RecordError(span, err)
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	migrations, err := s.store.ListMigrations(r.Context(), store.TenantIDFromContext(r.Context()), 100)
	if err != nil {
		telemetry.RecordError(span, err)
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	span.SetAttributes(
		attribute.Int("viaduct.inventory.vm_count", len(inventory.VMs)),
		attribute.Int("viaduct.snapshot_count", len(snapshots)),
		attribute.Int("viaduct.migration_count", len(migrations)),
	)

	summary := tenantSummary{
		TenantID:      store.TenantIDFromContext(r.Context()),
		WorkloadCount: len(inventory.VMs),
		SnapshotCount: len(snapshots),
		PlatformCounts: func() map[string]int {
			counts := make(map[string]int)
			for _, vm := range inventory.VMs {
				counts[string(vm.Platform)]++
			}
			return counts
		}(),
	}
	for _, migration := range migrations {
		switch migration.Phase {
		case string(migratepkg.PhaseComplete):
			summary.CompletedMigrations++
		case string(migratepkg.PhaseFailed):
			summary.FailedMigrations++
		default:
			summary.ActiveMigrations++
		}
	}
	if len(snapshots) > 0 {
		summary.LastDiscoveryAt = snapshots[0].DiscoveredAt
	}

	if format == "csv" {
		headers := []string{"tenant_id", "workload_count", "snapshot_count", "active_migrations", "completed_migrations", "failed_migrations", "last_discovery_at"}
		row := []string{
			summary.TenantID,
			fmt.Sprintf("%d", summary.WorkloadCount),
			fmt.Sprintf("%d", summary.SnapshotCount),
			fmt.Sprintf("%d", summary.ActiveMigrations),
			fmt.Sprintf("%d", summary.CompletedMigrations),
			fmt.Sprintf("%d", summary.FailedMigrations),
			summary.LastDiscoveryAt.Format(time.RFC3339),
		}
		writeCSV(w, "tenant-summary.csv", headers, [][]string{row})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) writeMigrationsReport(w http.ResponseWriter, r *http.Request, format string) {
	ctx, span := telemetry.StartSpan(r.Context(), telemetry.ScopeAPI, "report.migrations", withTenantSpanAttributes(r.Context(),
		attribute.String("viaduct.report.name", "migrations"),
		attribute.String("viaduct.report.format", format),
	)...)
	defer span.End()
	r = r.WithContext(ctx)

	items, err := s.store.ListMigrations(r.Context(), store.TenantIDFromContext(r.Context()), 200)
	if err != nil {
		telemetry.RecordError(span, err)
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	span.SetAttributes(attribute.Int("viaduct.migration_count", len(items)))

	if format == "csv" {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				item.ID,
				item.SpecName,
				item.Phase,
				item.StartedAt.Format(time.RFC3339),
				item.UpdatedAt.Format(time.RFC3339),
				item.CompletedAt.Format(time.RFC3339),
			})
		}
		writeCSV(w, "migrations.csv", []string{"id", "spec_name", "phase", "started_at", "updated_at", "completed_at"}, rows)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) writeAuditReport(w http.ResponseWriter, r *http.Request, format string) {
	ctx, span := telemetry.StartSpan(r.Context(), telemetry.ScopeAPI, "report.audit", withTenantSpanAttributes(r.Context(),
		attribute.String("viaduct.report.name", "audit"),
		attribute.String("viaduct.report.format", format),
	)...)
	defer span.End()
	r = r.WithContext(ctx)

	items, err := s.store.ListAuditEvents(r.Context(), store.TenantIDFromContext(r.Context()), 200)
	if err != nil {
		telemetry.RecordError(span, err)
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	span.SetAttributes(attribute.Int("viaduct.audit.event_count", len(items)))

	if format == "csv" {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				item.ID,
				item.CreatedAt.Format(time.RFC3339),
				item.Actor,
				item.Category,
				item.Action,
				item.Resource,
				string(item.Outcome),
				item.Message,
				item.RequestID,
			})
		}
		writeCSV(w, "audit.csv", []string{"id", "created_at", "actor", "category", "action", "resource", "outcome", "message", "request_id"}, rows)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// Audit writes default to best-effort asynchronous persistence so routine
// request latency does not depend on the backing store. Callers handling
// regulatory events can opt into durable writes by setting awaitAudit on the
// request context before recording the event.
func (s *Server) recordAuditEvent(r *http.Request, event models.AuditEvent) {
	if s == nil || s.store == nil {
		return
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	event = s.normalizeAuditEvent(ctx, event)

	saveEvent := func(ctx context.Context) {
		ctx, span := telemetry.StartSpan(ctx, telemetry.ScopeAPI, "audit.persist", withTenantSpanAttributes(ctx,
			attribute.String("viaduct.audit.category", strings.TrimSpace(event.Category)),
			attribute.String("viaduct.audit.action", strings.TrimSpace(event.Action)),
		)...)
		defer span.End()
		if err := s.persistAuditEvent(ctx, event); err != nil {
			telemetry.RecordError(span, err)
			log.Printf("component=api category=audit action=save outcome=failure message=%q", err.Error())
		}
	}
	if awaitAuditRequested(ctx) {
		saveEvent(ctx)
		return
	}

	go func(parentCtx context.Context, tenantID, requestID string) {
		ctx, cancel := s.backgroundTaskContext(context.WithoutCancel(parentCtx), tenantID, requestID)
		defer cancel()
		saveEvent(ctx)
	}(ctx, event.TenantID, event.RequestID)
}

func (s *Server) normalizeAuditEvent(ctx context.Context, event models.AuditEvent) models.AuditEvent {
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.TenantID == "" {
		event.TenantID = store.TenantIDFromContext(ctx)
	}
	if event.RequestID == "" {
		event.RequestID = RequestIDFromContext(ctx)
	}
	if event.Actor == "" {
		event.Actor = actorFromContext(ctx)
	}
	return event
}

func (s *Server) persistAuditEvent(ctx context.Context, event models.AuditEvent) error {
	if s == nil || s.store == nil {
		return nil
	}

	firstErr := s.store.SaveAuditEvent(ctx, event)
	if firstErr == nil {
		return nil
	}

	retryErr := s.store.SaveAuditEvent(ctx, event)
	if retryErr == nil {
		return nil
	}
	if queueErr := s.queueAuditRetryEvent(event); queueErr != nil {
		return fmt.Errorf("save audit event: %w; retry: %v; queue: %w", firstErr, retryErr, queueErr)
	}
	s.backgroundLogger().Warn(
		"audit event queued for retry after persistence failures",
		"event_id", event.ID,
		"tenant_id", event.TenantID,
		"request_id", event.RequestID,
	)
	return nil
}

func actorFromContext(ctx context.Context) string {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return "tenant:" + store.TenantIDFromContext(ctx)
	}
	if principal.ServiceAccount != nil {
		return "service-account:" + principal.ServiceAccount.ID
	}
	return "tenant:" + principal.Tenant.ID
}

func withAwaitAudit(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, awaitAuditContextKey{}, true)
}

func awaitAuditRequested(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	await, _ := ctx.Value(awaitAuditContextKey{}).(bool)
	return await
}

func writeCSV(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(headers); err != nil {
		http.Error(w, "failed to encode CSV", http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			http.Error(w, "failed to encode CSV", http.StatusInternalServerError)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, "failed to flush CSV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		packageLogger.Warn("failed to write CSV response", "file_name", filename, "error", err.Error())
	}
}
