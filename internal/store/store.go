package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

const defaultTenantName = "Default Tenant"

// DefaultTenantID identifies the built-in single-tenant compatibility scope.
const DefaultTenantID = "default"

type tenantContextKey struct{}

// ContextWithTenantID annotates a context with a tenant identifier for store-backed flows.
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, normalizeTenantID(tenantID))
}

// TenantIDFromContext returns the tenant identifier attached to a context or the default tenant.
func TenantIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return DefaultTenantID
	}

	value, _ := ctx.Value(tenantContextKey{}).(string)
	return normalizeTenantID(value)
}

// Store defines persistence operations for discovery snapshots, migration state, pilot workspaces, and inventory.
type Store interface {
	// SaveDiscovery persists a discovery result and returns the generated snapshot identifier.
	SaveDiscovery(ctx context.Context, tenantID string, result *models.DiscoveryResult) (snapshotID string, err error)
	// GetSnapshot retrieves a previously persisted discovery snapshot by identifier.
	GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*models.DiscoveryResult, error)
	// ListSnapshots returns snapshot metadata ordered from newest to oldest.
	ListSnapshots(ctx context.Context, tenantID string, platform models.Platform, limit int) ([]SnapshotMeta, error)
	// ListSnapshotsPage returns a single page of snapshot metadata plus the total number of matching snapshots.
	ListSnapshotsPage(ctx context.Context, tenantID string, platform models.Platform, page, perPage int) ([]SnapshotMeta, int, error)
	// QueryVMs returns VMs that match the supplied filter criteria across stored snapshots.
	QueryVMs(ctx context.Context, tenantID string, filter VMFilter) ([]models.VirtualMachine, error)
	// SaveMigration persists a migration state payload by identifier.
	SaveMigration(ctx context.Context, tenantID string, record MigrationRecord) error
	// GetMigration retrieves a previously persisted migration state payload by identifier.
	GetMigration(ctx context.Context, tenantID, migrationID string) (*MigrationRecord, error)
	// ListMigrations returns migration metadata ordered from newest to oldest updates.
	ListMigrations(ctx context.Context, tenantID string, limit int) ([]MigrationMeta, error)
	// ListMigrationsPage returns a single page of migration metadata plus the total number of matching migrations.
	ListMigrationsPage(ctx context.Context, tenantID string, page, perPage int) ([]MigrationMeta, int, error)
	// SaveRecoveryPoint persists a rollback recovery point payload for a migration.
	SaveRecoveryPoint(ctx context.Context, tenantID string, record RecoveryPointRecord) error
	// GetRecoveryPoint retrieves the persisted rollback recovery point for a migration.
	GetRecoveryPoint(ctx context.Context, tenantID, migrationID string) (*RecoveryPointRecord, error)
	// CreateTenant persists tenant metadata and credentials for API isolation.
	CreateTenant(ctx context.Context, tenant models.Tenant) error
	// UpdateTenant overwrites persisted tenant metadata, quotas, and service accounts.
	UpdateTenant(ctx context.Context, tenant models.Tenant) error
	// GetTenant retrieves a tenant by identifier.
	GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error)
	// ListTenants returns all configured tenants ordered from oldest to newest.
	ListTenants(ctx context.Context) ([]models.Tenant, error)
	// DeleteTenant removes a tenant and any tenant-scoped data it owns.
	DeleteTenant(ctx context.Context, tenantID string) error
	// SaveAuditEvent persists a tenant-scoped audit event.
	SaveAuditEvent(ctx context.Context, event models.AuditEvent) error
	// ListAuditEvents returns tenant audit events ordered from newest to oldest.
	ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]models.AuditEvent, error)
	// RevokeAuthSession persists a dashboard session revocation until the original session expiry.
	RevokeAuthSession(ctx context.Context, sessionID string, expiresAt time.Time) error
	// IsAuthSessionRevoked reports whether a dashboard session identifier has been revoked and is still active.
	IsAuthSessionRevoked(ctx context.Context, sessionID string) (bool, error)
	// CreateWorkspace persists a new pilot workspace for the supplied tenant.
	CreateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error
	// UpdateWorkspace overwrites a previously persisted pilot workspace.
	UpdateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error
	// GetWorkspace retrieves a previously persisted pilot workspace by identifier.
	GetWorkspace(ctx context.Context, tenantID, workspaceID string) (*models.PilotWorkspace, error)
	// ListWorkspaces returns pilot workspaces ordered from newest to oldest updates.
	ListWorkspaces(ctx context.Context, tenantID string, limit int) ([]models.PilotWorkspace, error)
	// DeleteWorkspace removes a persisted pilot workspace and any background jobs tied to it.
	DeleteWorkspace(ctx context.Context, tenantID, workspaceID string) error
	// SaveWorkspaceJob persists a pilot workspace background job record.
	SaveWorkspaceJob(ctx context.Context, tenantID string, job models.WorkspaceJob) error
	// GetWorkspaceJob retrieves a previously persisted workspace background job record by identifier.
	GetWorkspaceJob(ctx context.Context, tenantID, workspaceID, jobID string) (*models.WorkspaceJob, error)
	// ListWorkspaceJobs returns workspace background jobs ordered from newest to oldest updates.
	ListWorkspaceJobs(ctx context.Context, tenantID, workspaceID string, limit int) ([]models.WorkspaceJob, error)
	// Close releases any store resources.
	Close() error
}

// Diagnostics summarizes operator-visible details about the active state store.
type Diagnostics struct {
	// Backend identifies the store implementation, such as memory or postgres.
	Backend string `json:"backend" yaml:"backend"`
	// SchemaVersion reports the latest applied schema version when the backend persists versioned metadata.
	SchemaVersion int `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	// Persistent reports whether the backend survives process restarts.
	Persistent bool `json:"persistent" yaml:"persistent"`
	// DBPool reports connection-pool settings and runtime metrics when the backend is SQL-backed.
	DBPool *DBPoolDiagnostics `json:"db_pool,omitempty" yaml:"db_pool,omitempty"`
}

// DBPoolDiagnostics reports SQL pool configuration and runtime counters.
type DBPoolDiagnostics struct {
	MaxOpenConnections int           `json:"max_open_connections,omitempty" yaml:"max_open_connections,omitempty"`
	OpenConnections    int           `json:"open_connections,omitempty" yaml:"open_connections,omitempty"`
	InUse              int           `json:"in_use,omitempty" yaml:"in_use,omitempty"`
	Idle               int           `json:"idle,omitempty" yaml:"idle,omitempty"`
	WaitCount          int64         `json:"wait_count,omitempty" yaml:"wait_count,omitempty"`
	WaitDuration       time.Duration `json:"wait_duration,omitempty" yaml:"wait_duration,omitempty"`
	MaxIdleClosed      int64         `json:"max_idle_closed,omitempty" yaml:"max_idle_closed,omitempty"`
	MaxIdleTimeClosed  int64         `json:"max_idle_time_closed,omitempty" yaml:"max_idle_time_closed,omitempty"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed,omitempty" yaml:"max_lifetime_closed,omitempty"`
	ReadTimeout        time.Duration `json:"read_timeout,omitempty" yaml:"read_timeout,omitempty"`
	WriteTimeout       time.Duration `json:"write_timeout,omitempty" yaml:"write_timeout,omitempty"`
}

// DiagnosticsProvider is implemented by stores that expose operator-visible backend metadata.
type DiagnosticsProvider interface {
	// Diagnostics returns backend metadata suitable for API about and troubleshooting output.
	Diagnostics(ctx context.Context) (Diagnostics, error)
}

// SnapshotMeta summarizes a persisted discovery snapshot.
type SnapshotMeta struct {
	// ID is the snapshot identifier.
	ID string `json:"id" yaml:"id"`
	// TenantID is the tenant that owns the snapshot.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// Source is the source system that produced the snapshot.
	Source string `json:"source" yaml:"source"`
	// Platform is the source platform associated with the snapshot.
	Platform models.Platform `json:"platform" yaml:"platform"`
	// VMCount is the number of VMs in the snapshot.
	VMCount int `json:"vm_count" yaml:"vm_count"`
	// DiscoveredAt is when the snapshot inventory was collected.
	DiscoveredAt time.Time `json:"discovered_at" yaml:"discovered_at"`
}

// VMFilter filters VM queries across one or more stored snapshots.
type VMFilter struct {
	// Platform restricts results to a specific platform when provided.
	Platform models.Platform `json:"platform,omitempty" yaml:"platform,omitempty"`
	// PowerState restricts results to VMs in the specified power state when provided.
	PowerState models.PowerState `json:"power_state,omitempty" yaml:"power_state,omitempty"`
	// NameContains performs a case-insensitive substring match against the VM name.
	NameContains string `json:"name_contains,omitempty" yaml:"name_contains,omitempty"`
	// Limit caps the number of returned VMs. Zero or negative values disable the cap.
	Limit int `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// MigrationRecord stores a serialized migration state alongside queryable metadata.
type MigrationRecord struct {
	// ID is the migration identifier.
	ID string `json:"id" yaml:"id"`
	// TenantID is the tenant that owns the migration record.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// SpecName is the migration specification name associated with the state.
	SpecName string `json:"spec_name" yaml:"spec_name"`
	// Phase is the current migration phase name.
	Phase string `json:"phase" yaml:"phase"`
	// StartedAt is when the migration began.
	StartedAt time.Time `json:"started_at" yaml:"started_at"`
	// UpdatedAt is when the migration state was most recently updated.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// CompletedAt is when the migration finished when known.
	CompletedAt time.Time `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	// RawJSON is the serialized migration state payload.
	RawJSON json.RawMessage `json:"raw_json" yaml:"raw_json"`
}

// MigrationMeta summarizes a persisted migration state for history views.
type MigrationMeta struct {
	// ID is the migration identifier.
	ID string `json:"id" yaml:"id"`
	// TenantID is the tenant that owns the migration record.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// SpecName is the migration specification name associated with the state.
	SpecName string `json:"spec_name" yaml:"spec_name"`
	// Phase is the current migration phase name.
	Phase string `json:"phase" yaml:"phase"`
	// StartedAt is when the migration began.
	StartedAt time.Time `json:"started_at" yaml:"started_at"`
	// UpdatedAt is when the migration state was most recently updated.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// CompletedAt is when the migration finished when known.
	CompletedAt time.Time `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
}

// RecoveryPointRecord stores a serialized rollback recovery point for a migration.
type RecoveryPointRecord struct {
	// MigrationID is the identifier of the migration the recovery point belongs to.
	MigrationID string `json:"migration_id" yaml:"migration_id"`
	// TenantID is the tenant that owns the recovery point.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// Phase is the migration phase the recovery point was captured for.
	Phase string `json:"phase" yaml:"phase"`
	// CreatedAt is when the recovery point was recorded.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// RawJSON is the serialized recovery point payload.
	RawJSON json.RawMessage `json:"raw_json" yaml:"raw_json"`
}

func normalizeTenantID(tenantID string) string {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return DefaultTenantID
	}

	return tenantID
}
