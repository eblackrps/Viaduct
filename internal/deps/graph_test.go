package deps

import (
	"testing"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestBuildGraph_BasicInventory(t *testing.T) {
	t.Parallel()

	inventory := &models.DiscoveryResult{
		VMs: []models.VirtualMachine{
			{
				ID:       "vm-1",
				Name:     "web-01",
				Platform: models.PlatformVMware,
				NICs:     []models.NIC{{Network: "VM Network"}},
				Disks:    []models.Disk{{Name: "disk0", StorageBackend: "datastore1"}},
			},
			{
				ID:       "vm-2",
				Name:     "web-02",
				Platform: models.PlatformVMware,
				NICs:     []models.NIC{{Network: "VM Network"}},
				Disks:    []models.Disk{{Name: "disk0", StorageBackend: "datastore1"}},
			},
			{
				ID:       "vm-3",
				Name:     "db-01",
				Platform: models.PlatformProxmox,
				NICs:     []models.NIC{{Network: "Management"}},
				Disks:    []models.Disk{{Name: "disk1", StorageBackend: "ceph-ssd"}},
			},
		},
	}

	graph := BuildGraph(inventory, nil)
	if len(graph.Nodes) != 7 {
		t.Fatalf("len(Nodes) = %d, want 7", len(graph.Nodes))
	}
	if len(graph.Edges) != 6 {
		t.Fatalf("len(Edges) = %d, want 6", len(graph.Edges))
	}
}

func TestBuildGraph_WithBackups(t *testing.T) {
	t.Parallel()

	inventory := &models.DiscoveryResult{
		VMs: []models.VirtualMachine{
			{ID: "vm-1", Name: "web-01"},
			{ID: "vm-2", Name: "db-01"},
		},
	}
	backups := &models.BackupDiscoveryResult{
		Jobs: []models.BackupJob{
			{ID: "job-1", Name: "Daily", ProtectedVMs: []string{"web-01", "db-01"}},
		},
	}

	graph := BuildGraph(inventory, backups)
	if len(graph.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(graph.Edges))
	}
}

func TestBuildGraph_EmptyInventory(t *testing.T) {
	t.Parallel()

	graph := BuildGraph(nil, nil)
	if len(graph.Nodes) != 0 || len(graph.Edges) != 0 {
		t.Fatalf("unexpected graph: %#v", graph)
	}
}
