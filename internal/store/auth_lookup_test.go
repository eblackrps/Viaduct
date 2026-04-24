package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestMemoryStore_LookupCredentialByHash_ServiceAccount_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-a-key",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-operator",
				Name:      "Operator",
				APIKey:    "service-key",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	result, err := stateStore.LookupCredentialByHash(context.Background(), HashAPIKey("service-key"))
	if err != nil {
		t.Fatalf("LookupCredentialByHash() error = %v", err)
	}
	if result == nil || result.ServiceAccount == nil {
		t.Fatalf("LookupCredentialByHash() = %#v, want service account result", result)
	}
	if result.Tenant.ID != "tenant-a" || result.ServiceAccount.ID != "sa-operator" {
		t.Fatalf("unexpected credential lookup result: %#v", result)
	}
}

func TestPostgresStore_LookupCredentialByHash_UsesCredentialIndex_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	stateStore := &PostgresStore{db: db}
	createdAt := time.Date(2026, time.April, 24, 17, 0, 0, 0, time.UTC)
	serviceAccountsPayload, err := marshalServiceAccounts([]models.ServiceAccount{
		{
			ID:         "sa-operator",
			Name:       "Operator",
			APIKeyHash: HashAPIKey("service-key"),
			Role:       models.TenantRoleOperator,
			Active:     true,
			CreatedAt:  createdAt,
		},
	})
	if err != nil {
		t.Fatalf("Marshal(service accounts) error = %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT tenant_id, owner_type, service_account_id
		   FROM credential_hashes
		  WHERE credential_hash = $1
		  LIMIT 1`)).
		WithArgs(HashAPIKey("service-key")).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "owner_type", "service_account_id"}).
			AddRow("tenant-a", "service_account", "sa-operator"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants WHERE id = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "api_key", "api_key_hash", "created_at", "active", "settings", "quotas", "service_accounts"}).
			AddRow("tenant-a", "Tenant A", "", HashAPIKey("tenant-a-key"), createdAt, true, []byte(`{}`), []byte(`{}`), serviceAccountsPayload))

	result, err := stateStore.LookupCredentialByHash(context.Background(), HashAPIKey("service-key"))
	if err != nil {
		t.Fatalf("LookupCredentialByHash() error = %v", err)
	}
	if result == nil || result.ServiceAccount == nil {
		t.Fatalf("LookupCredentialByHash() = %#v, want service account result", result)
	}
	if result.Tenant.ID != "tenant-a" || result.ServiceAccount.ID != "sa-operator" {
		t.Fatalf("unexpected credential lookup result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet() error = %v", err)
	}
}
