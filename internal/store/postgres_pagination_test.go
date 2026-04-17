package store

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestNormalizePageRequest_DefaultsClampAndRejects_Expected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        int
		perPage     int
		wantPage    int
		wantPerPage int
		wantErr     bool
	}{
		{
			name:        "defaults zero page and per-page",
			page:        0,
			perPage:     0,
			wantPage:    1,
			wantPerPage: 50,
		},
		{
			name:        "clamps oversized per-page",
			page:        1,
			perPage:     500,
			wantPage:    1,
			wantPerPage: 200,
		},
		{
			name:    "rejects negative page",
			page:    -1,
			perPage: 25,
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			page, perPage, err := normalizePageRequest(test.page, test.perPage)
			if test.wantErr {
				if err == nil {
					t.Fatal("normalizePageRequest() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizePageRequest() error = %v", err)
			}
			if page != test.wantPage {
				t.Fatalf("page = %d, want %d", page, test.wantPage)
			}
			if perPage != test.wantPerPage {
				t.Fatalf("perPage = %d, want %d", perPage, test.wantPerPage)
			}
		})
	}
}

func TestPostgresStore_ListSnapshotsPage_FilterUsesPlaceholders_Expected(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	stateStore := &PostgresStore{db: db}
	filter := models.Platform(`' OR 1=1 --`)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM snapshots WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)`)).
		WithArgs(DefaultTenantID, string(filter)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, tenant_id, source, platform, vm_count, discovered_at
		 FROM snapshots
		 WHERE tenant_id = $1 AND ($2 = '' OR platform = $2)
		 ORDER BY discovered_at DESC
		 LIMIT $3 OFFSET $4`)).
		WithArgs(DefaultTenantID, string(filter), 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "source", "platform", "vm_count", "discovered_at"}))

	items, total, err := stateStore.ListSnapshotsPage(context.Background(), DefaultTenantID, filter, 0, 0)
	if err != nil {
		t.Fatalf("ListSnapshotsPage() error = %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations = %v", err)
	}
}
