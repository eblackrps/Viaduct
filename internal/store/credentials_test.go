package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/lib/pq"
)

var credentialHashRaceDriverCounter uint64

func TestPrepareTenantCredentials_HashesAndRedactsWithoutMutatingCaller_Expected(t *testing.T) {
	t.Parallel()

	original := models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-secret",
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:     "sa-1",
				Name:   "Automation",
				APIKey: "service-secret",
				Role:   models.TenantRoleOperator,
				Active: true,
			},
		},
	}

	normalized, err := prepareTenantCredentials(original)
	if err != nil {
		t.Fatalf("prepareTenantCredentials() error = %v", err)
	}

	if normalized.APIKey != "" {
		t.Fatalf("normalized.APIKey = %q, want redacted", normalized.APIKey)
	}
	if normalized.APIKeyHash != HashAPIKey("tenant-secret") {
		t.Fatalf("normalized.APIKeyHash = %q, want %q", normalized.APIKeyHash, HashAPIKey("tenant-secret"))
	}
	if normalized.ServiceAccounts[0].APIKey != "" {
		t.Fatalf("normalized.ServiceAccounts[0].APIKey = %q, want redacted", normalized.ServiceAccounts[0].APIKey)
	}
	if normalized.ServiceAccounts[0].APIKeyHash != HashAPIKey("service-secret") {
		t.Fatalf("normalized.ServiceAccounts[0].APIKeyHash = %q, want %q", normalized.ServiceAccounts[0].APIKeyHash, HashAPIKey("service-secret"))
	}

	if original.APIKey != "tenant-secret" {
		t.Fatalf("original.APIKey = %q, want caller value preserved", original.APIKey)
	}
	if original.ServiceAccounts[0].APIKey != "service-secret" {
		t.Fatalf("original.ServiceAccounts[0].APIKey = %q, want caller value preserved", original.ServiceAccounts[0].APIKey)
	}
}

func TestAPIKeyMatches_HashedAndLegacyValues_Expected(t *testing.T) {
	t.Parallel()

	hash := HashAPIKey("tenant-secret")
	if !APIKeyMatches("tenant-secret", hash, "") {
		t.Fatal("APIKeyMatches(hashed) = false, want true")
	}
	if APIKeyMatches("different-secret", hash, "tenant-secret") {
		t.Fatal("APIKeyMatches(hashed-wrong) = true, want false")
	}
	if !APIKeyMatches("legacy-secret", "", "legacy-secret") {
		t.Fatal("APIKeyMatches(legacy) = false, want true")
	}
	if APIKeyMatches("legacy-secret", "", "different") {
		t.Fatal("APIKeyMatches(legacy-wrong) = true, want false")
	}
}

func TestMemoryStore_CreateTenant_PersistsOnlyCredentialHashes_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	tenant := models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-secret",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-1",
				Name:      "Automation",
				APIKey:    "service-secret",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}

	if err := stateStore.CreateTenant(context.Background(), tenant); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	persisted, err := stateStore.GetTenant(context.Background(), tenant.ID)
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}

	if persisted.APIKey != "" {
		t.Fatalf("persisted.APIKey = %q, want redacted", persisted.APIKey)
	}
	if persisted.APIKeyHash != HashAPIKey("tenant-secret") {
		t.Fatalf("persisted.APIKeyHash = %q, want %q", persisted.APIKeyHash, HashAPIKey("tenant-secret"))
	}
	if len(persisted.ServiceAccounts) != 1 {
		t.Fatalf("len(persisted.ServiceAccounts) = %d, want 1", len(persisted.ServiceAccounts))
	}
	if persisted.ServiceAccounts[0].APIKey != "" {
		t.Fatalf("persisted.ServiceAccounts[0].APIKey = %q, want redacted", persisted.ServiceAccounts[0].APIKey)
	}
	if persisted.ServiceAccounts[0].APIKeyHash != HashAPIKey("service-secret") {
		t.Fatalf("persisted.ServiceAccounts[0].APIKeyHash = %q, want %q", persisted.ServiceAccounts[0].APIKeyHash, HashAPIKey("service-secret"))
	}
	if !APIKeyMatches("tenant-secret", persisted.APIKeyHash, persisted.APIKey) {
		t.Fatal("persisted tenant credential no longer authenticates")
	}
	if !APIKeyMatches("service-secret", persisted.ServiceAccounts[0].APIKeyHash, persisted.ServiceAccounts[0].APIKey) {
		t.Fatal("persisted service-account credential no longer authenticates")
	}
}

func TestMemoryStore_CreateTenant_GlobalCredentialUniqueness_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		Active: true,
		ServiceAccounts: []models.ServiceAccount{
			{
				ID:        "sa-1",
				Name:      "Automation",
				APIKey:    "shared-secret",
				Role:      models.TenantRoleOperator,
				Active:    true,
				CreatedAt: time.Now().UTC(),
			},
		},
	}); err != nil {
		t.Fatalf("CreateTenant(tenant-a) error = %v", err)
	}

	err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-b",
		Name:   "Tenant B",
		APIKey: "shared-secret",
		Active: true,
	})
	if !IsCredentialConflict(err) {
		t.Fatalf("CreateTenant(tenant-b) error = %v, want credential conflict", err)
	}
}

func TestMemoryStore_UpdateTenant_DuplicateCredentialWithinTenantRejected_Expected(t *testing.T) {
	t.Parallel()

	stateStore := NewMemoryStore()
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-a",
		Name:   "Tenant A",
		APIKey: "tenant-secret",
		Active: true,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}

	tenant, err := stateStore.GetTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("GetTenant() error = %v", err)
	}
	tenant.ServiceAccounts = append(tenant.ServiceAccounts, models.ServiceAccount{
		ID:        "sa-1",
		Name:      "Automation",
		APIKey:    "tenant-secret",
		Role:      models.TenantRoleOperator,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	})

	err = stateStore.UpdateTenant(context.Background(), *tenant)
	if !IsCredentialConflict(err) {
		t.Fatalf("UpdateTenant() error = %v, want credential conflict", err)
	}
}

func TestMigrateStoredCredentials_RewritesLegacyPlaintext_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	mock.MatchExpectationsInOrder(false)

	createdAt := time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)
	serviceAccountsPayload := []byte(`[{"id":"sa-1","name":"Automation","api_key":"service-secret","role":"operator","active":true,"created_at":"2026-04-17T12:00:00Z"}]`)
	rows := sqlmock.NewRows([]string{"id", "name", "api_key", "api_key_hash", "created_at", "active", "settings", "quotas", "service_accounts"}).
		AddRow("tenant-a", "Tenant A", "tenant-secret", "", createdAt, true, []byte(`{"region":"lab"}`), []byte(`{"requests_per_minute":10}`), serviceAccountsPayload)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants ORDER BY created_at ASC`)).
		WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (credential_hash) DO NOTHING`)).
		WithArgs(HashAPIKey("tenant-secret"), "tenant-a", "tenant", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (credential_hash) DO NOTHING`)).
		WithArgs(HashAPIKey("service-secret"), "tenant-a", "service_account", "sa-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
				SELECT 1
				FROM credential_hashes
				WHERE credential_hash = $1
				  AND tenant_id = $2
				  AND owner_type = $3
				  AND service_account_id = $4
			)`)).
		WithArgs(HashAPIKey("tenant-secret"), "tenant-a", "tenant", "").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
				SELECT 1
				FROM credential_hashes
				WHERE credential_hash = $1
				  AND tenant_id = $2
				  AND owner_type = $3
				  AND service_account_id = $4
			)`)).
		WithArgs(HashAPIKey("service-secret"), "tenant-a", "service_account", "sa-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants SET api_key = $2, api_key_hash = $3, service_accounts = $4 WHERE id = $1`)).
		WithArgs("tenant-a", "", HashAPIKey("tenant-secret"), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := migrateStoredCredentials(context.Background(), db); err != nil {
		t.Fatalf("migrateStoredCredentials() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}

func TestMigrateStoredCredentials_InsertFailureRollsBackBeforeClearingPlaintext_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()
	mock.MatchExpectationsInOrder(false)

	createdAt := time.Date(2026, time.April, 17, 12, 0, 0, 0, time.UTC)
	serviceAccountsPayload := []byte(`[{"id":"sa-1","name":"Automation","api_key":"service-secret","role":"operator","active":true,"created_at":"2026-04-17T12:00:00Z"}]`)
	rows := sqlmock.NewRows([]string{"id", "name", "api_key", "api_key_hash", "created_at", "active", "settings", "quotas", "service_accounts"}).
		AddRow("tenant-a", "Tenant A", "tenant-secret", "", createdAt, true, []byte(`{}`), []byte(`{}`), serviceAccountsPayload)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, api_key, api_key_hash, created_at, active, settings, quotas, service_accounts FROM tenants ORDER BY created_at ASC`)).
		WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (credential_hash) DO NOTHING`)).
		WithArgs(HashAPIKey("tenant-secret"), "tenant-a", "tenant", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO credential_hashes (credential_hash, tenant_id, owner_type, service_account_id)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (credential_hash) DO NOTHING`)).
		WithArgs(HashAPIKey("service-secret"), "tenant-a", "service_account", "sa-1").
		WillReturnError(fmt.Errorf("insert failure"))
	mock.ExpectRollback()

	if err := migrateStoredCredentials(context.Background(), db); err == nil {
		t.Fatal("migrateStoredCredentials() error = nil, want insert failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}

func TestClearStoredCredentialPlaintext_MissingHashPreconditionRollsBack_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	tenant := models.Tenant{
		ID:         "tenant-a",
		Name:       "Tenant A",
		APIKeyHash: HashAPIKey("tenant-secret"),
		Active:     true,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
				SELECT 1
				FROM credential_hashes
				WHERE credential_hash = $1
				  AND tenant_id = $2
				  AND owner_type = $3
				  AND service_account_id = $4
			)`)).
		WithArgs(HashAPIKey("tenant-secret"), "tenant-a", "tenant", "").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	err = clearStoredCredentialPlaintext(context.Background(), db, []models.Tenant{tenant})
	if err == nil {
		t.Fatal("clearStoredCredentialPlaintext() error = nil, want missing precondition")
	}
	if !strings.Contains(err.Error(), "docs/operations/credential-migration.md") {
		t.Fatalf("clearStoredCredentialPlaintext() error = %q, want remediation document path", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}

func TestInsertTenantCredentialHashes_ConcurrentDuplicate_ReturnsSingleConflict_Expected(t *testing.T) {
	t.Parallel()

	driverName := fmt.Sprintf("credential-hash-race-%d", atomic.AddUint64(&credentialHashRaceDriverCounter, 1))
	sql.Register(driverName, &credentialHashRaceDriver{
		state: &credentialHashRaceState{
			inserted: make(map[string]struct{}),
		},
	})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(2)

	tenant := models.Tenant{
		ID:         "tenant-a",
		Name:       "Tenant A",
		APIKeyHash: HashAPIKey("shared-secret"),
		Active:     true,
	}

	const workerCount = 2
	results := make(chan error, workerCount)
	var wg sync.WaitGroup
	wg.Add(workerCount)

	for index := 0; index < workerCount; index++ {
		go func() {
			defer wg.Done()

			tx, txErr := db.BeginTx(context.Background(), nil)
			if txErr != nil {
				results <- txErr
				return
			}

			if insertErr := insertTenantCredentialHashes(context.Background(), tx, tenant); insertErr != nil {
				// Rollback is best effort after the simulated unique-constraint failure.
				_ = tx.Rollback()
				results <- insertErr
				return
			}

			results <- tx.Commit()
		}()
	}

	wg.Wait()
	close(results)

	var successCount, conflictCount int
	for err := range results {
		switch {
		case err == nil:
			successCount++
		case IsCredentialConflict(err):
			conflictCount++
		default:
			t.Fatalf("insertTenantCredentialHashes() error = %v, want nil or credential conflict", err)
		}
	}

	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("successes/conflicts = %d/%d, want 1/1", successCount, conflictCount)
	}
}

func TestReadStoreMigrationFile_ReadsBundledSQLAndRejectsTraversal_Expected(t *testing.T) {
	t.Parallel()

	payload, err := readStoreMigrationFile("008_credential_hash_unique.sql")
	if err != nil {
		t.Fatalf("readStoreMigrationFile(valid) error = %v", err)
	}
	if !strings.Contains(payload, "CREATE UNIQUE INDEX CONCURRENTLY") {
		t.Fatalf("readStoreMigrationFile(valid) payload = %q, want migration SQL", payload)
	}

	if _, err := readStoreMigrationFile("../postgres.go"); err == nil {
		t.Fatal("readStoreMigrationFile(traversal) error = nil, want root-bounded rejection")
	}
}

func TestAnnotateCredentialMigrationError_DuplicateCredentialConflictActionable_Expected(t *testing.T) {
	t.Parallel()

	err := annotateCredentialMigrationError(&CredentialConflictError{
		Owners: []CredentialConflictOwner{
			{TenantID: "tenant-a"},
			{TenantID: "tenant-b", ServiceAccountID: "sa-1"},
		},
	})
	if err == nil {
		t.Fatal("annotateCredentialMigrationError() error = nil, want actionable message")
	}
	if !IsCredentialConflict(err) {
		t.Fatalf("annotateCredentialMigrationError() error = %v, want credential conflict", err)
	}
	message := err.Error()
	if !strings.Contains(message, "resolve duplicated tenant or service-account API keys before restarting") {
		t.Fatalf("annotateCredentialMigrationError() message = %q, want remediation guidance", message)
	}
	if !strings.Contains(message, "tenant-a") || !strings.Contains(message, "tenant-b") {
		t.Fatalf("annotateCredentialMigrationError() message = %q, want tenant identifiers", message)
	}
	if !strings.Contains(message, "docs/operations/credential-migration.md") {
		t.Fatalf("annotateCredentialMigrationError() message = %q, want remediation document path", message)
	}
}

func TestStoreMigrationSkipsTransaction_ConcurrentIndexPragma_Expected(t *testing.T) {
	t.Parallel()

	if !storeMigrationSkipsTransaction("-- MUST NOT be wrapped in a transaction; uses CREATE INDEX CONCURRENTLY.\nCREATE UNIQUE INDEX CONCURRENTLY foo ON bar (id);") {
		t.Fatal("storeMigrationSkipsTransaction() = false, want true for concurrent-index pragma")
	}
	if storeMigrationSkipsTransaction("CREATE TABLE example (id TEXT);") {
		t.Fatal("storeMigrationSkipsTransaction() = true, want false without pragma")
	}
}

func TestExecuteStoreMigration_SkipTransactionPragma_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	payload, err := readStoreMigrationFile("008_credential_hash_unique.sql")
	if err != nil {
		t.Fatalf("readStoreMigrationFile() error = %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(payload)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := executeStoreMigration(context.Background(), db, "008_credential_hash_unique.sql"); err != nil {
		t.Fatalf("executeStoreMigration() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}

func TestApplyCredentialHashUniqueIndexMigration_PreflightDuplicatesListsTenantIDs_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT tenant_id, service_account_id
		 FROM credential_hashes
		 WHERE credential_hash IN (
		 	SELECT credential_hash
		 	FROM credential_hashes
		 	GROUP BY credential_hash
		 	HAVING COUNT(*) > 1
		 )
		 ORDER BY tenant_id ASC, service_account_id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "service_account_id"}).
			AddRow("tenant-a", "").
			AddRow("tenant-b", "sa-1"))

	err = applyCredentialHashUniqueIndexMigration(context.Background(), db)
	if !IsCredentialConflict(err) {
		t.Fatalf("applyCredentialHashUniqueIndexMigration() error = %v, want credential conflict", err)
	}

	var conflictErr *CredentialConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("applyCredentialHashUniqueIndexMigration() error = %v, want credential conflict details", err)
	}
	if got := conflictErr.TenantIDs(); strings.Join(got, ",") != "tenant-a,tenant-b" {
		t.Fatalf("conflictErr.TenantIDs() = %v, want [tenant-a tenant-b]", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}

func TestCreateStoreSchemaSQL_CredentialHashesRegistryDurablyUnique_Expected(t *testing.T) {
	t.Parallel()

	if !strings.Contains(createStoreSchemaSQL, "CREATE TABLE IF NOT EXISTS credential_hashes") {
		t.Fatal("createStoreSchemaSQL is missing credential_hashes registry")
	}
	if !strings.Contains(createStoreSchemaSQL, "credential_hash TEXT PRIMARY KEY") {
		t.Fatal("createStoreSchemaSQL is missing durable credential hash primary key")
	}
}

type credentialHashRaceDriver struct {
	state *credentialHashRaceState
}

type credentialHashRaceState struct {
	mu       sync.Mutex
	inserted map[string]struct{}
}

type credentialHashRaceConn struct {
	state *credentialHashRaceState
}

type credentialHashRaceTx struct{}

type credentialHashRaceRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (d *credentialHashRaceDriver) Open(_ string) (driver.Conn, error) {
	return &credentialHashRaceConn{state: d.state}, nil
}

func (c *credentialHashRaceConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, fmt.Errorf("credential hash race driver does not support prepared statements")
}

func (c *credentialHashRaceConn) Close() error {
	return nil
}

func (c *credentialHashRaceConn) Begin() (driver.Tx, error) {
	return credentialHashRaceTx{}, nil
}

func (c *credentialHashRaceConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return credentialHashRaceTx{}, nil
}

func (c *credentialHashRaceConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "INSERT INTO credential_hashes") || !strings.Contains(query, "RETURNING credential_hash") {
		return nil, fmt.Errorf("credential hash race driver received unexpected query: %s", query)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("credential hash race driver requires the credential hash argument")
	}

	hash, ok := args[0].Value.(string)
	if !ok {
		return nil, fmt.Errorf("credential hash race driver hash arg type = %T, want string", args[0].Value)
	}

	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	if _, exists := c.state.inserted[hash]; exists {
		return nil, &pq.Error{
			Code:       "23505",
			Table:      "credential_hashes",
			Constraint: "credential_hashes_hash_unique",
		}
	}
	c.state.inserted[hash] = struct{}{}

	return &credentialHashRaceRows{
		columns: []string{"credential_hash"},
		values:  [][]driver.Value{{hash}},
	}, nil
}

func (credentialHashRaceTx) Commit() error {
	return nil
}

func (credentialHashRaceTx) Rollback() error {
	return nil
}

func (r *credentialHashRaceRows) Columns() []string {
	return r.columns
}

func (r *credentialHashRaceRows) Close() error {
	return nil
}

func (r *credentialHashRaceRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}
