package migrate

import (
	"testing"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestNetworkMapper_SingleNIC(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(map[string]string{"VM Network": "vmbr0"}, []models.NetworkInfo{{Name: "vmbr0", VlanID: 100}})
	mapped, err := mapper.MapNIC(models.NIC{Name: "nic-1", Network: "VM Network"})
	if err != nil {
		t.Fatalf("MapNIC() error = %v", err)
	}

	if mapped.TargetNetwork != "vmbr0" || mapped.TargetVLAN != 100 {
		t.Fatalf("unexpected mapped NIC: %#v", mapped)
	}
}

func TestNetworkMapper_MultipleNICs(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(
		map[string]string{"VM Network": "vmbr0", "Management": "vmbr1", "Backup": "vmbr2"},
		[]models.NetworkInfo{{Name: "vmbr0"}, {Name: "vmbr1"}, {Name: "vmbr2"}},
	)
	mapped, errs := mapper.MapAllNICs([]models.NIC{
		{Network: "VM Network"},
		{Network: "Management"},
		{Network: "Backup"},
	})
	if len(errs) != 0 {
		t.Fatalf("MapAllNICs() errors = %v, want none", errs)
	}
	if len(mapped) != 3 {
		t.Fatalf("len(mapped) = %d, want 3", len(mapped))
	}
}

func TestNetworkMapper_MissingMapping(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(map[string]string{"VM Network": "vmbr0"}, []models.NetworkInfo{{Name: "vmbr0"}})
	_, err := mapper.MapNIC(models.NIC{Network: "Management"})
	if err == nil {
		t.Fatal("MapNIC() error = nil, want error")
	}
}

func TestNetworkMapper_InvalidTarget(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(map[string]string{"VM Network": "vmbr9"}, []models.NetworkInfo{{Name: "vmbr0"}})
	errs := mapper.ValidateTargetNetworks()
	if len(errs) != 1 {
		t.Fatalf("len(ValidateTargetNetworks()) = %d, want 1", len(errs))
	}
}

func TestNetworkMapper_EmptyMappings(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(nil, []models.NetworkInfo{{Name: "vmbr0"}})
	_, errs := mapper.MapAllNICs([]models.NIC{{Network: "VM Network"}})
	if len(errs) != 1 {
		t.Fatalf("len(MapAllNICs() errs) = %d, want 1", len(errs))
	}
}

func TestValidateTargetNetworks(t *testing.T) {
	t.Parallel()

	mapper := NewNetworkMapper(
		map[string]string{"VM Network": "vmbr0", "Management": "vmbr9"},
		[]models.NetworkInfo{{Name: "vmbr0"}},
	)
	errs := mapper.ValidateTargetNetworks()
	if len(errs) != 1 {
		t.Fatalf("len(ValidateTargetNetworks()) = %d, want 1", len(errs))
	}
}
