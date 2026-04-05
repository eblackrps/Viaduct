package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const createStoreSchemaSQL = `
CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	api_key TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	active BOOLEAN NOT NULL,
	settings JSONB NOT NULL DEFAULT '{}'::jsonb
);

INSERT INTO tenants (id, name, api_key, created_at, active, settings)
VALUES ('default', 'Default Tenant', '', to_timestamp(0), TRUE, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;

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
`

// PostgresStore is a PostgreSQL-backed Store implementation.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL and ensures the required schema exists.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres store: open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: ping database: %w", err)
	}

	if _, err := db.ExecContext(ctx, createStoreSchemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres store: run migrations: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// SaveDiscovery persists a discovery result to PostgreSQL.
func (s *PostgresStore) SaveDiscovery(ctx context.Context, tenantID string, result *models.DiscoveryResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("postgres store: save discovery: result is nil")
	}

	tenantID = normalizeTenantID(tenantID)
	if err := s.ensureTenant(ctx, tenantID); err != nil {
		return "", fmt.Errorf("postgres store: save discovery: %w", err)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("postgres store: marshal discovery: %w", err)
	}

	snapshotID := uuid.NewString()
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
		return "", fmt.Errorf("postgres store: insert snapshot: %w", err)
	}

	return snapshotID, nil
}

// GetSnapshot retrieves a discovery snapshot by identifier from PostgreSQL.
func (s *PostgresStore) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*models.DiscoveryResult, error) {
	tenantID = normalizeTenantID(tenantID)

	var payload []byte
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT raw_json FROM snapshots WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		snapshotID,
	).Scan(&payload); err != nil {
		return nil, fmt.Errorf("postgres store: get snapshot %s: %w", snapshotID, err)
	}

	var result models.DiscoveryResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("postgres store: decode snapshot %s: %w", snapshotID, err)
	}

	return &result, nil
}

// ListSnapshots returns snapshot metadata ordered from newest to oldest.
func (s *PostgresStore) ListSnapshots(ctx context.Context, tenantID string, platform models.Platform, limit int) ([]SnapshotMeta, error) {
	tenantID = normalizeTenantID(tenantID)

	query := `SELECT id, tenant_id, source, platform, vm_count, discovered_at FROM snapshots WHERE tenant_id = $1`
	args := []interface{}{tenantID}

	if platform != "" {
		query += ` AND platform = $2`
		args = append(args, string(platform))
	}

	query += ` ORDER BY discovered_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list snapshots: %w", err)
	}
	defer rows.Close()

	items := make([]SnapshotMeta, 0)
	for rows.Next() {
		var item SnapshotMeta
		var platformName string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Source, &platformName, &item.VMCount, &item.DiscoveredAt); err != nil {
			return nil, fmt.Errorf("postgres store: scan snapshot metadata: %w", err)
		}
		item.Platform = models.Platform(platformName)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate snapshot metadata: %w", err)
	}

	return items, nil
}

// QueryVMs returns stored VMs that match the supplied filter criteria.
func (s *PostgresStore) QueryVMs(ctx context.Context, tenantID string, filter VMFilter) ([]models.VirtualMachine, error) {
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
func (s *PostgresStore) SaveMigration(ctx context.Context, tenantID string, record MigrationRecord) error {
	if record.ID == "" {
		return fmt.Errorf("postgres store: save migration: migration ID is empty")
	}
	if len(record.RawJSON) == 0 {
		return fmt.Errorf("postgres store: save migration: raw JSON is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID
	if err := s.ensureTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("postgres store: save migration: %w", err)
	}

	_, err := s.db.ExecContext(
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
func (s *PostgresStore) GetMigration(ctx context.Context, tenantID, migrationID string) (*MigrationRecord, error) {
	tenantID = normalizeTenantID(tenantID)

	var record MigrationRecord
	var completedAt sql.NullTime
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, spec_name, phase, started_at, updated_at, completed_at, raw_json
		 FROM migrations WHERE tenant_id = $1 AND id = $2`,
		tenantID,
		migrationID,
	).Scan(&record.ID, &record.TenantID, &record.SpecName, &record.Phase, &record.StartedAt, &record.UpdatedAt, &completedAt, &record.RawJSON); err != nil {
		return nil, fmt.Errorf("postgres store: get migration %s: %w", migrationID, err)
	}

	if completedAt.Valid {
		record.CompletedAt = completedAt.Time
	}

	return &record, nil
}

// ListMigrations returns migration metadata ordered by most recent update.
func (s *PostgresStore) ListMigrations(ctx context.Context, tenantID string, limit int) ([]MigrationMeta, error) {
	tenantID = normalizeTenantID(tenantID)

	query := `SELECT id, tenant_id, spec_name, phase, started_at, updated_at, completed_at
		FROM migrations WHERE tenant_id = $1 ORDER BY updated_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list migrations: %w", err)
	}
	defer rows.Close()

	items := make([]MigrationMeta, 0)
	for rows.Next() {
		var item MigrationMeta
		var completedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.TenantID, &item.SpecName, &item.Phase, &item.StartedAt, &item.UpdatedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("postgres store: scan migration metadata: %w", err)
		}
		if completedAt.Valid {
			item.CompletedAt = completedAt.Time
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres store: iterate migration metadata: %w", err)
	}

	return items, nil
}

// SaveRecoveryPoint persists a serialized recovery point to PostgreSQL.
func (s *PostgresStore) SaveRecoveryPoint(ctx context.Context, tenantID string, record RecoveryPointRecord) error {
	if record.MigrationID == "" {
		return fmt.Errorf("postgres store: save recovery point: migration ID is empty")
	}
	if len(record.RawJSON) == 0 {
		return fmt.Errorf("postgres store: save recovery point: raw JSON is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID
	if err := s.ensureTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("postgres store: save recovery point: %w", err)
	}

	_, err := s.db.ExecContext(
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
func (s *PostgresStore) GetRecoveryPoint(ctx context.Context, tenantID, migrationID string) (*RecoveryPointRecord, error) {
	tenantID = normalizeTenantID(tenantID)

	var record RecoveryPointRecord
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT tenant_id, migration_id, phase, created_at, raw_json
		 FROM recovery_points WHERE tenant_id = $1 AND migration_id = $2`,
		tenantID,
		migrationID,
	).Scan(&record.TenantID, &record.MigrationID, &record.Phase, &record.CreatedAt, &record.RawJSON); err != nil {
		return nil, fmt.Errorf("postgres store: get recovery point %s: %w", migrationID, err)
	}

	return &record, nil
}

// CreateTenant persists tenant metadata in PostgreSQL.
func (s *PostgresStore) CreateTenant(ctx context.Context, tenant models.Tenant) error {
	tenant.ID = normalizeTenantID(tenant.ID)
	if tenant.ID == DefaultTenantID {
		tenant = defaultTenant()
	}
	if tenant.Name == "" {
		return fmt.Errorf("postgres store: create tenant: tenant name is empty")
	}
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now().UTC()
	}
	if tenant.Settings == nil {
		tenant.Settings = map[string]string{}
	}

	settings, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant: marshal settings: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO tenants (id, name, api_key, created_at, active, settings)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		tenant.ID,
		tenant.Name,
		tenant.APIKey,
		tenant.CreatedAt,
		tenant.Active,
		settings,
	)
	if err != nil {
		return fmt.Errorf("postgres store: create tenant %s: %w", tenant.ID, err)
	}

	return nil
}

// GetTenant retrieves a tenant from PostgreSQL by identifier.
func (s *PostgresStore) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	tenantID = normalizeTenantID(tenantID)

	var tenant models.Tenant
	var settingsPayload []byte
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, api_key, created_at, active, settings FROM tenants WHERE id = $1`,
		tenantID,
	).Scan(&tenant.ID, &tenant.Name, &tenant.APIKey, &tenant.CreatedAt, &tenant.Active, &settingsPayload); err != nil {
		return nil, fmt.Errorf("postgres store: get tenant %s: %w", tenantID, err)
	}

	if len(settingsPayload) > 0 {
		if err := json.Unmarshal(settingsPayload, &tenant.Settings); err != nil {
			return nil, fmt.Errorf("postgres store: get tenant %s: decode settings: %w", tenantID, err)
		}
	}

	return &tenant, nil
}

// ListTenants returns all configured tenants from PostgreSQL.
func (s *PostgresStore) ListTenants(ctx context.Context) ([]models.Tenant, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, api_key, created_at, active, settings FROM tenants ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("postgres store: list tenants: %w", err)
	}
	defer rows.Close()

	items := make([]models.Tenant, 0)
	for rows.Next() {
		var tenant models.Tenant
		var settingsPayload []byte
		if err := rows.Scan(&tenant.ID, &tenant.Name, &tenant.APIKey, &tenant.CreatedAt, &tenant.Active, &settingsPayload); err != nil {
			return nil, fmt.Errorf("postgres store: scan tenant: %w", err)
		}
		if len(settingsPayload) > 0 {
			if err := json.Unmarshal(settingsPayload, &tenant.Settings); err != nil {
				return nil, fmt.Errorf("postgres store: decode tenant settings: %w", err)
			}
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
	tenantID = normalizeTenantID(tenantID)
	if tenantID == DefaultTenantID {
		return fmt.Errorf("postgres store: delete tenant: default tenant cannot be removed")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres store: delete tenant %s: begin transaction: %w", tenantID, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, statement := range []string{
		`DELETE FROM recovery_points WHERE tenant_id = $1`,
		`DELETE FROM migrations WHERE tenant_id = $1`,
		`DELETE FROM snapshots WHERE tenant_id = $1`,
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

func (s *PostgresStore) ensureTenant(ctx context.Context, tenantID string) error {
	if tenantID == DefaultTenantID {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO tenants (id, name, api_key, created_at, active, settings)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO NOTHING`,
			DefaultTenantID,
			defaultTenantName,
			"",
			time.Unix(0, 0).UTC(),
			true,
			`{}`,
		)
		if err != nil {
			return fmt.Errorf("ensure default tenant: %w", err)
		}
		return nil
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM tenants WHERE id = $1)`, tenantID).Scan(&exists); err != nil {
		return fmt.Errorf("check tenant %s: %w", tenantID, err)
	}
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	return nil
}

func nullTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}

	return value
}
