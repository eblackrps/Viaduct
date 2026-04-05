package lifecycle

import (
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestCostEngine_CalculateVMCost_VMwareProfile(t *testing.T) {
	t.Parallel()

	engine := NewCostEngine()
	engine.AddProfile(CostProfile{
		Platform:                  models.PlatformVMware,
		Name:                      "VMware VCF",
		CPUCostPerCoreMonth:       12,
		MemoryCostPerGBMonth:      4,
		StorageCostPerGBMonth:     0.5,
		LicenseCostPerCoreMonth:   8,
		LicenseModel:              LicenseModelPerCore,
		SupportCostMonthly:        25,
		LicenseCostPerSocketMonth: 0,
	})

	cost, err := engine.CalculateVMCost(sampleLifecycleVM(), models.PlatformVMware)
	if err != nil {
		t.Fatalf("CalculateVMCost() error = %v", err)
	}
	if cost.MonthlyTotal != 126 {
		t.Fatalf("MonthlyTotal = %.2f, want 126.00", cost.MonthlyTotal)
	}
}

func TestCostEngine_CalculateVMCost_ProxmoxProfile(t *testing.T) {
	t.Parallel()

	engine := NewCostEngine()
	engine.AddProfile(CostProfile{
		Platform:              models.PlatformProxmox,
		Name:                  "Proxmox Enterprise",
		CPUCostPerCoreMonth:   6,
		MemoryCostPerGBMonth:  2,
		StorageCostPerGBMonth: 0.4,
		LicenseModel:          LicenseModelNone,
		SupportCostMonthly:    10,
	})

	cost, err := engine.CalculateVMCost(sampleLifecycleVM(), models.PlatformProxmox)
	if err != nil {
		t.Fatalf("CalculateVMCost() error = %v", err)
	}
	if cost.MonthlyTotal != 46 {
		t.Fatalf("MonthlyTotal = %.2f, want 46.00", cost.MonthlyTotal)
	}
}

func TestCostEngine_CompareVM_CheapestPlatform(t *testing.T) {
	t.Parallel()

	engine := newLifecycleCostEngine()
	comparison, err := engine.CompareVM(sampleLifecycleVM())
	if err != nil {
		t.Fatalf("CompareVM() error = %v", err)
	}
	if comparison.CheapestPlatform != models.PlatformProxmox {
		t.Fatalf("CheapestPlatform = %q, want %q", comparison.CheapestPlatform, models.PlatformProxmox)
	}
	if comparison.MonthlySavings <= 0 {
		t.Fatalf("MonthlySavings = %.2f, want > 0", comparison.MonthlySavings)
	}
}

func TestCostEngine_CalculateFleetCost_AggregatesByCluster(t *testing.T) {
	t.Parallel()

	engine := newLifecycleCostEngine()
	fleet, err := engine.CalculateFleetCost(models.PlatformProxmox, []models.VirtualMachine{
		sampleLifecycleVM(),
		{
			ID:         "vm-2",
			Name:       "db-01",
			Platform:   models.PlatformVMware,
			CPUCount:   8,
			MemoryMB:   8192,
			Cluster:    "Cluster-A",
			Folder:     "/Prod/DB",
			PowerState: models.PowerOn,
			Disks: []models.Disk{
				{ID: "disk-2", SizeMB: 20480},
			},
		},
	})
	if err != nil {
		t.Fatalf("CalculateFleetCost() error = %v", err)
	}
	if fleet.TotalMonthly <= 0 || fleet.ByCluster["Cluster-A"] <= 0 || fleet.ByFolder["/Prod/Web"] <= 0 {
		t.Fatalf("unexpected fleet totals: %#v", fleet)
	}
}

func TestCostEngine_CalculateVMCost_MissingProfile(t *testing.T) {
	t.Parallel()

	engine := NewCostEngine()
	if _, err := engine.CalculateVMCost(sampleLifecycleVM(), models.PlatformNutanix); err == nil {
		t.Fatal("CalculateVMCost() error = nil, want missing profile")
	}
}

func TestLoadCostProfile_File_Expected(t *testing.T) {
	t.Parallel()

	profile, err := LoadCostProfile(filepath.Join("..", "..", "configs", "cost-profiles", "vmware-vcf.yaml"))
	if err != nil {
		t.Fatalf("LoadCostProfile() error = %v", err)
	}
	if profile.Platform != models.PlatformVMware {
		t.Fatalf("Platform = %q, want %q", profile.Platform, models.PlatformVMware)
	}
}

func newLifecycleCostEngine() *CostEngine {
	engine := NewCostEngine()
	engine.AddProfile(CostProfile{
		Platform:                models.PlatformVMware,
		Name:                    "VMware VCF",
		CPUCostPerCoreMonth:     12,
		MemoryCostPerGBMonth:    4,
		StorageCostPerGBMonth:   0.5,
		LicenseCostPerCoreMonth: 8,
		LicenseModel:            LicenseModelPerCore,
		SupportCostMonthly:      25,
	})
	engine.AddProfile(CostProfile{
		Platform:              models.PlatformProxmox,
		Name:                  "Proxmox Enterprise",
		CPUCostPerCoreMonth:   6,
		MemoryCostPerGBMonth:  2,
		StorageCostPerGBMonth: 0.4,
		LicenseModel:          LicenseModelNone,
		SupportCostMonthly:    10,
	})
	return engine
}

func sampleLifecycleVM() models.VirtualMachine {
	return models.VirtualMachine{
		ID:         "vm-1",
		Name:       "web-01",
		Platform:   models.PlatformVMware,
		PowerState: models.PowerOn,
		CPUCount:   4,
		MemoryMB:   4096,
		Cluster:    "Cluster-A",
		Folder:     "/Prod/Web",
		Tags: map[string]string{
			"env": "production",
		},
		Disks: []models.Disk{
			{ID: "disk-1", SizeMB: 10240},
		},
	}
}
