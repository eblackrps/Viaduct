package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/google/uuid"
)

type storedSnapshot struct {
	tenantID string
	result   *models.DiscoveryResult
}

type storedMigration struct {
	tenantID string
	record   MigrationRecord
}

type storedRecoveryPoint struct {
	tenantID string
	record   RecoveryPointRecord
}

type storedWorkspace struct {
	tenantID  string
	workspace models.PilotWorkspace
}

type storedWorkspaceJob struct {
	tenantID string
	job      models.WorkspaceJob
}

// MemoryStore is an in-memory Store implementation for testing and small deployments.
type MemoryStore struct {
	mu             sync.RWMutex
	tenants        map[string]models.Tenant
	snapshots      map[string]storedSnapshot
	migrations     map[string]storedMigration
	recoveryPoints map[string]storedRecoveryPoint
	auditEvents    map[string][]models.AuditEvent
	workspaces     map[string]storedWorkspace
	workspaceJobs  map[string]storedWorkspaceJob
}

// NewMemoryStore creates an empty in-memory discovery snapshot store.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		tenants:        make(map[string]models.Tenant),
		snapshots:      make(map[string]storedSnapshot),
		migrations:     make(map[string]storedMigration),
		recoveryPoints: make(map[string]storedRecoveryPoint),
		auditEvents:    make(map[string][]models.AuditEvent),
		workspaces:     make(map[string]storedWorkspace),
		workspaceJobs:  make(map[string]storedWorkspaceJob),
	}

	store.tenants[DefaultTenantID] = defaultTenant()
	return store
}

// SaveDiscovery persists a discovery snapshot in memory.
func (s *MemoryStore) SaveDiscovery(ctx context.Context, tenantID string, result *models.DiscoveryResult) (string, error) {
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("memory store: save discovery: %w", ctx.Err())
	default:
	}

	if result == nil {
		return "", fmt.Errorf("memory store: save discovery: result is nil")
	}

	tenantID = normalizeTenantID(tenantID)
	snapshotID := uuid.NewString()
	cloned, err := cloneDiscoveryResult(result)
	if err != nil {
		return "", fmt.Errorf("memory store: save discovery: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tenant, err := s.ensureTenantLocked(tenantID)
	if err != nil {
		return "", fmt.Errorf("memory store: save discovery: %w", err)
	}
	if limit := tenant.Quotas.MaxSnapshots; limit > 0 && s.snapshotCountLocked(tenantID) >= limit {
		return "", fmt.Errorf("memory store: save discovery: snapshot quota exceeded for tenant %s", tenantID)
	}

	s.snapshots[snapshotID] = storedSnapshot{
		tenantID: tenantID,
		result:   cloned,
	}
	return snapshotID, nil
}

// GetSnapshot retrieves a stored in-memory discovery snapshot by identifier.
func (s *MemoryStore) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*models.DiscoveryResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get snapshot: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.snapshots[snapshotID]
	if !ok || item.tenantID != tenantID {
		return nil, fmt.Errorf("memory store: get snapshot %s: not found", snapshotID)
	}

	cloned, err := cloneDiscoveryResult(item.result)
	if err != nil {
		return nil, fmt.Errorf("memory store: get snapshot: %w", err)
	}

	return cloned, nil
}

// ListSnapshots returns snapshot metadata from the in-memory store.
func (s *MemoryStore) ListSnapshots(ctx context.Context, tenantID string, platform models.Platform, limit int) ([]SnapshotMeta, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list snapshots: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]SnapshotMeta, 0, len(s.snapshots))
	for id, snapshot := range s.snapshots {
		if snapshot.result == nil || snapshot.tenantID != tenantID {
			continue
		}
		if platform != "" && snapshot.result.Platform != platform {
			continue
		}

		items = append(items, SnapshotMeta{
			ID:           id,
			TenantID:     tenantID,
			Source:       snapshot.result.Source,
			Platform:     snapshot.result.Platform,
			VMCount:      len(snapshot.result.VMs),
			DiscoveredAt: snapshot.result.DiscoveredAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].DiscoveredAt.After(items[j].DiscoveredAt)
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// QueryVMs returns VMs from stored snapshots that match the supplied filter.
func (s *MemoryStore) QueryVMs(ctx context.Context, tenantID string, filter VMFilter) ([]models.VirtualMachine, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: query VMs: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]models.VirtualMachine, 0)
	for _, snapshot := range s.snapshots {
		if snapshot.result == nil || snapshot.tenantID != tenantID {
			continue
		}

		for _, vm := range snapshot.result.VMs {
			if !matchesFilter(vm, filter) {
				continue
			}

			results = append(results, vm)
			if filter.Limit > 0 && len(results) >= filter.Limit {
				return results, nil
			}
		}
	}

	return results, nil
}

// SaveMigration persists a serialized migration record in memory.
func (s *MemoryStore) SaveMigration(ctx context.Context, tenantID string, record MigrationRecord) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: save migration: %w", ctx.Err())
	default:
	}

	if record.ID == "" {
		return fmt.Errorf("memory store: save migration: migration ID is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID

	s.mu.Lock()
	defer s.mu.Unlock()

	tenant, err := s.ensureTenantLocked(tenantID)
	if err != nil {
		return fmt.Errorf("memory store: save migration: %w", err)
	}
	if _, exists := s.migrations[tenantMigrationKey(tenantID, record.ID)]; !exists {
		if limit := tenant.Quotas.MaxMigrations; limit > 0 && s.migrationCountLocked(tenantID) >= limit {
			return fmt.Errorf("memory store: save migration: migration quota exceeded for tenant %s", tenantID)
		}
	}

	s.migrations[tenantMigrationKey(tenantID, record.ID)] = storedMigration{
		tenantID: tenantID,
		record:   cloneMigrationRecord(record),
	}
	return nil
}

// GetMigration retrieves a stored migration record by identifier.
func (s *MemoryStore) GetMigration(ctx context.Context, tenantID, migrationID string) (*MigrationRecord, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get migration: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.migrations[tenantMigrationKey(tenantID, migrationID)]
	if !ok || item.tenantID != tenantID {
		return nil, fmt.Errorf("memory store: get migration %s: not found", migrationID)
	}

	cloned := cloneMigrationRecord(item.record)
	return &cloned, nil
}

// ListMigrations returns migration metadata ordered from newest to oldest.
func (s *MemoryStore) ListMigrations(ctx context.Context, tenantID string, limit int) ([]MigrationMeta, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list migrations: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]MigrationMeta, 0, len(s.migrations))
	for _, item := range s.migrations {
		if item.tenantID != tenantID {
			continue
		}

		items = append(items, MigrationMeta{
			ID:          item.record.ID,
			TenantID:    tenantID,
			SpecName:    item.record.SpecName,
			Phase:       item.record.Phase,
			StartedAt:   item.record.StartedAt,
			UpdatedAt:   item.record.UpdatedAt,
			CompletedAt: item.record.CompletedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// SaveRecoveryPoint persists a serialized recovery point in memory.
func (s *MemoryStore) SaveRecoveryPoint(ctx context.Context, tenantID string, record RecoveryPointRecord) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: save recovery point: %w", ctx.Err())
	default:
	}

	if record.MigrationID == "" {
		return fmt.Errorf("memory store: save recovery point: migration ID is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	record.TenantID = tenantID

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.ensureTenantLocked(tenantID); err != nil {
		return fmt.Errorf("memory store: save recovery point: %w", err)
	}

	s.recoveryPoints[tenantRecoveryPointKey(tenantID, record.MigrationID)] = storedRecoveryPoint{
		tenantID: tenantID,
		record:   cloneRecoveryPointRecord(record),
	}
	return nil
}

// GetRecoveryPoint retrieves a stored recovery point by migration identifier.
func (s *MemoryStore) GetRecoveryPoint(ctx context.Context, tenantID, migrationID string) (*RecoveryPointRecord, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get recovery point: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.recoveryPoints[tenantRecoveryPointKey(tenantID, migrationID)]
	if !ok || item.tenantID != tenantID {
		return nil, fmt.Errorf("memory store: get recovery point %s: not found", migrationID)
	}

	cloned := cloneRecoveryPointRecord(item.record)
	return &cloned, nil
}

// CreateTenant persists tenant metadata in memory.
func (s *MemoryStore) CreateTenant(ctx context.Context, tenant models.Tenant) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: create tenant: %w", ctx.Err())
	default:
	}

	tenant.ID = normalizeTenantID(tenant.ID)
	if tenant.ID == DefaultTenantID {
		tenant = defaultTenant()
	}
	if tenant.Name == "" {
		return fmt.Errorf("memory store: create tenant: tenant name is empty")
	}
	tenant = normalizeTenant(tenant)

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.tenants[tenant.ID]; ok && existing.ID != "" {
		return fmt.Errorf("memory store: create tenant %s: already exists", tenant.ID)
	}

	s.tenants[tenant.ID] = cloneTenant(tenant)
	return nil
}

// UpdateTenant overwrites persisted tenant metadata in memory.
func (s *MemoryStore) UpdateTenant(ctx context.Context, tenant models.Tenant) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: update tenant: %w", ctx.Err())
	default:
	}

	tenant.ID = normalizeTenantID(tenant.ID)
	if tenant.Name == "" {
		return fmt.Errorf("memory store: update tenant: tenant name is empty")
	}
	if tenant.ID == DefaultTenantID {
		existing := defaultTenant()
		existing.APIKey = tenant.APIKey
		existing.Active = tenant.Active
		existing.Settings = copyStringMap(tenant.Settings)
		existing.Quotas = tenant.Quotas
		existing.ServiceAccounts = cloneServiceAccounts(tenant.ServiceAccounts)
		tenant = existing
	} else {
		tenant = normalizeTenant(tenant)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tenants[tenant.ID]; !ok {
		return fmt.Errorf("memory store: update tenant %s: not found", tenant.ID)
	}

	s.tenants[tenant.ID] = cloneTenant(tenant)
	return nil
}

// GetTenant retrieves a stored tenant by identifier.
func (s *MemoryStore) GetTenant(ctx context.Context, tenantID string) (*models.Tenant, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get tenant: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	tenant, ok := s.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("memory store: get tenant %s: not found", tenantID)
	}

	cloned := cloneTenant(tenant)
	return &cloned, nil
}

// ListTenants returns all configured tenants in chronological order.
func (s *MemoryStore) ListTenants(ctx context.Context) ([]models.Tenant, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list tenants: %w", ctx.Err())
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]models.Tenant, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		items = append(items, cloneTenant(tenant))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

// DeleteTenant removes a tenant and all associated tenant-scoped data.
func (s *MemoryStore) DeleteTenant(ctx context.Context, tenantID string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: delete tenant: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)
	if tenantID == DefaultTenantID {
		return fmt.Errorf("memory store: delete tenant: default tenant cannot be removed")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tenants[tenantID]; !ok {
		return fmt.Errorf("memory store: delete tenant %s: not found", tenantID)
	}

	delete(s.tenants, tenantID)
	for id, snapshot := range s.snapshots {
		if snapshot.tenantID == tenantID {
			delete(s.snapshots, id)
		}
	}
	for id, migration := range s.migrations {
		if migration.tenantID == tenantID {
			delete(s.migrations, id)
		}
	}
	for key, recoveryPoint := range s.recoveryPoints {
		if recoveryPoint.tenantID == tenantID {
			delete(s.recoveryPoints, key)
		}
	}
	for key, workspace := range s.workspaces {
		if workspace.tenantID == tenantID {
			delete(s.workspaces, key)
		}
	}
	for key, job := range s.workspaceJobs {
		if job.tenantID == tenantID {
			delete(s.workspaceJobs, key)
		}
	}
	delete(s.auditEvents, tenantID)

	return nil
}

// SaveAuditEvent persists a tenant-scoped audit event in memory.
func (s *MemoryStore) SaveAuditEvent(ctx context.Context, event models.AuditEvent) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: save audit event: %w", ctx.Err())
	default:
	}

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

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.ensureTenantLocked(event.TenantID); err != nil {
		return fmt.Errorf("memory store: save audit event: %w", err)
	}

	cloned := event
	cloned.Details = copyStringMap(event.Details)
	s.auditEvents[event.TenantID] = append(s.auditEvents[event.TenantID], cloned)
	return nil
}

// ListAuditEvents returns tenant audit events ordered from newest to oldest.
func (s *MemoryStore) ListAuditEvents(ctx context.Context, tenantID string, limit int) ([]models.AuditEvent, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list audit events: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	source := s.auditEvents[tenantID]
	items := make([]models.AuditEvent, 0, len(source))
	for _, event := range source {
		cloned := event
		cloned.Details = copyStringMap(event.Details)
		items = append(items, cloned)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// CreateWorkspace persists a pilot workspace in memory.
func (s *MemoryStore) CreateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: create workspace: %w", ctx.Err())
	default:
	}

	if strings.TrimSpace(workspace.ID) == "" {
		return fmt.Errorf("memory store: create workspace: workspace ID is empty")
	}
	if strings.TrimSpace(workspace.Name) == "" {
		return fmt.Errorf("memory store: create workspace: workspace name is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	workspace = normalizeWorkspace(tenantID, workspace)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.ensureTenantLocked(tenantID); err != nil {
		return fmt.Errorf("memory store: create workspace: %w", err)
	}
	key := tenantWorkspaceKey(tenantID, workspace.ID)
	if _, exists := s.workspaces[key]; exists {
		return fmt.Errorf("memory store: create workspace %s: already exists", workspace.ID)
	}

	cloned, err := cloneWorkspace(workspace)
	if err != nil {
		return fmt.Errorf("memory store: create workspace: %w", err)
	}
	s.workspaces[key] = storedWorkspace{tenantID: tenantID, workspace: cloned}
	return nil
}

// UpdateWorkspace overwrites a persisted pilot workspace in memory.
func (s *MemoryStore) UpdateWorkspace(ctx context.Context, tenantID string, workspace models.PilotWorkspace) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: update workspace: %w", ctx.Err())
	default:
	}

	if strings.TrimSpace(workspace.ID) == "" {
		return fmt.Errorf("memory store: update workspace: workspace ID is empty")
	}
	if strings.TrimSpace(workspace.Name) == "" {
		return fmt.Errorf("memory store: update workspace: workspace name is empty")
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.ensureTenantLocked(tenantID); err != nil {
		return fmt.Errorf("memory store: update workspace: %w", err)
	}

	key := tenantWorkspaceKey(tenantID, workspace.ID)
	existing, ok := s.workspaces[key]
	if !ok || existing.tenantID != tenantID {
		return fmt.Errorf("memory store: update workspace %s: not found", workspace.ID)
	}

	createdAt := workspace.CreatedAt
	workspace = normalizeWorkspace(tenantID, workspace)
	if createdAt.IsZero() {
		workspace.CreatedAt = existing.workspace.CreatedAt
	}
	if workspace.UpdatedAt.IsZero() || !workspace.UpdatedAt.After(existing.workspace.UpdatedAt) {
		workspace.UpdatedAt = time.Now().UTC()
	}

	cloned, err := cloneWorkspace(workspace)
	if err != nil {
		return fmt.Errorf("memory store: update workspace: %w", err)
	}
	s.workspaces[key] = storedWorkspace{tenantID: tenantID, workspace: cloned}
	return nil
}

// GetWorkspace retrieves a persisted pilot workspace by identifier.
func (s *MemoryStore) GetWorkspace(ctx context.Context, tenantID, workspaceID string) (*models.PilotWorkspace, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get workspace: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.workspaces[tenantWorkspaceKey(tenantID, workspaceID)]
	if !ok || item.tenantID != tenantID {
		return nil, fmt.Errorf("memory store: get workspace %s: not found", workspaceID)
	}

	cloned, err := cloneWorkspace(item.workspace)
	if err != nil {
		return nil, fmt.Errorf("memory store: get workspace: %w", err)
	}
	return &cloned, nil
}

// ListWorkspaces returns pilot workspaces ordered from newest to oldest updates.
func (s *MemoryStore) ListWorkspaces(ctx context.Context, tenantID string, limit int) ([]models.PilotWorkspace, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list workspaces: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]models.PilotWorkspace, 0, len(s.workspaces))
	for _, item := range s.workspaces {
		if item.tenantID != tenantID {
			continue
		}
		cloned, err := cloneWorkspace(item.workspace)
		if err != nil {
			return nil, fmt.Errorf("memory store: list workspaces: %w", err)
		}
		items = append(items, cloned)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// SaveWorkspaceJob persists a pilot workspace background job in memory.
func (s *MemoryStore) SaveWorkspaceJob(ctx context.Context, tenantID string, job models.WorkspaceJob) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("memory store: save workspace job: %w", ctx.Err())
	default:
	}

	if strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("memory store: save workspace job: job ID is empty")
	}
	if strings.TrimSpace(job.WorkspaceID) == "" {
		return fmt.Errorf("memory store: save workspace job: workspace ID is empty")
	}

	tenantID = normalizeTenantID(tenantID)
	job = normalizeWorkspaceJob(tenantID, job)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.ensureTenantLocked(tenantID); err != nil {
		return fmt.Errorf("memory store: save workspace job: %w", err)
	}
	if _, ok := s.workspaces[tenantWorkspaceKey(tenantID, job.WorkspaceID)]; !ok {
		return fmt.Errorf("memory store: save workspace job: workspace %s not found", job.WorkspaceID)
	}

	cloned, err := cloneWorkspaceJob(job)
	if err != nil {
		return fmt.Errorf("memory store: save workspace job: %w", err)
	}
	s.workspaceJobs[tenantWorkspaceJobKey(tenantID, job.WorkspaceID, job.ID)] = storedWorkspaceJob{
		tenantID: tenantID,
		job:      cloned,
	}
	return nil
}

// GetWorkspaceJob retrieves a persisted pilot workspace background job by identifier.
func (s *MemoryStore) GetWorkspaceJob(ctx context.Context, tenantID, workspaceID, jobID string) (*models.WorkspaceJob, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: get workspace job: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.workspaceJobs[tenantWorkspaceJobKey(tenantID, workspaceID, jobID)]
	if !ok || item.tenantID != tenantID {
		return nil, fmt.Errorf("memory store: get workspace job %s: not found", jobID)
	}

	cloned, err := cloneWorkspaceJob(item.job)
	if err != nil {
		return nil, fmt.Errorf("memory store: get workspace job: %w", err)
	}
	return &cloned, nil
}

// ListWorkspaceJobs returns pilot workspace background jobs ordered from newest to oldest updates.
func (s *MemoryStore) ListWorkspaceJobs(ctx context.Context, tenantID, workspaceID string, limit int) ([]models.WorkspaceJob, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: list workspace jobs: %w", ctx.Err())
	default:
	}

	tenantID = normalizeTenantID(tenantID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]models.WorkspaceJob, 0, len(s.workspaceJobs))
	for _, item := range s.workspaceJobs {
		if item.tenantID != tenantID || item.job.WorkspaceID != workspaceID {
			continue
		}
		cloned, err := cloneWorkspaceJob(item.job)
		if err != nil {
			return nil, fmt.Errorf("memory store: list workspace jobs: %w", err)
		}
		items = append(items, cloned)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// Close releases resources held by the in-memory store.
func (s *MemoryStore) Close() error {
	return nil
}

// Diagnostics returns backend metadata for the in-memory store.
func (s *MemoryStore) Diagnostics(ctx context.Context) (Diagnostics, error) {
	return Diagnostics{
		Backend:    "memory",
		Persistent: false,
	}, nil
}

func (s *MemoryStore) ensureTenantLocked(tenantID string) (models.Tenant, error) {
	if tenant, ok := s.tenants[tenantID]; ok {
		return tenant, nil
	}
	if tenantID == DefaultTenantID {
		tenant := defaultTenant()
		s.tenants[tenantID] = tenant
		return tenant, nil
	}

	return models.Tenant{}, fmt.Errorf("tenant %s not found", tenantID)
}

func (s *MemoryStore) snapshotCountLocked(tenantID string) int {
	count := 0
	for _, snapshot := range s.snapshots {
		if snapshot.tenantID == tenantID {
			count++
		}
	}
	return count
}

func (s *MemoryStore) migrationCountLocked(tenantID string) int {
	count := 0
	for _, migration := range s.migrations {
		if migration.tenantID == tenantID {
			count++
		}
	}
	return count
}

func matchesFilter(vm models.VirtualMachine, filter VMFilter) bool {
	if filter.Platform != "" && vm.Platform != filter.Platform {
		return false
	}

	if filter.PowerState != "" && vm.PowerState != filter.PowerState {
		return false
	}

	if filter.NameContains != "" && !strings.Contains(strings.ToLower(vm.Name), strings.ToLower(filter.NameContains)) {
		return false
	}

	return true
}

func cloneDiscoveryResult(result *models.DiscoveryResult) (*models.DiscoveryResult, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("clone discovery result: %w", err)
	}

	var cloned models.DiscoveryResult
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, fmt.Errorf("clone discovery result: %w", err)
	}

	return &cloned, nil
}

func cloneMigrationRecord(record MigrationRecord) MigrationRecord {
	record.RawJSON = append(json.RawMessage(nil), record.RawJSON...)
	return record
}

func cloneRecoveryPointRecord(record RecoveryPointRecord) RecoveryPointRecord {
	record.RawJSON = append(json.RawMessage(nil), record.RawJSON...)
	return record
}

func cloneWorkspace(workspace models.PilotWorkspace) (models.PilotWorkspace, error) {
	payload, err := json.Marshal(workspace)
	if err != nil {
		return models.PilotWorkspace{}, fmt.Errorf("clone workspace: %w", err)
	}

	var cloned models.PilotWorkspace
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return models.PilotWorkspace{}, fmt.Errorf("clone workspace: %w", err)
	}
	return cloned, nil
}

func cloneWorkspaceJob(job models.WorkspaceJob) (models.WorkspaceJob, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return models.WorkspaceJob{}, fmt.Errorf("clone workspace job: %w", err)
	}

	var cloned models.WorkspaceJob
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return models.WorkspaceJob{}, fmt.Errorf("clone workspace job: %w", err)
	}
	return cloned, nil
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}

	return cloned
}

func cloneServiceAccounts(accounts []models.ServiceAccount) []models.ServiceAccount {
	if len(accounts) == 0 {
		return nil
	}

	cloned := make([]models.ServiceAccount, 0, len(accounts))
	for _, account := range accounts {
		item := account
		item.Metadata = copyStringMap(account.Metadata)
		if len(account.Permissions) > 0 {
			item.Permissions = append([]models.TenantPermission(nil), account.Permissions...)
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func cloneTenant(tenant models.Tenant) models.Tenant {
	tenant.Settings = copyStringMap(tenant.Settings)
	tenant.ServiceAccounts = cloneServiceAccounts(tenant.ServiceAccounts)
	return tenant
}

func normalizeTenant(tenant models.Tenant) models.Tenant {
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now().UTC()
	}
	if tenant.Settings == nil {
		tenant.Settings = make(map[string]string)
	}
	for index := range tenant.ServiceAccounts {
		if tenant.ServiceAccounts[index].Role == "" {
			tenant.ServiceAccounts[index].Role = models.TenantRoleViewer
		}
		if tenant.ServiceAccounts[index].CreatedAt.IsZero() {
			tenant.ServiceAccounts[index].CreatedAt = tenant.CreatedAt
		}
		if tenant.ServiceAccounts[index].Metadata == nil {
			tenant.ServiceAccounts[index].Metadata = make(map[string]string)
		}
		if len(tenant.ServiceAccounts[index].Permissions) > 0 {
			permissions := make([]models.TenantPermission, 0, len(tenant.ServiceAccounts[index].Permissions))
			for _, permission := range tenant.ServiceAccounts[index].Permissions {
				if permission.Valid() {
					permissions = append(permissions, permission)
				}
			}
			tenant.ServiceAccounts[index].Permissions = permissions
		}
	}
	return tenant
}

func normalizeWorkspace(tenantID string, workspace models.PilotWorkspace) models.PilotWorkspace {
	now := time.Now().UTC()
	workspace.TenantID = normalizeTenantID(tenantID)
	workspace.Name = strings.TrimSpace(workspace.Name)
	workspace.Description = strings.TrimSpace(workspace.Description)
	if workspace.Status == "" {
		workspace.Status = models.PilotWorkspaceStatusDraft
	}
	if workspace.CreatedAt.IsZero() {
		workspace.CreatedAt = now
	}
	if workspace.UpdatedAt.IsZero() {
		workspace.UpdatedAt = workspace.CreatedAt
	}
	workspace.SelectedWorkloadIDs = copyStringSlice(workspace.SelectedWorkloadIDs)
	for index := range workspace.SourceConnections {
		if strings.TrimSpace(workspace.SourceConnections[index].ID) == "" {
			workspace.SourceConnections[index].ID = uuid.NewString()
		}
		workspace.SourceConnections[index].Name = strings.TrimSpace(workspace.SourceConnections[index].Name)
		workspace.SourceConnections[index].Address = strings.TrimSpace(workspace.SourceConnections[index].Address)
		workspace.SourceConnections[index].CredentialRef = strings.TrimSpace(workspace.SourceConnections[index].CredentialRef)
	}
	workspace.TargetAssumptions.Address = strings.TrimSpace(workspace.TargetAssumptions.Address)
	workspace.TargetAssumptions.CredentialRef = strings.TrimSpace(workspace.TargetAssumptions.CredentialRef)
	workspace.TargetAssumptions.DefaultHost = strings.TrimSpace(workspace.TargetAssumptions.DefaultHost)
	workspace.TargetAssumptions.DefaultStorage = strings.TrimSpace(workspace.TargetAssumptions.DefaultStorage)
	workspace.TargetAssumptions.DefaultNetwork = strings.TrimSpace(workspace.TargetAssumptions.DefaultNetwork)
	workspace.TargetAssumptions.Notes = strings.TrimSpace(workspace.TargetAssumptions.Notes)
	workspace.PlanSettings.Name = strings.TrimSpace(workspace.PlanSettings.Name)
	workspace.PlanSettings.ApprovedBy = strings.TrimSpace(workspace.PlanSettings.ApprovedBy)
	workspace.PlanSettings.ApprovalTicket = strings.TrimSpace(workspace.PlanSettings.ApprovalTicket)
	for index := range workspace.Approvals {
		if strings.TrimSpace(workspace.Approvals[index].ID) == "" {
			workspace.Approvals[index].ID = uuid.NewString()
		}
		if workspace.Approvals[index].Status == "" {
			workspace.Approvals[index].Status = models.WorkspaceApprovalStatusPending
		}
		if workspace.Approvals[index].CreatedAt.IsZero() {
			workspace.Approvals[index].CreatedAt = workspace.UpdatedAt
		}
		workspace.Approvals[index].Stage = strings.TrimSpace(workspace.Approvals[index].Stage)
		workspace.Approvals[index].ApprovedBy = strings.TrimSpace(workspace.Approvals[index].ApprovedBy)
		workspace.Approvals[index].Ticket = strings.TrimSpace(workspace.Approvals[index].Ticket)
		workspace.Approvals[index].Notes = strings.TrimSpace(workspace.Approvals[index].Notes)
	}
	for index := range workspace.Notes {
		if strings.TrimSpace(workspace.Notes[index].ID) == "" {
			workspace.Notes[index].ID = uuid.NewString()
		}
		if workspace.Notes[index].Kind == "" {
			workspace.Notes[index].Kind = models.WorkspaceNoteKindOperator
		}
		if workspace.Notes[index].CreatedAt.IsZero() {
			workspace.Notes[index].CreatedAt = workspace.UpdatedAt
		}
		workspace.Notes[index].Author = strings.TrimSpace(workspace.Notes[index].Author)
		workspace.Notes[index].Body = strings.TrimSpace(workspace.Notes[index].Body)
	}
	for index := range workspace.Reports {
		if strings.TrimSpace(workspace.Reports[index].ID) == "" {
			workspace.Reports[index].ID = uuid.NewString()
		}
		if workspace.Reports[index].ExportedAt.IsZero() {
			workspace.Reports[index].ExportedAt = workspace.UpdatedAt
		}
		workspace.Reports[index].Name = strings.TrimSpace(workspace.Reports[index].Name)
		workspace.Reports[index].Format = strings.TrimSpace(workspace.Reports[index].Format)
		workspace.Reports[index].FileName = strings.TrimSpace(workspace.Reports[index].FileName)
		workspace.Reports[index].CorrelationID = strings.TrimSpace(workspace.Reports[index].CorrelationID)
	}
	if workspace.Simulation != nil {
		workspace.Simulation.SelectedWorkloadIDs = copyStringSlice(workspace.Simulation.SelectedWorkloadIDs)
	}
	if workspace.SavedPlan != nil {
		workspace.SavedPlan.SelectedWorkloadIDs = copyStringSlice(workspace.SavedPlan.SelectedWorkloadIDs)
	}
	if workspace.Readiness != nil {
		workspace.Readiness.BlockingIssues = copyStringSlice(workspace.Readiness.BlockingIssues)
		workspace.Readiness.WarningIssues = copyStringSlice(workspace.Readiness.WarningIssues)
	}
	return workspace
}

func normalizeWorkspaceJob(tenantID string, job models.WorkspaceJob) models.WorkspaceJob {
	now := time.Now().UTC()
	job.TenantID = normalizeTenantID(tenantID)
	job.WorkspaceID = strings.TrimSpace(job.WorkspaceID)
	job.RequestedBy = strings.TrimSpace(job.RequestedBy)
	job.CorrelationID = strings.TrimSpace(job.CorrelationID)
	job.Message = strings.TrimSpace(job.Message)
	job.Error = strings.TrimSpace(job.Error)
	if job.Status == "" {
		job.Status = models.WorkspaceJobStatusQueued
	}
	if job.RequestedAt.IsZero() {
		job.RequestedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.RequestedAt
	}
	return job
}

func defaultTenant() models.Tenant {
	return models.Tenant{
		ID:        DefaultTenantID,
		Name:      defaultTenantName,
		CreatedAt: time.Unix(0, 0).UTC(),
		Active:    true,
		Settings:  map[string]string{},
	}
}

func tenantRecoveryPointKey(tenantID, migrationID string) string {
	return tenantID + ":" + migrationID
}

func tenantMigrationKey(tenantID, migrationID string) string {
	return tenantID + ":" + migrationID
}

func tenantWorkspaceKey(tenantID, workspaceID string) string {
	return tenantID + ":" + workspaceID
}

func tenantWorkspaceJobKey(tenantID, workspaceID, jobID string) string {
	return tenantID + ":" + workspaceID + ":" + jobID
}

func copyStringSlice(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	return append([]string(nil), input...)
}
