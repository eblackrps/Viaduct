package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
)

const currentStoreSchemaVersion = 8

type storeSchemaVersion struct {
	version     int
	description string
}

var storeSchemaHistory = []storeSchemaVersion{
	{version: 1, description: "initial tenant, snapshot, migration, and recovery-point schema"},
	{version: 2, description: "tenant-scoped migration and recovery-point identifiers"},
	{version: 3, description: "tenant-scoped audit event history"},
	{version: 4, description: "tenant quotas and service-account metadata"},
	{version: 5, description: "schema version tracking and operator diagnostics"},
	{version: 6, description: "pilot workspaces and persisted workspace background jobs"},
	{version: 7, description: "hashed tenant credentials and durable credential uniqueness registry"},
	{version: 8, description: "dashboard session revocation registry"},
}

const createStoreSchemaSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	description TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	api_key TEXT NOT NULL,
	api_key_hash TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL,
	active BOOLEAN NOT NULL,
	settings JSONB NOT NULL DEFAULT '{}'::jsonb,
	quotas JSONB NOT NULL DEFAULT '{}'::jsonb,
	service_accounts JSONB NOT NULL DEFAULT '[]'::jsonb
);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS api_key_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS quotas JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS service_accounts JSONB NOT NULL DEFAULT '[]'::jsonb;

INSERT INTO tenants (id, name, api_key, created_at, active, settings)
VALUES ('default', 'Default Tenant', '', to_timestamp(0), TRUE, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS credential_hashes (
	credential_hash TEXT PRIMARY KEY,
	tenant_id TEXT NOT NULL,
	owner_type TEXT NOT NULL,
	service_account_id TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credential_hashes_tenant ON credential_hashes(tenant_id);

CREATE TABLE IF NOT EXISTS snapshots (
	id TEXT PRIMARY KEY,
	tenant_id TEXT NOT NULL DEFAULT 'default',
	source TEXT NOT NULL,
	platform TEXT NOT NULL,
	vm_count INTEGER NOT NULL,
	discovered_at TIMESTAMPTZ NOT NULL,
	duration_ms BIGINT NOT NULL,
	raw_json JSONB NOT NULL
);
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
CREATE INDEX IF NOT EXISTS idx_snapshots_tenant_platform_discovered_at ON snapshots(tenant_id, platform, discovered_at DESC);

CREATE TABLE IF NOT EXISTS migrations (
	tenant_id TEXT NOT NULL DEFAULT 'default',
	id TEXT NOT NULL,
	spec_name TEXT NOT NULL,
	phase TEXT NOT NULL,
	started_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	completed_at TIMESTAMPTZ,
	raw_json JSONB NOT NULL,
	PRIMARY KEY (tenant_id, id)
);
ALTER TABLE migrations ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'migrations_pkey'
		  AND conrelid = 'migrations'::regclass
	) THEN
		ALTER TABLE migrations DROP CONSTRAINT migrations_pkey;
	END IF;
EXCEPTION
	WHEN undefined_table THEN NULL;
END $$;
ALTER TABLE migrations ADD CONSTRAINT migrations_pkey PRIMARY KEY (tenant_id, id);
CREATE INDEX IF NOT EXISTS idx_migrations_tenant_updated_at ON migrations(tenant_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS recovery_points (
	tenant_id TEXT NOT NULL DEFAULT 'default',
	migration_id TEXT NOT NULL,
	phase TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	raw_json JSONB NOT NULL,
	PRIMARY KEY (tenant_id, migration_id)
);
ALTER TABLE recovery_points ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';

CREATE TABLE IF NOT EXISTS audit_events (
	id TEXT PRIMARY KEY,
	tenant_id TEXT NOT NULL DEFAULT 'default',
	actor TEXT NOT NULL,
	request_id TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL,
	action TEXT NOT NULL,
	resource TEXT NOT NULL DEFAULT '',
	outcome TEXT NOT NULL,
	message TEXT NOT NULL,
	details JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL
);
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS actor TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS resource TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT 'success';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS message TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT to_timestamp(0);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created_at ON audit_events(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS revoked_sessions (
	session_id TEXT PRIMARY KEY,
	expires_at TIMESTAMPTZ NOT NULL,
	revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_revoked_sessions_expires_at ON revoked_sessions(expires_at);

CREATE TABLE IF NOT EXISTS workspaces (
	tenant_id TEXT NOT NULL DEFAULT 'default',
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	raw_json JSONB NOT NULL,
	PRIMARY KEY (tenant_id, id)
);
CREATE INDEX IF NOT EXISTS idx_workspaces_tenant_updated_at ON workspaces(tenant_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS workspace_jobs (
	tenant_id TEXT NOT NULL DEFAULT 'default',
	workspace_id TEXT NOT NULL,
	id TEXT NOT NULL,
	type TEXT NOT NULL,
	status TEXT NOT NULL,
	requested_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	completed_at TIMESTAMPTZ,
	correlation_id TEXT NOT NULL DEFAULT '',
	raw_json JSONB NOT NULL,
	PRIMARY KEY (tenant_id, workspace_id, id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_jobs_tenant_workspace_updated_at ON workspace_jobs(tenant_id, workspace_id, updated_at DESC);
`

// PostgresStore is a PostgreSQL-backed Store implementation.
type PostgresStore struct {
	db           *sql.DB
	readTimeout  time.Duration
	writeTimeout time.Duration
	maxOpenConns int
}

// NewPostgresStore connects to PostgreSQL and ensures the required schema exists.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	settings := loadPostgresSettings()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres store: open database: %w", err)
	}
	db.SetMaxOpenConns(settings.MaxOpenConns)
	db.SetMaxIdleConns(settings.MaxIdleConns)
	db.SetConnMaxLifetime(settings.ConnMaxLifetime)

	pingCtx, cancel := withTimeout(ctx, settings.WriteTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		// Close is best effort while unwinding a failed store initialization.
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: ping database: %w", err)
	}

	migrationCtx, migrationCancel := withTimeout(ctx, settings.WriteTimeout)
	defer migrationCancel()
	if _, err := db.ExecContext(migrationCtx, createStoreSchemaSQL); err != nil {
		// Close is best effort while unwinding a failed store initialization.
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: run migrations: %w", err)
	}
	recordCtx, recordCancel := withTimeout(ctx, settings.WriteTimeout)
	defer recordCancel()
	if err := recordStoreSchemaHistory(recordCtx, db); err != nil {
		// Close is best effort while unwinding a failed store initialization.
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: record schema history: %w", err)
	}
	if err := migrateStoredCredentials(recordCtx, db); err != nil {
		// Close is best effort while unwinding a failed credential migration.
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: migrate credentials: %w", annotateCredentialMigrationError(err))
	}
	if err := applyCredentialHashUniqueIndexMigration(recordCtx, db); err != nil {
		// Close is best effort while unwinding a failed credential-index migration.
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: credential hash unique index: %w", annotateCredentialMigrationError(err))
	}

	return &PostgresStore{
		db:           db,
		readTimeout:  settings.ReadTimeout,
		writeTimeout: settings.WriteTimeout,
		maxOpenConns: settings.MaxOpenConns,
	}, nil
}

type postgresSettings struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func loadPostgresSettings() postgresSettings {
	return postgresSettings{
		ReadTimeout:     durationFromEnv("VIADUCT_DB_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    durationFromEnv("VIADUCT_DB_WRITE_TIMEOUT", 10*time.Second),
		MaxOpenConns:    intFromEnv("VIADUCT_DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    intFromEnv("VIADUCT_DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: durationFromEnv("VIADUCT_DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

func (s *PostgresStore) readContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s == nil {
		return withTimeout(ctx, 5*time.Second)
	}
	return withTimeout(ctx, s.readTimeout)
}

func (s *PostgresStore) writeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if s == nil {
		return withTimeout(ctx, 10*time.Second)
	}
	return withTimeout(ctx, s.writeTimeout)
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= timeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intFromEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// SaveDiscovery persists a discovery result to PostgreSQL.
func (s *PostgresStore) SaveDiscovery(ctx context.Context, tenantID string, result *models.DiscoveryResult) (snapshotID string, err error) {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if result == nil {
		return "", fmt.Errorf("postgres store: save discovery: result is nil")
	}

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "save_discovery", tenantID,
		attribute.String("viaduct.snapshot.source", strings.TrimSpace(result.Source)),
		attribute.String("viaduct.snapshot.platform", string(result.Platform)),
		attribute.Int("viaduct.snapshot.vm_count", len(result.VMs)),
	)
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 1)
		attrs = appendStoreStringAttr(attrs, "viaduct.snapshot.id", snapshotID)
		finishStoreSpan(span, err, attrs...)
	}()

	tenant, err := s.ensureTenant(ctx, tenantID)
	if err != nil {
		err = fmt.Errorf("postgres store: save discovery: %w", err)
		return "", err
	}
	if tenant.Quotas.MaxSnapshots > 0 {
		currentCount, err := s.snapshotCount(ctx, tenantID)
		if err != nil {
			err = fmt.Errorf("postgres store: save discovery: %w", err)
			return "", err
		}
		if currentCount >= tenant.Quotas.MaxSnapshots {
			err = fmt.Errorf("postgres store: save discovery: snapshot quota exceeded for tenant %s", tenantID)
			return "", err
		}
	}

	payload, err := json.Marshal(result)
	if err != nil {
		err = fmt.Errorf("postgres store: marshal discovery: %w", err)
		return "", err
	}

	snapshotID = uuid.NewString()
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO snapshots (id, tenant_id, source, platform, vm_count, discovered_at, duration_ms, raw_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		snapshotID,
		tenantID,
		result.Source,
		string(result.Platform),
		len(result.VMs),
		result.DiscoveredAt,
		result.Duration.Milliseconds(),
		payload,
	)
	if err != nil {
		err = fmt.Errorf("postgres store: insert snapshot: %w", err)
		return "", err
	}

	return snapshotID, nil
}

// GetSnapshot retrieves a discovery snapshot by identifier from PostgreSQL.
func (s *PostgresStore) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (snapshot *models.DiscoveryResult, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "get_snapshot", tenantID,
		attribute.String("viaduct.snapshot.id", strings.TrimSpace(snapshotID)),
	)
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 3)
		if snapshot != nil {
			attrs = appendStoreStringAttr(attrs, "viaduct.snapshot.source", snapshot.Source)
			attrs = appendStoreStringAttr(attrs, "viaduct.snapshot.platform", string(snapshot.Platform))
			attrs = append(attrs, attribute.Int("viaduct.snapshot.vm_count", len(snapshot.VMs)))
		}
		finishStoreSpan(span, err, attrs...)
	}()

	var payload []byte
	if err = s.db.QueryRowContext(
		ctx,
		`SELECT raw_json FROM snapshots WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		snapshotID,
	).Scan(&payload); err != nil {
		err = fmt.Errorf("postgres store: get snapshot %s: %w", snapshotID, err)
		return nil, err
	}

	var result models.DiscoveryResult
	if err = json.Unmarshal(payload, &result); err != nil {
		err = fmt.Errorf("postgres store: decode snapshot %s: %w", snapshotID, err)
		return nil, err
	}

	snapshot = &result
	return snapshot, nil
}

// ListSnapshots returns snapshot metadata ordered from newest to oldest.
func (s *PostgresStore) ListSnapshots(ctx context.Context, tenantID string, platform models.Platform, limit int) ([]SnapshotMeta, error) {
	items, _, err := s.ListSnapshotsPage(ctx, tenantID, platform, 1, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ListSnapshotsPage returns paginated snapshot metadata ordered from newest to oldest.
func (s *PostgresStore) ListSnapshotsPage(ctx context.Context, tenantID string, platform models.Platform, page, perPage int) (items []SnapshotMeta, total int, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	page, perPage, err = normalizePageRequest(page, perPage)
	if err != nil {
		err = fmt.Errorf("postgres store: list snapshots: %w", err)
		return nil, 0, err
	}

	tenantID = normalizeTenantID(tenantID)
	platformName := string(platform)
	ctx, span := startPostgresStoreSpan(ctx, "list_snapshots", tenantID,
		attribute.String("viaduct.snapshot.platform", platformName),
		attribute.Int("viaduct.store.page", page),
		attribute.Int("viaduct.store.per_page", perPage),
	)
	defer func() {
		finishStoreSpan(span, err,
			attribute.Int("viaduct.store.result_count", len(items)),
			attribute.Int("viaduct.store.total", total),
		)
	}()

	if err = s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM snapshots WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)`,
		tenantID,
		platformName,
	).Scan(&total); err != nil {
		err = fmt.Errorf("postgres store: count snapshots: %w", err)
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, tenant_id, source, platform, vm_count, discovered_at
		 FROM snapshots
		 WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)
		 ORDER BY discovered_at DESC
		 LIMIT $3 OFFSET $4`,
		tenantID,
		platformName,
		perPage,
		pageOffset(page, perPage),
	)
	if err != nil {
		err = fmt.Errorf("postgres store: list snapshots: %w", err)
		return nil, 0, err
	}
	defer rows.Close()

	items = make([]SnapshotMeta, 0, pageCapacity(total, perPage))
	for rows.Next() {
		var item SnapshotMeta
		var platformName string
		if err = rows.Scan(&item.ID, &item.TenantID, &item.Source, &platformName, &item.VMCount, &item.DiscoveredAt); err != nil {
			err = fmt.Errorf("postgres store: scan snapshot metadata: %w", err)
			return nil, 0, err
		}
		item.Platform = models.Platform(platformName)
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("postgres store: iterate snapshot metadata: %w", err)
		return nil, 0, err
	}

	return items, total, nil
}

// QueryVMs returns stored VMs that match the supplied filter criteria.
func (s *PostgresStore) QueryVMs(ctx context.Context, tenantID string, filter VMFilter) ([]models.VirtualMachine, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	rows, err := s.db.QueryContext(ctx, `SELECT raw_json FROM snapshots WHERE tenant_id = $1 ORDER BY discovered_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("postgres store: query snapshots for VMs: %w", err)
	}
	defer rows.Close()

	results := make([]models.VirtualMachine, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("postgres store: scan VM payload: %w", err)
		}

		var snapshot models.DiscoveryResult
		if err := json.Unmarshal(payload, &snapshot); err != nil {
			return nil, fmt.Errorf("postgres store: decode VM payload: %w", err)
		}

		for _, vm := range snapshot.VMs {
			if !matchesFilter(vm, filter) {
				continue
			}

			results = append(results, vm)
			if filter.Limit > 0 && len(results) >= filter.Limit {
				return results, nil
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate VM payloads: %w", err)
	}

	return results, nil
}

// SaveMigration persists a serialized migration record to PostgreSQL.
func (s *PostgresStore) SaveMigration(ctx context.Context, tenantID string, record MigrationRecord) (err error) {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if record.ID == "" {
		return fmt.Errorf("postgres store: save migration: migration ID is empty")
	}
	if len(record.RawJSON) == 0 {
		return fmt.Errorf("postgres store: save migration: raw JSON is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID
	ctx, span := startPostgresStoreSpan(ctx, "save_migration", tenantID,
		attribute.String("viaduct.migration.id", strings.TrimSpace(record.ID)),
		attribute.String("viaduct.migration.spec_name", strings.TrimSpace(record.SpecName)),
		attribute.String("viaduct.migration.phase", strings.TrimSpace(record.Phase)),
	)
	defer func() {
		finishStoreSpan(span, err)
	}()

	tenant, err := s.ensureTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("postgres store: save migration: %w", err)
	}
	if tenant.Quotas.MaxMigrations > 0 {
		exists, err := s.migrationExists(ctx, tenantID, record.ID)
		if err != nil {
			return fmt.Errorf("postgres store: save migration: %w", err)
		}
		if !exists {
			currentCount, err := s.migrationCount(ctx, tenantID)
			if err != nil {
				return fmt.Errorf("postgres store: save migration: %w", err)
			}
			if currentCount >= tenant.Quotas.MaxMigrations {
				return fmt.Errorf("postgres store: save migration: migration quota exceeded for tenant %s", tenantID)
			}
		}
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO migrations (id, tenant_id, spec_name, phase, started_at, updated_at, completed_at, raw_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (tenant_id, id) DO UPDATE SET
		   spec_name = EXCLUDED.spec_name,
		   phase = EXCLUDED.phase,
		   started_at = EXCLUDED.started_at,
		   updated_at = EXCLUDED.updated_at,
		   completed_at = EXCLUDED.completed_at,
		   raw_json = EXCLUDED.raw_json`,
		record.ID,
		tenantID,
		record.SpecName,
		record.Phase,
		record.StartedAt,
		record.UpdatedAt,
		nullTime(record.CompletedAt),
		record.RawJSON,
	)
	if err != nil {
		return fmt.Errorf("postgres store: save migration %s: %w", record.ID, err)
	}

	return nil
}

// GetMigration retrieves a serialized migration record from PostgreSQL.
func (s *PostgresStore) GetMigration(ctx context.Context, tenantID, migrationID string) (record *MigrationRecord, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "get_migration", tenantID,
		attribute.String("viaduct.migration.id", strings.TrimSpace(migrationID)),
	)
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 2)
		if record != nil {
			attrs = appendStoreStringAttr(attrs, "viaduct.migration.spec_name", record.SpecName)
			attrs = appendStoreStringAttr(attrs, "viaduct.migration.phase", record.Phase)
		}
		finishStoreSpan(span, err, attrs...)
	}()

	var item MigrationRecord
	var completedAt sql.NullTime
	if err = s.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, spec_name, phase, started_at, updated_at, completed_at, raw_json
		 FROM migrations WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		migrationID,
	).Scan(&item.ID, &item.TenantID, &item.SpecName, &item.Phase, &item.StartedAt, &item.UpdatedAt, &completedAt, &item.RawJSON); err != nil {
		return nil, fmt.Errorf("postgres store: get migration %s: %w", migrationID, err)
	}

	if completedAt.Valid {
		item.CompletedAt = completedAt.Time
	}

	record = &item
	return record, nil
}

// ListMigrations returns migration metadata ordered by most recent update.
func (s *PostgresStore) ListMigrations(ctx context.Context, tenantID string, limit int) ([]MigrationMeta, error) {
	items, _, err := s.ListMigrationsPage(ctx, tenantID, 1, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ListMigrationsPage returns paginated migration metadata ordered by most recent update.
func (s *PostgresStore) ListMigrationsPage(ctx context.Context, tenantID string, page, perPage int) (items []MigrationMeta, total int, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	page, perPage, err = normalizePageRequest(page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres store: list migrations: %w", err)
	}

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "list_migrations", tenantID,
		attribute.Int("viaduct.store.page", page),
		attribute.Int("viaduct.store.per_page", perPage),
	)
	defer func() {
		finishStoreSpan(span, err,
			attribute.Int("viaduct.store.result_count", len(items)),
			attribute.Int("viaduct.store.total", total),
		)
	}()

	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migrations WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres store: count migrations: %w", err)
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, tenant_id, spec_name, phase, started_at, updated_at, completed_at
		 FROM migrations
		 WHERE tenant_id = $1
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		tenantID,
		perPage,
		pageOffset(page, perPage),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres store: list migrations: %w", err)
	}
	defer rows.Close()

	items = make([]MigrationMeta, 0, pageCapacity(total, perPage))
	for rows.Next() {
		var item MigrationMeta
		var completedAt sql.NullTime
		if err = rows.Scan(&item.ID, &item.TenantID, &item.SpecName, &item.Phase, &item.StartedAt, &item.UpdatedAt, &completedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres store: scan migration metadata: %w", err)
		}
		if completedAt.Valid {
			item.CompletedAt = completedAt.Time
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres store: iterate migration metadata: %w", err)
	}

	return items, total, nil
}

// SaveRecoveryPoint persists a serialized recovery point to PostgreSQL.
func (s *PostgresStore) SaveRecoveryPoint(ctx context.Context, tenantID string, record RecoveryPointRecord) (err error) {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if record.MigrationID == "" {
		return fmt.Errorf("postgres store: save recovery point: migration ID is empty")
	}
	if len(record.RawJSON) == 0 {
		return fmt.Errorf("postgres store: save recovery point: raw JSON is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID
	ctx, span := startPostgresStoreSpan(ctx, "save_recovery_point", tenantID,
		attribute.String("viaduct.migration.id", strings.TrimSpace(record.MigrationID)),
		attribute.String("viaduct.recovery.phase", strings.TrimSpace(record.Phase)),
	)
	defer func() {
		finishStoreSpan(span, err)
	}()

	if _, err := s.ensureTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("postgres store: save recovery point: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO recovery_points (tenant_id, migration_id, phase, created_at, raw_json)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, migration_id) DO UPDATE SET
		   phase = EXCLUDED.phase,
		   created_at = EXCLUDED.created_at,
		   raw_json = EXCLUDED.raw_json`,
		tenantID,
		record.MigrationID,
		record.Phase,
		record.CreatedAt,
		record.RawJSON,
	)
	if err != nil {
		return fmt.Errorf("postgres store: save recovery point %s: %w", record.MigrationID, err)
	}

	return nil
}

// GetRecoveryPoint retrieves a serialized recovery point from PostgreSQL.
func (s *PostgresStore) GetRecoveryPoint(ctx context.Context, tenantID, migrationID string) (record *RecoveryPointRecord, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "get_recovery_point", tenantID,
		attribute.String("viaduct.migration.id", strings.TrimSpace(migrationID)),
	)
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 1)
		if record != nil {
			attrs = appendStoreStringAttr(attrs, "viaduct.recovery.phase", record.Phase)
		}
		finishStoreSpan(span, err, attrs...)
	}()

	var item RecoveryPointRecord
	if err = s.db.QueryRowContext(
		ctx,
		`SELECT tenant_id, migration_id, phase, created_at, raw_json
		 FROM recovery_points WHERE tenant_id = $1 AND migration_id = $2`,
		tenantID,
		migrationID,
	).Scan(&item.TenantID, &item.MigrationID, &item.Phase, &item.CreatedAt, &item.RawJSON); err != nil {
		return nil, fmt.Errorf("postgres store: get recovery point %s: %w", migrationID, err)
	}

	record = &item
	return record, nil
}

// CreateTenant persists tenant metadata in PostgreSQL.
func (s *PostgresStore) CreateTenant(ctx context.Context, tenant models.Tenant) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	tenant.ID = normalizeTenantID(tenant.ID)
	if tenant.ID == DefaultTenantID {
		tenant = defaultTenant()
	}
	if tenant.Name == "" {
		return fmt.Errorf("postgres store: create tenant: tenant name is empty")
	}
	tenant = normalizeTenant(tenant)
	tenant, err := prepareTenantCredentials(tenant)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant %s: %w", tenant.ID, err)
	}
	existingTenants, err := s.ListTenants(ctx)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant %s: validate credentials: %w", tenant.ID, err)
	}
	if err := validateCredentialUniqueness(existingTenants, tenant); err != nil {
		return fmt.Errorf("postgres store: create tenant %s: %w", tenant.ID, err)
	}
	settingsPayload, err := marshalTenantComponent("settings", tenant.Settings)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant: %w", err)
	}
	quotasPayload, err := marshalTenantComponent("quotas", tenant.Quotas)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant: %w", err)
	}
	serviceAccountsPayload, err := marshalServiceAccounts(tenant.ServiceAccounts)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant %s: begin transaction: %w", tenant.ID, err)
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO tenants (id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tenant.ID,
		tenant.Name,
		"",
		tenant.APIKeyHash,
		tenant.CreatedAt,
		tenant.Active,
		settingsPayload,
		quotasPayload,
		serviceAccountsPayload,
	); err != nil {
		return fmt.Errorf("postgres store: create tenant %s: %w", tenant.ID, translateCredentialConstraintError(err))
	}
	if err := replaceTenantCredentialHashes(ctx, tx, tenant); err != nil {
		return fmt.Errorf("postgres store: create tenant %s: %w", tenant.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: create tenant %s: commit transaction: %w", tenant.ID, err)
	}
	return nil
}

// UpdateTenant overwrites tenant metadata in PostgreSQL.
func (s *PostgresStore) UpdateTenant(ctx context.Context, tenant models.Tenant) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	tenant.ID = normalizeTenantID(tenant.ID)
	if tenant.Name == "" {
		return fmt.Errorf("postgres store: update tenant: tenant name is empty")
	}
	if tenant.ID == DefaultTenantID {
		existing := defaultTenant()
		existing.APIKey = tenant.APIKey
		existing.APIKeyHash = tenant.APIKeyHash
		existing.Active = tenant.Active
		existing.Settings = tenant.Settings
		existing.Quotas = tenant.Quotas
		existing.ServiceAccounts = tenant.ServiceAccounts
		tenant = existing
	}
	tenant = normalizeTenant(tenant)
	tenant, err := prepareTenantCredentials(tenant)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}
	existingTenants, err := s.ListTenants(ctx)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: validate credentials: %w", tenant.ID, err)
	}
	if err := validateCredentialUniqueness(existingTenants, tenant); err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}
	settingsPayload, err := marshalTenantComponent("settings", tenant.Settings)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}
	quotasPayload, err := marshalTenantComponent("quotas", tenant.Quotas)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}
	serviceAccountsPayload, err := marshalServiceAccounts(tenant.ServiceAccounts)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: begin transaction: %w", tenant.ID, err)
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(
		ctx,
		`UPDATE tenants
		 SET name = $2,
		     api_key = $3,
		     api_key_hash = $4,
		     created_at = $5,
		     active = $6,
		     settings = $7,
		     quotas = $8,
		     service_accounts = $9
		 WHERE id = $1`,
		tenant.ID,
		tenant.Name,
		"",
		tenant.APIKeyHash,
		tenant.CreatedAt,
		tenant.Active,
		settingsPayload,
		quotasPayload,
		serviceAccountsPayload,
	)
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, translateCredentialConstraintError(err))
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres store: update tenant %s: rows affected: %w", tenant.ID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("postgres store: update tenant %s: not found", tenant.ID)
	}
	if err := replaceTenantCredentialHashes(ctx, tx, tenant); err != nil {
		return fmt.Errorf("postgres store: update tenant %s: %w", tenant.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: update tenant %s: commit transaction: %w", tenant.ID, err)
	}

	return nil
}

// GetTenant retrieves a tenant from PostgreSQL by identifier.
func (s *PostgresStore) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	var (
		tenant                 models.Tenant
		settingsPayload        []byte
		quotasPayload          []byte
		serviceAccountsPayload []byte
	)
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants WHERE id = $1`,
		tenantID,
	).Scan(&tenant.ID, &tenant.Name, &tenant.APIKey, &tenant.APIKeyHash, &tenant.CreatedAt, &tenant.Active, &settingsPayload, &quotasPayload, &serviceAccountsPayload); err != nil {
		return nil, fmt.Errorf("postgres store: get tenant %s: %w", tenantID, err)
	}

	if err := decodeTenantComponents(&tenant, settingsPayload, quotasPayload, serviceAccountsPayload); err != nil {
		return nil, fmt.Errorf("postgres store: get tenant %s: %w", tenantID, err)
	}

	return &tenant, nil
}

// ListTenants returns all configured tenants from PostgreSQL.
func (s *PostgresStore) ListTenants(ctx context.Context) ([]models.Tenant, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list tenants: %w", err)
	}
	defer rows.Close()

	items := make([]models.Tenant, 0)
	for rows.Next() {
		var (
			tenant                 models.Tenant
			settingsPayload        []byte
			quotasPayload          []byte
			serviceAccountsPayload []byte
		)
		if err := rows.Scan(&tenant.ID, &tenant.Name, &tenant.APIKey, &tenant.APIKeyHash, &tenant.CreatedAt, &tenant.Active, &settingsPayload, &quotasPayload, &serviceAccountsPayload); err != nil {
			return nil, fmt.Errorf("postgres store: scan tenant: %w", err)
		}
		if err := decodeTenantComponents(&tenant, settingsPayload, quotasPayload, serviceAccountsPayload); err != nil {
			return nil, fmt.Errorf("postgres store: decode tenant %s: %w", tenant.ID, err)
		}
		items = append(items, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate tenants: %w", err)
	}

	return items, nil
}

// DeleteTenant removes a tenant and all associated tenant-scoped data.
func (s *PostgresStore) DeleteTenant(ctx context.Context, tenantID string) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)
	if tenantID == DefaultTenantID {
		return fmt.Errorf("postgres store: delete tenant: default tenant cannot be removed")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: delete tenant %s: begin transaction: %w", tenantID, err)
	}
	defer func() {
		// Rollback is best effort after commit or terminal transaction failure.
		_ = tx.Rollback()
	}()

	for _, statement := range []string{
		`DELETE FROM audit_events WHERE tenant_id = $1`,
		`DELETE FROM workspace_jobs WHERE tenant_id = $1`,
		`DELETE FROM workspaces WHERE tenant_id = $1`,
		`DELETE FROM recovery_points WHERE tenant_id = $1`,
		`DELETE FROM migrations WHERE tenant_id = $1`,
		`DELETE FROM snapshots WHERE tenant_id = $1`,
		`DELETE FROM credential_hashes WHERE tenant_id = $1`,
		`DELETE FROM tenants WHERE id = $1`,
	} {
		if _, err := tx.ExecContext(ctx, statement, tenantID); err != nil {
			return fmt.Errorf("postgres store: delete tenant %s: %w", tenantID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: delete tenant %s: commit: %w", tenantID, err)
	}

	return nil
}

// Close releases the PostgreSQL database connection pool.
func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

// Diagnostics returns backend metadata for the PostgreSQL store.
func (s *PostgresStore) Diagnostics(ctx context.Context) (Diagnostics, error) {
	if s == nil || s.db == nil {
		return Diagnostics{}, fmt.Errorf("postgres store: diagnostics: database is not configured")
	}

	ctx, cancel := s.readContext(ctx)
	defer cancel()

	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return Diagnostics{}, fmt.Errorf("postgres store: diagnostics: query schema version: %w", err)
	}

	stats := s.db.Stats()

	return Diagnostics{
		Backend:       "postgres",
		SchemaVersion: version,
		Persistent:    true,
		DBPool: &DBPoolDiagnostics{
			MaxOpenConnections: stats.MaxOpenConnections,
			OpenConnections:    stats.OpenConnections,
			InUse:              stats.InUse,
			Idle:               stats.Idle,
			WaitCount:          stats.WaitCount,
			WaitDuration:       stats.WaitDuration,
			MaxIdleClosed:      stats.MaxIdleClosed,
			MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
			MaxLifetimeClosed:  stats.MaxLifetimeClosed,
			ReadTimeout:        s.readTimeout,
			WriteTimeout:       s.writeTimeout,
		},
	}, nil
}

// SaveAuditEvent persists a tenant-scoped audit event to PostgreSQL.
func (s *PostgresStore) SaveAuditEvent(ctx context.Context, event models.AuditEvent) (err error) {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	event.TenantID = normalizeTenantID(event.TenantID)
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Outcome == "" {
		event.Outcome = models.AuditOutcomeSuccess
	}
	if event.Details == nil {
		event.Details = map[string]string{}
	}
	ctx, span := startPostgresStoreSpan(ctx, "save_audit_event", event.TenantID,
		attribute.String("viaduct.audit.id", strings.TrimSpace(event.ID)),
		attribute.String("viaduct.audit.category", strings.TrimSpace(event.Category)),
		attribute.String("viaduct.audit.action", strings.TrimSpace(event.Action)),
		attribute.String("viaduct.audit.outcome", strings.TrimSpace(string(event.Outcome))),
	)
	defer func() {
		attrs := make([]attribute.KeyValue, 0, 1)
		attrs = appendStoreStringAttr(attrs, "viaduct.audit.id", event.ID)
		finishStoreSpan(span, err, attrs...)
	}()

	if _, err := s.ensureTenant(ctx, event.TenantID); err != nil {
		return fmt.Errorf("postgres store: save audit event: %w", err)
	}

	detailsPayload, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("postgres store: save audit event: marshal details: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO audit_events (id, tenant_id, actor, request_id, category, action, resource, outcome, message, details, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.ID,
		event.TenantID,
		event.Actor,
		event.RequestID,
		event.Category,
		event.Action,
		event.Resource,
		string(event.Outcome),
		event.Message,
		detailsPayload,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres store: save audit event %s: %w", event.ID, err)
	}

	return nil
}

// ListAuditEvents returns tenant audit events ordered from newest to oldest.
func (s *PostgresStore) ListAuditEvents(ctx context.Context, tenantID string, limit int) (items []models.AuditEvent, err error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)
	ctx, span := startPostgresStoreSpan(ctx, "list_audit_events", tenantID,
		attribute.Int("viaduct.store.limit", limit),
	)
	defer func() {
		finishStoreSpan(span, err, attribute.Int("viaduct.store.result_count", len(items)))
	}()

	query := `SELECT id, tenant_id, actor, request_id, category, action, resource, outcome, message, details, created_at
		FROM audit_events WHERE tenant_id = $1 ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list audit events: %w", err)
	}
	defer rows.Close()

	items = make([]models.AuditEvent, 0)
	for rows.Next() {
		var (
			event          models.AuditEvent
			outcome        string
			detailsPayload []byte
		)
		if err = rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.Actor,
			&event.RequestID,
			&event.Category,
			&event.Action,
			&event.Resource,
			&outcome,
			&event.Message,
			&detailsPayload,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgres store: scan audit event: %w", err)
		}
		event.Outcome = models.AuditOutcome(outcome)
		if len(detailsPayload) > 0 {
			if err := json.Unmarshal(detailsPayload, &event.Details); err != nil {
				return nil, fmt.Errorf("postgres store: decode audit event details: %w", err)
			}
		}
		items = append(items, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate audit events: %w", err)
	}

	return items, nil
}

// RevokeAuthSession persists an auth session revocation to PostgreSQL until the original session expires.
func (s *PostgresStore) RevokeAuthSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("postgres store: revoke auth session: session ID is empty")
	}
	if expiresAt.IsZero() {
		return fmt.Errorf("postgres store: revoke auth session: expires at is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: revoke auth session %s: begin transaction: %w", sessionID, err)
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO revoked_sessions (session_id, expires_at, revoked_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (session_id) DO UPDATE SET expires_at = EXCLUDED.expires_at, revoked_at = NOW()`,
		sessionID,
		expiresAt.UTC(),
	); err != nil {
		return fmt.Errorf("postgres store: revoke auth session %s: %w", sessionID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: revoke auth session %s: commit transaction: %w", sessionID, err)
	}

	return nil
}

// IsAuthSessionRevoked reports whether a non-expired auth session revocation exists in PostgreSQL.
func (s *PostgresStore) IsAuthSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}

	var revoked bool
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM revoked_sessions
			WHERE session_id = $1
			  AND expires_at > NOW()
		)`,
		sessionID,
	).Scan(&revoked); err != nil {
		return false, fmt.Errorf("postgres store: read auth session revocation %s: %w", sessionID, err)
	}

	return revoked, nil
}

// CreateWorkspace persists a pilot workspace to PostgreSQL.
func (s *PostgresStore) CreateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if strings.TrimSpace(workspace.ID) == "" {
		return fmt.Errorf("postgres store: create workspace: workspace ID is empty")
	}
	if strings.TrimSpace(workspace.Name) == "" {
		return fmt.Errorf("postgres store: create workspace: workspace name is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	if _, err := s.ensureTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("postgres store: create workspace: %w", err)
	}

	workspace = normalizeWorkspace(tenantID, workspace)
	payload, err := json.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("postgres store: create workspace: marshal workspace: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO workspaces (tenant_id, id, name, status, created_at, updated_at, raw_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID,
		workspace.ID,
		workspace.Name,
		string(workspace.Status),
		workspace.CreatedAt,
		workspace.UpdatedAt,
		payload,
	)
	if err != nil {
		return fmt.Errorf("postgres store: create workspace %s: %w", workspace.ID, err)
	}

	return nil
}

// UpdateWorkspace overwrites a persisted pilot workspace in PostgreSQL.
func (s *PostgresStore) UpdateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if strings.TrimSpace(workspace.ID) == "" {
		return fmt.Errorf("postgres store: update workspace: workspace ID is empty")
	}
	if strings.TrimSpace(workspace.Name) == "" {
		return fmt.Errorf("postgres store: update workspace: workspace name is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	existing, err := s.GetWorkspace(ctx, tenantID, workspace.ID)
	if err != nil {
		return fmt.Errorf("postgres store: update workspace %s: %w", workspace.ID, err)
	}

	workspace = normalizeWorkspace(tenantID, workspace)
	workspace.CreatedAt = existing.CreatedAt
	if workspace.UpdatedAt.IsZero() || !workspace.UpdatedAt.After(existing.UpdatedAt) {
		workspace.UpdatedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("postgres store: update workspace: marshal workspace: %w", err)
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE workspaces
		    SET name = $3,
		        status = $4,
		        created_at = $5,
		        updated_at = $6,
		        raw_json = $7
		  WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		workspace.ID,
		workspace.Name,
		string(workspace.Status),
		workspace.CreatedAt,
		workspace.UpdatedAt,
		payload,
	)
	if err != nil {
		return fmt.Errorf("postgres store: update workspace %s: %w", workspace.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres store: update workspace %s: rows affected: %w", workspace.ID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("postgres store: update workspace %s: not found", workspace.ID)
	}

	return nil
}

// GetWorkspace retrieves a persisted pilot workspace from PostgreSQL by identifier.
func (s *PostgresStore) GetWorkspace(ctx context.Context, tenantID, workspaceID string) (*models.PilotWorkspace, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	var payload []byte
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT raw_json FROM workspaces WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		workspaceID,
	).Scan(&payload); err != nil {
		return nil, fmt.Errorf("postgres store: get workspace %s: %w", workspaceID, err)
	}

	var workspace models.PilotWorkspace
	if err := json.Unmarshal(payload, &workspace); err != nil {
		return nil, fmt.Errorf("postgres store: decode workspace %s: %w", workspaceID, err)
	}
	workspace = normalizeWorkspace(tenantID, workspace)
	return &workspace, nil
}

// ListWorkspaces returns pilot workspaces ordered from newest to oldest updates.
func (s *PostgresStore) ListWorkspaces(ctx context.Context, tenantID string, limit int) ([]models.PilotWorkspace, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	query := `SELECT raw_json FROM workspaces WHERE tenant_id = $1 ORDER BY updated_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list workspaces: %w", err)
	}
	defer rows.Close()

	items := make([]models.PilotWorkspace, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("postgres store: scan workspace: %w", err)
		}

		var workspace models.PilotWorkspace
		if err := json.Unmarshal(payload, &workspace); err != nil {
			return nil, fmt.Errorf("postgres store: decode workspace: %w", err)
		}
		items = append(items, normalizeWorkspace(tenantID, workspace))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate workspaces: %w", err)
	}

	return items, nil
}

// DeleteWorkspace removes a persisted pilot workspace and any background jobs tied to it.
func (s *PostgresStore) DeleteWorkspace(ctx context.Context, tenantID, workspaceID string) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: delete workspace %s: begin transaction: %w", workspaceID, err)
	}
	defer func() {
		// Rollback is best effort after commit or terminal transaction failure.
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_jobs WHERE tenant_id = $1 AND workspace_id = $2`, tenantID, workspaceID); err != nil {
		return fmt.Errorf("postgres store: delete workspace %s jobs: %w", workspaceID, err)
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM workspaces WHERE tenant_id = $1 AND id = $2`, tenantID, workspaceID)
	if err != nil {
		return fmt.Errorf("postgres store: delete workspace %s: %w", workspaceID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres store: delete workspace %s: rows affected: %w", workspaceID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("postgres store: delete workspace %s: not found", workspaceID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres store: delete workspace %s: commit: %w", workspaceID, err)
	}
	return nil
}

// SaveWorkspaceJob persists a pilot workspace background job to PostgreSQL.
func (s *PostgresStore) SaveWorkspaceJob(ctx context.Context, tenantID string, job models.WorkspaceJob) error {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("postgres store: save workspace job: job ID is empty")
	}
	if strings.TrimSpace(job.WorkspaceID) == "" {
		return fmt.Errorf("postgres store: save workspace job: workspace ID is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	if _, err := s.ensureTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("postgres store: save workspace job: %w", err)
	}
	existing, existingErr := s.GetWorkspaceJob(ctx, tenantID, job.WorkspaceID, job.ID)
	if existingErr == nil {
		if job.RequestedAt.IsZero() {
			job.RequestedAt = existing.RequestedAt
		}
	} else if _, err := s.GetWorkspace(ctx, tenantID, job.WorkspaceID); err != nil {
		return fmt.Errorf("postgres store: save workspace job: %w", err)
	}

	job = normalizeWorkspaceJob(tenantID, job)
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("postgres store: save workspace job: marshal job: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO workspace_jobs (tenant_id, workspace_id, id, type, status, requested_at, updated_at, completed_at, correlation_id, raw_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (tenant_id, workspace_id, id) DO UPDATE
		     SET type = EXCLUDED.type,
		         status = EXCLUDED.status,
		         requested_at = EXCLUDED.requested_at,
		         updated_at = EXCLUDED.updated_at,
		         completed_at = EXCLUDED.completed_at,
		         correlation_id = EXCLUDED.correlation_id,
		         raw_json = EXCLUDED.raw_json`,
		tenantID,
		job.WorkspaceID,
		job.ID,
		string(job.Type),
		string(job.Status),
		job.RequestedAt,
		job.UpdatedAt,
		nullTime(job.CompletedAt),
		job.CorrelationID,
		payload,
	)
	if err != nil {
		return fmt.Errorf("postgres store: save workspace job %s: %w", job.ID, err)
	}

	return nil
}

// GetWorkspaceJob retrieves a persisted pilot workspace background job from PostgreSQL by identifier.
func (s *PostgresStore) GetWorkspaceJob(ctx context.Context, tenantID, workspaceID, jobID string) (*models.WorkspaceJob, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	var payload []byte
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT raw_json FROM workspace_jobs WHERE tenant_id = $1 AND workspace_id = $2 AND id = $3`,
		tenantID,
		workspaceID,
		jobID,
	).Scan(&payload); err != nil {
		return nil, fmt.Errorf("postgres store: get workspace job %s: %w", jobID, err)
	}

	var job models.WorkspaceJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return nil, fmt.Errorf("postgres store: decode workspace job %s: %w", jobID, err)
	}
	job = normalizeWorkspaceJob(tenantID, job)
	return &job, nil
}

// ListWorkspaceJobs returns pilot workspace background jobs ordered from newest to oldest updates.
func (s *PostgresStore) ListWorkspaceJobs(ctx context.Context, tenantID, workspaceID string, limit int) ([]models.WorkspaceJob, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	tenantID = normalizeTenantID(tenantID)

	query := `SELECT raw_json FROM workspace_jobs WHERE tenant_id = $1 AND workspace_id = $2 ORDER BY updated_at DESC`
	args := []interface{}{tenantID, workspaceID}
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list workspace jobs: %w", err)
	}
	defer rows.Close()

	items := make([]models.WorkspaceJob, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("postgres store: scan workspace job: %w", err)
		}

		var job models.WorkspaceJob
		if err := json.Unmarshal(payload, &job); err != nil {
			return nil, fmt.Errorf("postgres store: decode workspace job: %w", err)
		}
		items = append(items, normalizeWorkspaceJob(tenantID, job))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate workspace jobs: %w", err)
	}

	return items, nil
}

func (s *PostgresStore) ensureTenant(ctx context.Context, tenantID string) (models.Tenant, error) {
	ctx, cancel := s.writeContext(ctx)
	defer cancel()

	if tenantID == DefaultTenantID {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO tenants (id, name, api_key, created_at, active, settings, quotas, service_accounts)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (id) DO NOTHING`,
			DefaultTenantID,
			defaultTenantName,
			"",
			time.Unix(0, 0).UTC(),
			true,
			`{}`,
			`{}`,
			`[]`,
		)
		if err != nil {
			return models.Tenant{}, fmt.Errorf("ensure default tenant: %w", err)
		}
		tenant, err := s.GetTenant(ctx, tenantID)
		if err != nil {
			return models.Tenant{}, fmt.Errorf("ensure default tenant: %w", err)
		}
		return *tenant, nil
	}

	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		return models.Tenant{}, fmt.Errorf("check tenant %s: %w", tenantID, err)
	}
	return *tenant, nil
}

func (s *PostgresStore) snapshotCount(ctx context.Context, tenantID string) (int, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots WHERE tenant_id = $1`, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count snapshots for tenant %s: %w", tenantID, err)
	}
	return count, nil
}

func (s *PostgresStore) migrationExists(ctx context.Context, tenantID, migrationID string) (bool, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM migrations WHERE tenant_id = $1 AND id = $2)`, tenantID, migrationID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %s for tenant %s: %w", migrationID, tenantID, err)
	}
	return exists, nil
}

func (s *PostgresStore) migrationCount(ctx context.Context, tenantID string) (int, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migrations WHERE tenant_id = $1`, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count migrations for tenant %s: %w", tenantID, err)
	}
	return count, nil
}

type persistedServiceAccount struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description,omitempty"`
	APIKey        string                    `json:"api_key,omitempty"`
	APIKeyHash    string                    `json:"api_key_hash,omitempty"`
	Role          models.TenantRole         `json:"role"`
	Active        bool                      `json:"active"`
	CreatedAt     time.Time                 `json:"created_at"`
	LastRotatedAt time.Time                 `json:"last_rotated_at,omitempty"`
	ExpiresAt     time.Time                 `json:"expires_at,omitempty"`
	Metadata      map[string]string         `json:"metadata,omitempty"`
	Permissions   []models.TenantPermission `json:"permissions,omitempty"`
}

func decodeTenantComponents(tenant *models.Tenant, settingsPayload, quotasPayload, serviceAccountsPayload []byte) error {
	if len(settingsPayload) > 0 {
		if err := json.Unmarshal(settingsPayload, &tenant.Settings); err != nil {
			return fmt.Errorf("decode settings: %w", err)
		}
	}
	if len(quotasPayload) > 0 {
		if err := json.Unmarshal(quotasPayload, &tenant.Quotas); err != nil {
			return fmt.Errorf("decode quotas: %w", err)
		}
	}
	if len(serviceAccountsPayload) > 0 {
		accounts, err := decodeServiceAccounts(serviceAccountsPayload)
		if err != nil {
			return fmt.Errorf("decode service accounts: %w", err)
		}
		tenant.ServiceAccounts = accounts
	}
	tenant.Settings = copyStringMap(tenant.Settings)
	tenant.ServiceAccounts = cloneServiceAccounts(tenant.ServiceAccounts)
	normalized, err := prepareTenantCredentials(*tenant)
	if err != nil {
		return fmt.Errorf("normalize tenant credentials: %w", err)
	}
	*tenant = normalized
	return nil
}

func marshalTenantComponent(name string, value interface{}) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal %s: %w", name, err)
	}
	return payload, nil
}

func marshalServiceAccounts(accounts []models.ServiceAccount) ([]byte, error) {
	items := make([]persistedServiceAccount, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, persistedServiceAccount{
			ID:            account.ID,
			Name:          account.Name,
			Description:   account.Description,
			APIKeyHash:    strings.TrimSpace(account.APIKeyHash),
			Role:          account.Role,
			Active:        account.Active,
			CreatedAt:     account.CreatedAt,
			LastRotatedAt: account.LastRotatedAt,
			ExpiresAt:     account.ExpiresAt,
			Metadata:      copyStringMap(account.Metadata),
			Permissions:   append([]models.TenantPermission(nil), account.Permissions...),
		})
	}
	return marshalTenantComponent("service_accounts", items)
}

func decodeServiceAccounts(payload []byte) ([]models.ServiceAccount, error) {
	if len(payload) == 0 {
		return nil, nil
	}

	var stored []persistedServiceAccount
	if err := json.Unmarshal(payload, &stored); err != nil {
		return nil, err
	}

	accounts := make([]models.ServiceAccount, 0, len(stored))
	for _, item := range stored {
		accounts = append(accounts, models.ServiceAccount{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			APIKey:        strings.TrimSpace(item.APIKey),
			APIKeyHash:    strings.TrimSpace(item.APIKeyHash),
			Role:          item.Role,
			Active:        item.Active,
			CreatedAt:     item.CreatedAt,
			LastRotatedAt: item.LastRotatedAt,
			ExpiresAt:     item.ExpiresAt,
			Metadata:      copyStringMap(item.Metadata),
			Permissions:   append([]models.TenantPermission(nil), item.Permissions...),
		})
	}
	return accounts, nil
}

func replaceTenantCredentialHashes(ctx context.Context, tx *sql.Tx, tenant models.Tenant) error {
	if tx == nil {
		return fmt.Errorf("replace credential hashes: transaction is nil")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM credential_hashes WHERE tenant_id = $1`, tenant.ID); err != nil {
		return fmt.Errorf("delete credential hashes for tenant %s: %w", tenant.ID, err)
	}
	return insertTenantCredentialHashes(ctx, tx, tenant)
}

func insertTenantCredentialHashes(ctx context.Context, tx *sql.Tx, tenant models.Tenant) error {
	if tx == nil {
		return fmt.Errorf("insert credential hashes: transaction is nil")
	}
	for _, entry := range orderedTenantCredentialHashes(tenant) {
		var insertedHash string
		if err := tx.QueryRowContext(
			ctx,
			`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 RETURNING credential_hash`,
			entry.hash,
			tenant.ID,
			credentialOwnerType(entry.owner),
			entry.owner.serviceAccountID,
		).Scan(&insertedHash); err != nil {
			return translateCredentialConstraintError(err)
		}
		if strings.TrimSpace(insertedHash) == "" {
			return fmt.Errorf("insert credential hash for tenant %s returned an empty identifier", tenant.ID)
		}
	}
	return nil
}

type tenantCredentialHashInsert struct {
	hash  string
	owner credentialOwner
}

func orderedTenantCredentialHashes(tenant models.Tenant) []tenantCredentialHashInsert {
	ordered := make([]tenantCredentialHashInsert, 0, 1+len(tenant.ServiceAccounts))
	seen := make(map[string]struct{}, 1+len(tenant.ServiceAccounts))

	if hash := strings.TrimSpace(tenant.APIKeyHash); hash != "" {
		ordered = append(ordered, tenantCredentialHashInsert{
			hash:  hash,
			owner: credentialOwner{tenantID: tenant.ID},
		})
		seen[hash] = struct{}{}
	}

	for _, account := range tenant.ServiceAccounts {
		hash := strings.TrimSpace(account.APIKeyHash)
		if hash == "" {
			continue
		}
		if _, exists := seen[hash]; exists {
			continue
		}
		ordered = append(ordered, tenantCredentialHashInsert{
			hash: hash,
			owner: credentialOwner{
				tenantID:         tenant.ID,
				serviceAccountID: account.ID,
			},
		})
		seen[hash] = struct{}{}
	}

	return ordered
}

func credentialOwnerType(owner credentialOwner) string {
	if strings.TrimSpace(owner.serviceAccountID) != "" {
		return "service_account"
	}
	return "tenant"
}

func translateCredentialConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code == "23505" && (strings.Contains(pqErr.Constraint, "credential_hashes") || pqErr.Table == "credential_hashes") {
			return &CredentialConflictError{}
		}
	}
	return err
}

func annotateCredentialMigrationError(err error) error {
	if err == nil {
		return nil
	}
	if IsCredentialConflict(err) {
		return fmt.Errorf("%w; resolve duplicated tenant or service-account API keys before restarting so each persisted credential is globally unique; see docs/operations/credential-migration.md", err)
	}
	return err
}

func migrateStoredCredentials(ctx context.Context, db *sql.DB) error {
	tenants, err := loadCredentialMigrationTenants(ctx, db)
	if err != nil {
		return err
	}

	for _, tenant := range tenants {
		if err := validateCredentialUniqueness(tenants, tenant); err != nil {
			return err
		}
	}

	if err := seedStoredCredentialHashes(ctx, db, tenants); err != nil {
		return err
	}
	return clearStoredCredentialPlaintext(ctx, db, tenants)
}

func loadCredentialMigrationTenants(ctx context.Context, db *sql.DB) ([]models.Tenant, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants := make([]models.Tenant, 0)
	for rows.Next() {
		var (
			tenant                 models.Tenant
			settingsPayload        []byte
			quotasPayload          []byte
			serviceAccountsPayload []byte
		)
		if err := rows.Scan(&tenant.ID, &tenant.Name, &tenant.APIKey, &tenant.APIKeyHash, &tenant.CreatedAt, &tenant.Active, &settingsPayload, &quotasPayload, &serviceAccountsPayload); err != nil {
			return nil, err
		}
		if err := decodeTenantComponents(&tenant, settingsPayload, quotasPayload, serviceAccountsPayload); err != nil {
			return nil, err
		}
		tenants = append(tenants, tenant)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tenants, nil
}

func seedStoredCredentialHashes(ctx context.Context, db *sql.DB, tenants []models.Tenant) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()

	for _, tenant := range tenants {
		if err := seedTenantCredentialHashes(ctx, tx, tenant); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func seedTenantCredentialHashes(ctx context.Context, tx *sql.Tx, tenant models.Tenant) error {
	if tx == nil {
		return fmt.Errorf("seed credential hashes: transaction is nil")
	}
	for _, entry := range orderedTenantCredentialHashes(tenant) {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (credential_hash) DO NOTHING`,
			entry.hash,
			tenant.ID,
			credentialOwnerType(entry.owner),
			entry.owner.serviceAccountID,
		); err != nil {
			return translateCredentialConstraintError(err)
		}
	}
	return nil
}

func clearStoredCredentialPlaintext(ctx context.Context, db *sql.DB, tenants []models.Tenant) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()

	for _, tenant := range tenants {
		if err := ensureTenantCredentialHashesPresent(ctx, tx, tenant); err != nil {
			return err
		}
		serviceAccountsPayload, err := marshalServiceAccounts(tenant.ServiceAccounts)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE tenants SET api_key = $2, api_key_hash = $3, service_accounts = $4 WHERE id = $1`,
			tenant.ID,
			"",
			tenant.APIKeyHash,
			serviceAccountsPayload,
		); err != nil {
			return translateCredentialConstraintError(err)
		}
	}

	return tx.Commit()
}

func ensureTenantCredentialHashesPresent(ctx context.Context, tx *sql.Tx, tenant models.Tenant) error {
	if tx == nil {
		return fmt.Errorf("ensure credential hashes present: transaction is nil")
	}
	for _, entry := range orderedTenantCredentialHashes(tenant) {
		var exists bool
		if err := tx.QueryRowContext(
			ctx,
			`SELECT EXISTS (
				SELECT 1
				FROM credential_hashes
				WHERE credential_hash = $1
				  AND tenant_id = $2
				  AND owner_type = $3
				  AND service_account_id = $4
			)`,
			entry.hash,
			tenant.ID,
			credentialOwnerType(entry.owner),
			entry.owner.serviceAccountID,
		).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("credential hash precondition missing for tenant %s; see docs/operations/credential-migration.md", tenant.ID)
		}
	}
	return nil
}

func nullTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}

	return value
}

func pageOffset(page, perPage int) int {
	if page <= 1 || perPage <= 0 {
		return 0
	}
	return (page - 1) * perPage
}

func normalizePageRequest(page, perPage int) (int, int, error) {
	switch {
	case page < 0:
		return 0, 0, fmt.Errorf("page must be greater than or equal to zero")
	case page == 0:
		page = 1
	}

	switch {
	case perPage <= 0:
		perPage = 50
	case perPage > 200:
		perPage = 200
	}

	return page, perPage, nil
}

func pageCapacity(total, perPage int) int {
	switch {
	case total <= 0:
		return 0
	case perPage <= 0:
		return total
	case total < perPage:
		return total
	default:
		return perPage
	}
}

func recordStoreSchemaHistory(ctx context.Context, db *sql.DB) error {
	if len(storeSchemaHistory) == 0 || storeSchemaHistory[len(storeSchemaHistory)-1].version != currentStoreSchemaVersion {
		return fmt.Errorf("store schema history is out of sync with current schema version %d", currentStoreSchemaVersion)
	}
	for _, item := range storeSchemaHistory {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, description, applied_at)
			 VALUES ($1, $2, NOW())
			 ON CONFLICT (version) DO NOTHING`,
			item.version,
			item.description,
		); err != nil {
			return err
		}
	}
	return nil
}

func applyCredentialHashUniqueIndexMigration(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("apply credential hash unique index migration: database is nil")
	}

	conflicts, err := duplicateCredentialOwners(ctx, db)
	if err != nil {
		return fmt.Errorf("preflight credential hash uniqueness: %w", err)
	}
	if len(conflicts) > 0 {
		return &CredentialConflictError{Owners: conflicts}
	}

	if err := executeStoreMigration(ctx, db, "008_credential_hash_unique.sql"); err != nil {
		return fmt.Errorf("apply credential hash unique index migration: %w", err)
	}
	return nil
}

func duplicateCredentialOwners(ctx context.Context, db *sql.DB) ([]CredentialConflictOwner, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT DISTINCT tenant_id, service_account_id
		 FROM credential_hashes
		 WHERE credential_hash IN (
		 	SELECT credential_hash
		 	FROM credential_hashes
		 	GROUP BY credential_hash
		 	HAVING COUNT(*) > 1
		 )
		 ORDER BY tenant_id ASC, service_account_id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	owners := make([]CredentialConflictOwner, 0)
	for rows.Next() {
		var owner CredentialConflictOwner
		if err := rows.Scan(&owner.TenantID, &owner.ServiceAccountID); err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return owners, nil
}

func executeStoreMigration(ctx context.Context, db *sql.DB, name string) error {
	migrationSQL, err := readStoreMigrationFile(name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(migrationSQL) == "" {
		return nil
	}
	if storeMigrationSkipsTransaction(migrationSQL) {
		_, err = db.ExecContext(ctx, migrationSQL)
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback is best effort after commit or terminal transaction failure.
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, migrationSQL); err != nil {
		return err
	}
	return tx.Commit()
}

func storeMigrationSkipsTransaction(payload string) bool {
	for _, line := range strings.Split(payload, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "--") {
			return false
		}
		if strings.Contains(trimmed, "MUST NOT be wrapped in a transaction") {
			return true
		}
	}
	return false
}

func readStoreMigrationFile(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("read store migration file: migration name is empty")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("read store migration file: caller path unavailable")
	}
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return "", fmt.Errorf("read store migration file: open store root: %w", err)
	}
	defer root.Close()

	migrationsRoot, err := root.OpenRoot("migrations")
	if err != nil {
		return "", fmt.Errorf("read store migration file: open migrations root: %w", err)
	}
	defer migrationsRoot.Close()

	migrationFile, err := migrationsRoot.Open(name)
	if err != nil {
		return "", fmt.Errorf("read store migration file %s: %w", name, err)
	}
	defer migrationFile.Close()

	payload, err := io.ReadAll(migrationFile)
	if err != nil {
		return "", fmt.Errorf("read store migration file %s: %w", name, err)
	}

	return string(payload), nil
}
