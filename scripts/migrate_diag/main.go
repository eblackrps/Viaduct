package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type tenantMigrationStatus struct {
	TenantID                   string
	HashRowsPresent            bool
	TenantPlaintextCleared     bool
	ServiceAccountPlainCleared bool
}

type persistedServiceAccount struct {
	APIKey string `json:"api_key,omitempty"`
}

func main() {
	var dsn string
	flag.StringVar(&dsn, "dsn", strings.TrimSpace(os.Getenv("STATE_STORE_DSN")), "PostgreSQL DSN for the Viaduct state store")
	flag.Parse()

	if strings.TrimSpace(dsn) == "" {
		fmt.Fprintln(os.Stderr, "migrate-diag: state store DSN is required via -dsn or STATE_STORE_DSN")
		os.Exit(1)
	}

	if err := run(context.Background(), dsn, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dsn string, stdout *os.File) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("migrate-diag: open postgres store: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("migrate-diag: ping postgres store: %w", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT id, api_key, service_accounts FROM tenants ORDER BY id ASC`)
	if err != nil {
		return fmt.Errorf("migrate-diag: query tenants: %w", err)
	}
	defer rows.Close()

	statuses := make([]tenantMigrationStatus, 0)
	for rows.Next() {
		var (
			tenantID               string
			tenantAPIKey           string
			serviceAccountsPayload []byte
		)
		if err := rows.Scan(&tenantID, &tenantAPIKey, &serviceAccountsPayload); err != nil {
			return fmt.Errorf("migrate-diag: scan tenant row: %w", err)
		}
		status, err := loadTenantMigrationStatus(ctx, db, tenantID, tenantAPIKey, serviceAccountsPayload)
		if err != nil {
			return err
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate-diag: iterate tenant rows: %w", err)
	}

	fmt.Fprintln(stdout, "tenant_id\thash_rows_present\ttenant_plaintext_cleared\tservice_account_plaintext_cleared")
	for _, status := range statuses {
		fmt.Fprintf(
			stdout,
			"%s\t%t\t%t\t%t\n",
			status.TenantID,
			status.HashRowsPresent,
			status.TenantPlaintextCleared,
			status.ServiceAccountPlainCleared,
		)
	}
	return nil
}

func loadTenantMigrationStatus(ctx context.Context, db *sql.DB, tenantID, tenantAPIKey string, serviceAccountsPayload []byte) (tenantMigrationStatus, error) {
	status := tenantMigrationStatus{
		TenantID:               strings.TrimSpace(tenantID),
		TenantPlaintextCleared: strings.TrimSpace(tenantAPIKey) == "",
	}

	var hashCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM credential_hashes WHERE tenant_id = $1`, tenantID).Scan(&hashCount); err != nil {
		return tenantMigrationStatus{}, fmt.Errorf("migrate-diag: count credential hashes for tenant %s: %w", tenantID, err)
	}
	status.HashRowsPresent = hashCount > 0

	status.ServiceAccountPlainCleared = true
	if len(serviceAccountsPayload) > 0 {
		var accounts []persistedServiceAccount
		if err := json.Unmarshal(serviceAccountsPayload, &accounts); err != nil {
			return tenantMigrationStatus{}, fmt.Errorf("migrate-diag: decode service accounts for tenant %s: %w", tenantID, err)
		}
		for _, account := range accounts {
			if strings.TrimSpace(account.APIKey) != "" {
				status.ServiceAccountPlainCleared = false
				break
			}
		}
	}

	return status, nil
}
