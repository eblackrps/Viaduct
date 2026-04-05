package discovery

import (
	"testing"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestDiffInventories_IdenticalSets(t *testing.T) {
	t.Parallel()

	resultA := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "web-01"}, {Name: "db-01"}}}
	resultB := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "web-01"}, {Name: "db-01"}}}

	diff := DiffInventories(resultA, resultB)
	if len(diff.InBoth) != 2 || len(diff.OnlyInA) != 0 || len(diff.OnlyInB) != 0 {
		t.Fatalf("unexpected diff result: %#v", diff)
	}
}

func TestDiffInventories_DisjointSets(t *testing.T) {
	t.Parallel()

	resultA := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "web-01"}}}
	resultB := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "db-01"}}}

	diff := DiffInventories(resultA, resultB)
	if len(diff.OnlyInA) != 1 || len(diff.OnlyInB) != 1 || len(diff.InBoth) != 0 {
		t.Fatalf("unexpected diff result: %#v", diff)
	}
}

func TestDiffInventories_PartialOverlap(t *testing.T) {
	t.Parallel()

	resultA := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "web-01"}, {Name: "db-01"}}}
	resultB := &models.DiscoveryResult{VMs: []models.VirtualMachine{{Name: "web-01"}, {Name: "cache-01"}}}

	diff := DiffInventories(resultA, resultB)
	if len(diff.InBoth) != 1 || len(diff.OnlyInA) != 1 || len(diff.OnlyInB) != 1 {
		t.Fatalf("unexpected diff result: %#v", diff)
	}
}
