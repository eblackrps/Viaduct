package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestPostgresStore_QueryVMs_UsesCurrentSnapshotQuery_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	payload, err := json.Marshal(models.DiscoveryResult{
		Source:   "vcenter",
		Platform: models.PlatformVMware,
		VMs: []models.VirtualMachine{
			{Name: "web-current", Platform: models.PlatformVMware, PowerState: models.PowerOn},
			{Name: "db-off", Platform: models.PlatformVMware, PowerState: models.PowerOff},
		},
		DiscoveredAt: time.Date(2026, time.April, 3, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	mock.ExpectQuery(`SELECT raw_json\s+FROM \(\s+SELECT DISTINCT ON \(lower\(btrim\(source\)\), platform\)`).
		WithArgs(DefaultTenantID, string(models.PlatformVMware)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_json"}).AddRow(payload))

	stateStore := &PostgresStore{db: db}
	items, err := stateStore.QueryVMs(context.Background(), DefaultTenantID, VMFilter{
		Platform:   models.PlatformVMware,
		PowerState: models.PowerOn,
	})
	if err != nil {
		t.Fatalf("QueryVMs() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "web-current" {
		t.Fatalf("QueryVMs() = %#v, want filtered current VM", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet() error = %v", err)
	}
}
