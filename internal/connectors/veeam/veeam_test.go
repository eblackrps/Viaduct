package veeam

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestMapJob(t *testing.T) {
	t.Parallel()

	items := readFixtureList(t, "veeam_jobs.json")
	job := mapJob(items[0])
	if job.Name != "Daily Production" || job.TargetRepo != "primary-repo" {
		t.Fatalf("unexpected job: %#v", job)
	}
}

func TestMapRestorePoint(t *testing.T) {
	t.Parallel()

	items := readFixtureList(t, "veeam_restore_points.json")
	point := mapRestorePoint(items[0])
	if point.VMName != "web-01" || point.Type != "full" {
		t.Fatalf("unexpected restore point: %#v", point)
	}
}

func TestMapRepository(t *testing.T) {
	t.Parallel()

	items := readFixtureList(t, "veeam_repositories.json")
	repository := mapRepository(items[0])
	if repository.Name != "primary-repo" || repository.FreeMB == 0 {
		t.Fatalf("unexpected repository: %#v", repository)
	}
}

func TestCorrelateBackups(t *testing.T) {
	t.Parallel()

	inventory := &models.DiscoveryResult{
		VMs: []models.VirtualMachine{
			{ID: "1", Name: "web-01"},
			{ID: "2", Name: "web-02"},
			{ID: "3", Name: "db-01"},
			{ID: "4", Name: "cache-01"},
			{ID: "5", Name: "ops-01"},
		},
	}
	backups := &models.BackupDiscoveryResult{
		Jobs: []models.BackupJob{
			{Name: "job-1", ProtectedVMs: []string{"web-01", "db-01"}},
			{Name: "job-2", ProtectedVMs: []string{"web-02"}},
		},
	}

	correlated := CorrelateBackups(inventory, backups)
	if len(correlated) != 3 {
		t.Fatalf("len(CorrelateBackups()) = %d, want 3", len(correlated))
	}
}

func TestCorrelateBackups_NoMatch(t *testing.T) {
	t.Parallel()

	inventory := &models.DiscoveryResult{VMs: []models.VirtualMachine{{ID: "1", Name: "web-01"}}}
	backups := &models.BackupDiscoveryResult{Jobs: []models.BackupJob{{Name: "job-1", ProtectedVMs: []string{"db-01"}}}}
	correlated := CorrelateBackups(inventory, backups)
	if len(correlated) != 0 {
		t.Fatalf("len(CorrelateBackups()) = %d, want 0", len(correlated))
	}
}

func TestVeeamConnector_ConnectFailure(t *testing.T) {
	t.Parallel()

	connector := NewVeeamConnector(connectors.Config{
		Address:  "127.0.0.1:1",
		Username: "demo",
		Password: "demo",
		Insecure: true,
	})
	if err := connector.Connect(context.Background()); err == nil {
		t.Fatal("Connect() error = nil, want error")
	}
}

func readFixtureList(t *testing.T, name string) []map[string]interface{} {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(payload, &items); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", name, err)
	}

	return items
}
