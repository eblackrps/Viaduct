package store

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var storeTracerProviderMu sync.Mutex

func TestPostgresStore_ListSnapshotsPage_EmitsStoreSpan_Expected(t *testing.T) {
	recorder := installStoreSpanRecorder(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	stateStore := &PostgresStore{db: db}
	discoveredAt := time.Date(2026, time.April, 23, 15, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM snapshots WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)`)).
		WithArgs(DefaultTenantID, "vmware").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, tenant_id, source, platform, vm_count, discovered_at
		 FROM snapshots
		 WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)
		 ORDER BY discovered_at DESC
		 LIMIT $3 OFFSET $4`)).
		WithArgs(DefaultTenantID, "vmware", 25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "source", "platform", "vm_count", "discovered_at"}).
			AddRow("snapshot-1", DefaultTenantID, "lab-vcenter", "vmware", 12, discoveredAt))

	items, total, err := stateStore.ListSnapshotsPage(context.Background(), DefaultTenantID, "vmware", 1, 25)
	if err != nil {
		t.Fatalf("ListSnapshotsPage() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("len(spans) = %d, want 1", len(spans))
	}
	span := spans[0]
	if span.Name() != "store.postgres.list_snapshots" {
		t.Fatalf("span.Name() = %q, want store.postgres.list_snapshots", span.Name())
	}
	assertStoreSpanAttr(t, span, "viaduct.store.backend", "postgres")
	assertStoreSpanAttr(t, span, "viaduct.store.operation", "list_snapshots")
	assertStoreSpanAttr(t, span, "tenant.id", DefaultTenantID)
	assertStoreSpanAttr(t, span, "viaduct.snapshot.platform", "vmware")
	assertStoreSpanAttr(t, span, "viaduct.store.page", int64(1))
	assertStoreSpanAttr(t, span, "viaduct.store.per_page", int64(25))
	assertStoreSpanAttr(t, span, "viaduct.store.result_count", int64(1))
	assertStoreSpanAttr(t, span, "viaduct.store.total", int64(1))
}

func TestPostgresStore_SaveMigration_EmitsStoreSpan_Expected(t *testing.T) {
	recorder := installStoreSpanRecorder(t)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	stateStore := &PostgresStore{db: db}
	now := time.Date(2026, time.April, 23, 16, 0, 0, 0, time.UTC)
	record := MigrationRecord{
		ID:        "migration-42",
		SpecName:  "pilot-plan",
		Phase:     "planned",
		StartedAt: now,
		UpdatedAt: now,
		RawJSON:   []byte(`{"id":"migration-42","phase":"planned"}`),
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants WHERE id = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key", "api_key_hash", "created_at", "active", "settings", "quotas", "service_accounts"}).
			AddRow("tenant-a", "Tenant A", "", "", now, true, []byte(`{}`), []byte(`{}`), []byte(`[]`)))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO migrations (id, tenant_id, spec_name, phase, started_at, updated_at, completed_at, raw_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (tenant_id, id) DO UPDATE SET
		   spec_name = EXCLUDED.spec_name,
		   phase = EXCLUDED.phase,
		   started_at = EXCLUDED.started_at,
		   updated_at = EXCLUDED.updated_at,
		   completed_at = EXCLUDED.completed_at,
		   raw_json = EXCLUDED.raw_json`)).
		WithArgs(record.ID, "tenant-a", record.SpecName, record.Phase, record.StartedAt, record.UpdatedAt, nil, record.RawJSON).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := stateStore.SaveMigration(context.Background(), "tenant-a", record); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("len(spans) = %d, want 1", len(spans))
	}
	span := spans[0]
	if span.Name() != "store.postgres.save_migration" {
		t.Fatalf("span.Name() = %q, want store.postgres.save_migration", span.Name())
	}
	assertStoreSpanAttr(t, span, "viaduct.store.backend", "postgres")
	assertStoreSpanAttr(t, span, "viaduct.store.operation", "save_migration")
	assertStoreSpanAttr(t, span, "tenant.id", "tenant-a")
	assertStoreSpanAttr(t, span, "viaduct.migration.id", "migration-42")
	assertStoreSpanAttr(t, span, "viaduct.migration.spec_name", "pilot-plan")
	assertStoreSpanAttr(t, span, "viaduct.migration.phase", "planned")
}

func installStoreSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	storeTracerProviderMu.Lock()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(previous)
		storeTracerProviderMu.Unlock()
	})
	return recorder
}

func assertStoreSpanAttr(t *testing.T, span sdktrace.ReadOnlySpan, key string, want any) {
	t.Helper()

	for _, attr := range span.Attributes() {
		if string(attr.Key) == key {
			if got := attr.Value.AsInterface(); got != want {
				t.Fatalf("span attr %s = %#v, want %#v", key, got, want)
			}
			return
		}
	}
	t.Fatalf("span attr %s was not recorded", key)
}
