package discovery

import "github.com/eblackrps/viaduct/internal/models"

// VMPair contains the representation of the same VM discovered in two inventories.
type VMPair struct {
	// A is the VM as it appears in inventory A.
	A models.VirtualMachine `json:"a" yaml:"a"`
	// B is the VM as it appears in inventory B.
	B models.VirtualMachine `json:"b" yaml:"b"`
}

// InventoryDiff contains the set-wise comparison of two normalized inventories.
type InventoryDiff struct {
	// OnlyInA contains VMs found only in inventory A.
	OnlyInA []models.VirtualMachine `json:"only_in_a" yaml:"only_in_a"`
	// OnlyInB contains VMs found only in inventory B.
	OnlyInB []models.VirtualMachine `json:"only_in_b" yaml:"only_in_b"`
	// InBoth contains VM pairs found in both inventories by name.
	InBoth []VMPair `json:"in_both" yaml:"in_both"`
}

// DiffInventories compares two discovery results using VM name as the join key.
func DiffInventories(a, b *models.DiscoveryResult) *InventoryDiff {
	diff := &InventoryDiff{}
	if a == nil && b == nil {
		return diff
	}
	if a == nil {
		diff.OnlyInB = append(diff.OnlyInB, b.VMs...)
		return diff
	}
	if b == nil {
		diff.OnlyInA = append(diff.OnlyInA, a.VMs...)
		return diff
	}

	bByName := make(map[string]models.VirtualMachine, len(b.VMs))
	for _, vm := range b.VMs {
		bByName[vm.Name] = vm
	}

	seen := make(map[string]struct{}, len(a.VMs))
	for _, vmA := range a.VMs {
		vmB, ok := bByName[vmA.Name]
		if !ok {
			diff.OnlyInA = append(diff.OnlyInA, vmA)
			continue
		}

		diff.InBoth = append(diff.InBoth, VMPair{A: vmA, B: vmB})
		seen[vmA.Name] = struct{}{}
	}

	for _, vmB := range b.VMs {
		if _, ok := seen[vmB.Name]; ok {
			continue
		}

		diff.OnlyInB = append(diff.OnlyInB, vmB)
	}

	return diff
}
