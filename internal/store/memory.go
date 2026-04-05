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

// MemoryStore is an in-memory Store implementation for testing and small deployments.
type MemoryStore struct {
	mu             sync.RWMutex
	tenants        map[string]models.Tenant
	snapshots      map[string]storedSnapshot
	migrations     map[string]storedMigration
	recoveryPoints map[string]storedRecoveryPoint
}

// NewMemoryStore creates an empty in-memory discovery snapshot store.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		tenants:        make(map[string]models.Tenant),
		snapshots:      make(map[string]storedSnapshot),
		migrations:     make(map[string]storedMigration),
		recoveryPoints: make(map[string]storedRecoveryPoint),
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

	if err := s.ensureTenantLocked(tenantID); err != nil {
		return "", fmt.Errorf("memory store: save discovery: %w", err)
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

	if err := s.ensureTenantLocked(tenantID); err != nil {
		return fmt.Errorf("memory store: save migration: %w", err)
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

	if err := s.ensureTenantLocked(tenantID); err != nil {
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
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = time.Now().UTC()
	}
	if tenant.Settings == nil {
		tenant.Settings = make(map[string]string)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.tenants[tenant.ID]; ok && existing.ID != "" {
		return fmt.Errorf("memory store: create tenant %s: already exists", tenant.ID)
	}

	s.tenants[tenant.ID] = tenant
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

	cloned := tenant
	cloned.Settings = copyStringMap(tenant.Settings)
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
		cloned := tenant
		cloned.Settings = copyStringMap(tenant.Settings)
		items = append(items, cloned)
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

	return nil
}

// Close releases resources held by the in-memory store.
func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) ensureTenantLocked(tenantID string) error {
	if _, ok := s.tenants[tenantID]; ok {
		return nil
	}
	if tenantID == DefaultTenantID {
		s.tenants[tenantID] = defaultTenant()
		return nil
	}

	return fmt.Errorf("tenant %s not found", tenantID)
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
