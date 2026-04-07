package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestMemoryStore_SaveAndRetrieve_DefaultTenant(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	result := &models.DiscoveryResult{
		Source:       "lab-vcenter",
		Platform:     models.PlatformVMware,
		VMs:          []models.VirtualMachine{{Name: "web-01", Platform: models.PlatformVMware, PowerState: models.PowerOn}},
		DiscoveredAt: time.Date(2026, time.April, 3, 12, 0, 0, 0, time.UTC),
	}

	snapshotID, err := stateStore.SaveDiscovery(context.Background(), DefaultTenantID, result)
	if err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	got, err := stateStore.GetSnapshot(context.Background(), DefaultTenantID, snapshotID)
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}

	if got.Source != result.Source || len(got.VMs) != 1 || got.VMs[0].Name != "web-01" {
		t.Fatalf("unexpected snapshot contents: %#v", got)
	}
}

func TestMemoryStore_ListSnapshots_TenantScoped(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	_, _ = stateStore.SaveDiscovery(ctx, DefaultTenantID, &models.DiscoveryResult{
		Source:       "vcenter",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Date(2026, time.April, 3, 12, 0, 0, 0, time.UTC),
	})
	_, _ = stateStore.SaveDiscovery(ctx, "tenant-a", &models.DiscoveryResult{
		Source:       "proxmox",
		Platform:     models.PlatformProxmox,
		DiscoveredAt: time.Date(2026, time.April, 3, 13, 0, 0, 0, time.UTC),
	})

	items, err := stateStore.ListSnapshots(ctx, "tenant-a", models.PlatformProxmox, 10)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}

	if len(items) != 1 || items[0].Platform != models.PlatformProxmox || items[0].TenantID != "tenant-a" {
		t.Fatalf("unexpected snapshot metadata: %#v", items)
	}
}

func TestMemoryStore_QueryVMs_TenantScoped(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	_, _ = stateStore.SaveDiscovery(ctx, "tenant-a", &models.DiscoveryResult{
		Source:   "mixed",
		Platform: models.PlatformProxmox,
		VMs: []models.VirtualMachine{
			{Name: "web-01", Platform: models.PlatformProxmox, PowerState: models.PowerOn},
			{Name: "db-01", Platform: models.PlatformProxmox, PowerState: models.PowerOff},
		},
		DiscoveredAt: time.Now().UTC(),
	})
	_, _ = stateStore.SaveDiscovery(ctx, DefaultTenantID, &models.DiscoveryResult{
		Source:   "default",
		Platform: models.PlatformProxmox,
		VMs: []models.VirtualMachine{
			{Name: "web-default", Platform: models.PlatformProxmox, PowerState: models.PowerOn},
		},
		DiscoveredAt: time.Now().UTC(),
	})

	items, err := stateStore.QueryVMs(ctx, "tenant-a", VMFilter{
		Platform:     models.PlatformProxmox,
		PowerState:   models.PowerOn,
		NameContains: "web",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("QueryVMs() error = %v", err)
	}

	if len(items) != 1 || items[0].Name != "web-01" {
		t.Fatalf("unexpected query results: %#v", items)
	}
}

func TestMemoryStore_SaveAndListMigration_TenantScoped(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	record := MigrationRecord{
		ID:        "mig-001",
		SpecName:  "phase3-test",
		Phase:     "plan",
		StartedAt: time.Date(2026, time.April, 3, 18, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.April, 3, 18, 1, 0, 0, time.UTC),
		RawJSON:   json.RawMessage(`{"id":"mig-001","phase":"plan"}`),
	}

	if err := stateStore.SaveMigration(ctx, "tenant-a", record); err != nil {
		t.Fatalf("SaveMigration() error = %v", err)
	}

	got, err := stateStore.GetMigration(ctx, "tenant-a", "mig-001")
	if err != nil {
		t.Fatalf("GetMigration() error = %v", err)
	}

	if got.Phase != "plan" || got.SpecName != "phase3-test" || got.TenantID != "tenant-a" {
		t.Fatalf("unexpected migration record: %#v", got)
	}

	items, err := stateStore.ListMigrations(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}

	if len(items) != 1 || items[0].ID != "mig-001" || items[0].TenantID != "tenant-a" {
		t.Fatalf("unexpected migration metadata: %#v", items)
	}
}

func TestMemoryStore_SaveMigration_SameIDAcrossTenants_Isolated(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	for _, tenant := range []models.Tenant{
		{
			ID:        "tenant-a",
			Name:      "Tenant A",
			APIKey:    "tenant-a-key",
			CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
			Active:    true,
		},
		{
			ID:        "tenant-b",
			Name:      "Tenant B",
			APIKey:    "tenant-b-key",
			CreatedAt: time.Date(2026, time.April, 4, 8, 1, 0, 0, time.UTC),
			Active:    true,
		},
	} {
		if err := stateStore.CreateTenant(ctx, tenant); err != nil {
			t.Fatalf("CreateTenant(%s) error = %v", tenant.ID, err)
		}
	}

	for tenantID, phase := range map[string]string{
		"tenant-a": "plan",
		"tenant-b": "verify",
	} {
		if err := stateStore.SaveMigration(ctx, tenantID, MigrationRecord{
			ID:        "shared-migration-id",
			SpecName:  tenantID + "-spec",
			Phase:     phase,
			StartedAt: time.Date(2026, time.April, 4, 9, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, time.April, 4, 9, 1, 0, 0, time.UTC),
			RawJSON:   json.RawMessage(`{"id":"shared-migration-id"}`),
		}); err != nil {
			t.Fatalf("SaveMigration(%s) error = %v", tenantID, err)
		}
	}

	recordA, err := stateStore.GetMigration(ctx, "tenant-a", "shared-migration-id")
	if err != nil {
		t.Fatalf("GetMigration(tenant-a) error = %v", err)
	}
	recordB, err := stateStore.GetMigration(ctx, "tenant-b", "shared-migration-id")
	if err != nil {
		t.Fatalf("GetMigration(tenant-b) error = %v", err)
	}

	if recordA.SpecName != "tenant-a-spec" || recordA.TenantID != "tenant-a" {
		t.Fatalf("unexpected tenant-a migration record: %#v", recordA)
	}
	if recordB.SpecName != "tenant-b-spec" || recordB.TenantID != "tenant-b" {
		t.Fatalf("unexpected tenant-b migration record: %#v", recordB)
	}
}

func TestMemoryStore_SaveAndGetRecoveryPoint_TenantScoped(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	record := RecoveryPointRecord{
		MigrationID: "mig-rollback",
		Phase:       "convert",
		CreatedAt:   time.Date(2026, time.April, 3, 18, 2, 0, 0, time.UTC),
		RawJSON:     json.RawMessage(`{"migration_id":"mig-rollback","phase":"convert"}`),
	}

	if err := stateStore.SaveRecoveryPoint(ctx, "tenant-a", record); err != nil {
		t.Fatalf("SaveRecoveryPoint() error = %v", err)
	}

	got, err := stateStore.GetRecoveryPoint(ctx, "tenant-a", "mig-rollback")
	if err != nil {
		t.Fatalf("GetRecoveryPoint() error = %v", err)
	}

	if got.Phase != "convert" || got.TenantID != "tenant-a" {
		t.Fatalf("unexpected recovery point: %#v", got)
	}
}

func TestMemoryStore_CreateAndDeleteTenant_RemovesScopedData(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	tenant := models.Tenant{
		ID:        "tenant-delete",
		Name:      "Tenant Delete",
		APIKey:    "tenant-delete-key",
		CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, time.UTC),
		Active:    true,
		Settings:  map[string]string{"region": "east"},
	}

	if err := stateStore.CreateTenant(ctx, tenant); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	if _, err := stateStore.SaveDiscovery(ctx, tenant.ID, &models.DiscoveryResult{
		Source:       "tenant-delete",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	items, err := stateStore.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants() error = %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("expected tenant list to include default and created tenant, got %#v", items)
	}

	if err := stateStore.DeleteTenant(ctx, tenant.ID); err != nil {
		t.Fatalf("DeleteTenant() error = %v", err)
	}

	if _, err := stateStore.GetTenant(ctx, tenant.ID); err == nil {
		t.Fatal("GetTenant() error = nil, want not found")
	}
}

func TestMemoryStore_SaveAndListAuditEvents_TenantScoped(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:        "tenant-a",
		Name:      "Tenant A",
		APIKey:    "tenant-a-key",
		CreatedAt: time.Date(2026, time.April, 5, 8, 0, 0, 0, time.UTC),
		Active:    true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	for _, event := range []models.AuditEvent{
		{
			ID:        "event-a",
			TenantID:  "tenant-a",
			Actor:     "tenant:tenant-a",
			Category:  "migration",
			Action:    "execute",
			Resource:  "migration-1",
			Outcome:   models.AuditOutcomeSuccess,
			Message:   "migration started",
			CreatedAt: time.Date(2026, time.April, 5, 9, 0, 0, 0, time.UTC),
		},
		{
			ID:        "event-b",
			TenantID:  DefaultTenantID,
			Actor:     "tenant:default",
			Category:  "admin",
			Action:    "read",
			Resource:  "summary",
			Outcome:   models.AuditOutcomeSuccess,
			Message:   "summary read",
			CreatedAt: time.Date(2026, time.April, 5, 8, 0, 0, 0, time.UTC),
		},
	} {
		if err := stateStore.SaveAuditEvent(ctx, event); err != nil {
			t.Fatalf("SaveAuditEvent(%s) error = %v", event.ID, err)
		}
	}

	items, err := stateStore.ListAuditEvents(ctx, "tenant-a", 10)
	if err != nil {
		t.Fatalf("ListAuditEvents() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "event-a" || items[0].TenantID != "tenant-a" {
		t.Fatalf("unexpected audit event: %#v", items[0])
	}
}

func TestMemoryStore_SaveDiscovery_SnapshotQuotaExceeded_ReturnsError(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:     "tenant-quota",
		Name:   "Quota Tenant",
		APIKey: "tenant-quota-key",
		Active: true,
		Quotas: models.TenantQuota{
			MaxSnapshots: 1,
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	for index := 0; index < 2; index++ {
		_, err := stateStore.SaveDiscovery(ctx, "tenant-quota", &models.DiscoveryResult{
			Source:       "source",
			Platform:     models.PlatformVMware,
			DiscoveredAt: time.Now().UTC(),
		})
		if index == 0 && err != nil {
			t.Fatalf("SaveDiscovery(first) error = %v", err)
		}
		if index == 1 && err == nil {
			t.Fatal("SaveDiscovery(second) error = nil, want quota exceeded")
		}
	}
}

func TestMemoryStore_SaveMigration_MigrationQuotaExceeded_ReturnsError(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	if err := stateStore.CreateTenant(ctx, models.Tenant{
		ID:     "tenant-migration-quota",
		Name:   "Migration Quota Tenant",
		APIKey: "tenant-migration-quota-key",
		Active: true,
		Quotas: models.TenantQuota{
			MaxMigrations: 1,
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	first := MigrationRecord{
		ID:        "migration-1",
		SpecName:  "quota-test",
		Phase:     "plan",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		RawJSON:   json.RawMessage(`{"id":"migration-1"}`),
	}
	if err := stateStore.SaveMigration(ctx, "tenant-migration-quota", first); err != nil {
		t.Fatalf("SaveMigration(first) error = %v", err)
	}
	if err := stateStore.SaveMigration(ctx, "tenant-migration-quota", MigrationRecord{
		ID:        "migration-2",
		SpecName:  "quota-test",
		Phase:     "plan",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		RawJSON:   json.RawMessage(`{"id":"migration-2"}`),
	}); err == nil {
		t.Fatal("SaveMigration(second) error = nil, want quota exceeded")
	}
}

func TestMemoryStore_UpdateTenant_ServiceAccountsPersisted_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	ctx := context.Background()
	tenant := models.Tenant{
		ID:     "tenant-service-accounts",
		Name:   "Service Account Tenant",
		APIKey: "tenant-service-accounts-key",
		Active: true,
	}
	if err := stateStore.CreateTenant(ctx, tenant); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	tenant.ServiceAccounts = []models.ServiceAccount{
		{
			ID:     "sa-1",
			Name:   "Read Only",
			APIKey: "sa-1-key",
			Role:   models.TenantRoleViewer,
			Permissions: []models.TenantPermission{
				models.TenantPermissionInventoryRead,
			},
			Active:    true,
			CreatedAt: time.Date(2026, time.April, 7, 10, 0, 0, 0, time.UTC),
			Metadata:  map[string]string{"owner": "ops"},
		},
	}
	tenant.Quotas = models.TenantQuota{RequestsPerMinute: 120}
	if err := stateStore.UpdateTenant(ctx, tenant); err != nil {
		t.Fatalf("UpdateTenant() error = %v", err)
	}

	persisted, err := stateStore.GetTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	if persisted.Quotas.RequestsPerMinute != 120 {
		t.Fatalf("RequestsPerMinute = %d, want 120", persisted.Quotas.RequestsPerMinute)
	}
	if len(persisted.ServiceAccounts) != 1 || persisted.ServiceAccounts[0].ID != "sa-1" {
		t.Fatalf("unexpected service accounts: %#v", persisted.ServiceAccounts)
	}
	if persisted.ServiceAccounts[0].Metadata["owner"] != "ops" {
		t.Fatalf("unexpected service-account metadata: %#v", persisted.ServiceAccounts[0].Metadata)
	}
	if len(persisted.ServiceAccounts[0].Permissions) != 1 || persisted.ServiceAccounts[0].Permissions[0] != models.TenantPermissionInventoryRead {
		t.Fatalf("unexpected service-account permissions: %#v", persisted.ServiceAccounts[0].Permissions)
	}
}

func TestMemoryStore_Diagnostics_ReturnsBackendMetadata_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	diagnostics, err := stateStore.Diagnostics(context.Background())
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.Backend != "memory" || diagnostics.Persistent {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestPostgresSchemaHistory_CurrentVersionSynchronized_Expected(t *testing.T) {
	t.Parallel()

	if len(storeSchemaHistory) == 0 {
		t.Fatal("storeSchemaHistory is empty")
	}
	if storeSchemaHistory[len(storeSchemaHistory)-1].version != currentStoreSchemaVersion {
		t.Fatalf("latest schema version = %d, want %d", storeSchemaHistory[len(storeSchemaHistory)-1].version, currentStoreSchemaVersion)
	}
	if !strings.Contains(createStoreSchemaSQL, "schema_migrations") {
		t.Fatalf("createStoreSchemaSQL missing schema_migrations table: %s", createStoreSchemaSQL)
	}
}
