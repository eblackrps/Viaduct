package lifecycle

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
	"gopkg.in/yaml.v3"
)

// LicenseModel describes how a platform license is billed.
type LicenseModel string

const (
	// LicenseModelPerSocket bills by physical socket equivalents.
	LicenseModelPerSocket LicenseModel = "per-socket"
	// LicenseModelPerCore bills by virtual or licensable cores.
	LicenseModelPerCore LicenseModel = "per-core"
	// LicenseModelPerVM bills at a flat per-workload rate.
	LicenseModelPerVM LicenseModel = "per-vm"
	// LicenseModelNone represents no platform license charge.
	LicenseModelNone LicenseModel = "none"
)

// CostProfile defines the cost assumptions for a target platform.
type CostProfile struct {
	// Platform identifies the platform the pricing profile applies to.
	Platform models.Platform `json:"platform" yaml:"platform"`
	// Name is the human-readable profile name.
	Name string `json:"name" yaml:"name"`
	// CPUCostPerCoreMonth is the monthly cost per CPU core.
	CPUCostPerCoreMonth float64 `json:"cpu_cost_per_core_month" yaml:"cpu_cost_per_core_month"`
	// MemoryCostPerGBMonth is the monthly cost per GB of memory.
	MemoryCostPerGBMonth float64 `json:"memory_cost_per_gb_month" yaml:"memory_cost_per_gb_month"`
	// StorageCostPerGBMonth is the monthly cost per GB of storage.
	StorageCostPerGBMonth float64 `json:"storage_cost_per_gb_month" yaml:"storage_cost_per_gb_month"`
	// LicenseCostPerSocketMonth is the monthly license cost per socket.
	LicenseCostPerSocketMonth float64 `json:"license_cost_per_socket_month" yaml:"license_cost_per_socket_month"`
	// LicenseCostPerCoreMonth is the monthly license cost per core.
	LicenseCostPerCoreMonth float64 `json:"license_cost_per_core_month" yaml:"license_cost_per_core_month"`
	// LicenseModel identifies how licensing is calculated.
	LicenseModel LicenseModel `json:"license_model" yaml:"license_model"`
	// SupportCostMonthly is the monthly support charge.
	SupportCostMonthly float64 `json:"support_cost_monthly" yaml:"support_cost_monthly"`
	// Notes records free-form profile context.
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// VMCost contains the computed monthly and annual cost of a VM on a platform.
type VMCost struct {
	// VM is the workload being priced.
	VM models.VirtualMachine `json:"vm" yaml:"vm"`
	// MonthlyCPUCost is the monthly CPU cost component.
	MonthlyCPUCost float64 `json:"monthly_cpu_cost" yaml:"monthly_cpu_cost"`
	// MonthlyMemoryCost is the monthly memory cost component.
	MonthlyMemoryCost float64 `json:"monthly_memory_cost" yaml:"monthly_memory_cost"`
	// MonthlyStorageCost is the monthly storage cost component.
	MonthlyStorageCost float64 `json:"monthly_storage_cost" yaml:"monthly_storage_cost"`
	// MonthlyLicenseCost is the monthly license cost component.
	MonthlyLicenseCost float64 `json:"monthly_license_cost" yaml:"monthly_license_cost"`
	// MonthlyTotal is the total monthly workload cost.
	MonthlyTotal float64 `json:"monthly_total" yaml:"monthly_total"`
	// AnnualTotal is the total annual workload cost.
	AnnualTotal float64 `json:"annual_total" yaml:"annual_total"`
}

// PlatformComparison compares the same VM across all known cost profiles.
type PlatformComparison struct {
	// VM is the workload being compared.
	VM models.VirtualMachine `json:"vm" yaml:"vm"`
	// CostByPlatform contains the computed cost for each available platform.
	CostByPlatform map[models.Platform]VMCost `json:"cost_by_platform" yaml:"cost_by_platform"`
	// CheapestPlatform is the least expensive platform for the workload.
	CheapestPlatform models.Platform `json:"cheapest_platform" yaml:"cheapest_platform"`
	// MonthlySavings is the difference between the current and cheapest monthly totals.
	MonthlySavings float64 `json:"monthly_savings" yaml:"monthly_savings"`
}

// FleetCost aggregates per-platform cost totals across a VM fleet.
type FleetCost struct {
	// Platform is the platform the fleet is being priced on.
	Platform models.Platform `json:"platform" yaml:"platform"`
	// TotalMonthly is the aggregate monthly cost.
	TotalMonthly float64 `json:"total_monthly" yaml:"total_monthly"`
	// TotalAnnual is the aggregate annual cost.
	TotalAnnual float64 `json:"total_annual" yaml:"total_annual"`
	// VMCosts contains the individual workload cost breakdowns.
	VMCosts []VMCost `json:"vm_costs" yaml:"vm_costs"`
	// ByCluster aggregates monthly totals by cluster.
	ByCluster map[string]float64 `json:"by_cluster,omitempty" yaml:"by_cluster,omitempty"`
	// ByFolder aggregates monthly totals by folder.
	ByFolder map[string]float64 `json:"by_folder,omitempty" yaml:"by_folder,omitempty"`
}

// CostEngine calculates workload cost across one or more platforms.
type CostEngine struct {
	profiles map[models.Platform]*CostProfile
}

// NewCostEngine creates an empty cost engine.
func NewCostEngine() *CostEngine {
	return &CostEngine{
		profiles: make(map[models.Platform]*CostProfile),
	}
}

// AddProfile registers or replaces a cost profile for a platform.
func (e *CostEngine) AddProfile(profile CostProfile) {
	if e == nil || profile.Platform == "" {
		return
	}

	cloned := profile
	e.profiles[profile.Platform] = &cloned
}

// CalculateVMCost computes workload cost on the supplied platform.
func (e *CostEngine) CalculateVMCost(vm models.VirtualMachine, platform models.Platform) (VMCost, error) {
	if e == nil {
		return VMCost{}, fmt.Errorf("calculate VM cost: engine is nil")
	}

	profile, ok := e.profiles[platform]
	if !ok {
		return VMCost{}, fmt.Errorf("calculate VM cost: missing profile for platform %q", platform)
	}

	storageGB := 0.0
	for _, disk := range vm.Disks {
		storageGB += float64(disk.SizeMB) / 1024
	}

	monthlyCPU := float64(vm.CPUCount) * profile.CPUCostPerCoreMonth
	monthlyMemory := (float64(vm.MemoryMB) / 1024) * profile.MemoryCostPerGBMonth
	monthlyStorage := storageGB * profile.StorageCostPerGBMonth
	monthlyLicense := calculateLicenseCost(vm, profile)
	monthlyTotal := roundCurrency(monthlyCPU + monthlyMemory + monthlyStorage + monthlyLicense + profile.SupportCostMonthly)

	return VMCost{
		VM:                 vm,
		MonthlyCPUCost:     roundCurrency(monthlyCPU),
		MonthlyMemoryCost:  roundCurrency(monthlyMemory),
		MonthlyStorageCost: roundCurrency(monthlyStorage),
		MonthlyLicenseCost: roundCurrency(monthlyLicense),
		MonthlyTotal:       monthlyTotal,
		AnnualTotal:        roundCurrency(monthlyTotal * 12),
	}, nil
}

// CompareVM calculates the cost of a workload across all known platform profiles.
func (e *CostEngine) CompareVM(vm models.VirtualMachine) (*PlatformComparison, error) {
	if e == nil {
		return nil, fmt.Errorf("compare VM: engine is nil")
	}
	if len(e.profiles) == 0 {
		return nil, fmt.Errorf("compare VM: no cost profiles configured")
	}

	comparison := &PlatformComparison{
		VM:             vm,
		CostByPlatform: make(map[models.Platform]VMCost, len(e.profiles)),
	}

	var cheapestPlatform models.Platform
	cheapestCost := math.MaxFloat64

	for platform := range e.profiles {
		cost, err := e.CalculateVMCost(vm, platform)
		if err != nil {
			return nil, err
		}
		comparison.CostByPlatform[platform] = cost
		if cost.MonthlyTotal < cheapestCost {
			cheapestCost = cost.MonthlyTotal
			cheapestPlatform = platform
		}
	}

	currentCost, err := e.CalculateVMCost(vm, vm.Platform)
	if err != nil {
		currentCost = VMCost{}
	}

	comparison.CheapestPlatform = cheapestPlatform
	comparison.MonthlySavings = roundCurrency(currentCost.MonthlyTotal - cheapestCost)
	if comparison.MonthlySavings < 0 {
		comparison.MonthlySavings = 0
	}

	return comparison, nil
}

// CalculateFleetCost computes aggregate monthly and annual totals for a VM fleet on a platform.
func (e *CostEngine) CalculateFleetCost(platform models.Platform, vms []models.VirtualMachine) (*FleetCost, error) {
	if e == nil {
		return nil, fmt.Errorf("calculate fleet cost: engine is nil")
	}

	fleet := &FleetCost{
		Platform:  platform,
		VMCosts:   make([]VMCost, 0, len(vms)),
		ByCluster: make(map[string]float64),
		ByFolder:  make(map[string]float64),
	}

	for _, vm := range vms {
		cost, err := e.CalculateVMCost(vm, platform)
		if err != nil {
			return nil, err
		}

		fleet.VMCosts = append(fleet.VMCosts, cost)
		fleet.TotalMonthly += cost.MonthlyTotal
		fleet.TotalAnnual += cost.AnnualTotal

		cluster := vm.Cluster
		if cluster == "" {
			cluster = "unassigned"
		}
		fleet.ByCluster[cluster] = roundCurrency(fleet.ByCluster[cluster] + cost.MonthlyTotal)

		folder := vm.Folder
		if folder == "" {
			folder = "root"
		}
		fleet.ByFolder[folder] = roundCurrency(fleet.ByFolder[folder] + cost.MonthlyTotal)
	}

	fleet.TotalMonthly = roundCurrency(fleet.TotalMonthly)
	fleet.TotalAnnual = roundCurrency(fleet.TotalAnnual)
	return fleet, nil
}

// LoadCostProfile loads a cost profile from YAML on disk.
func LoadCostProfile(path string) (*CostProfile, error) {
	// #nosec G304 -- cost profiles are loaded from an explicit operator-selected path or profile directory.
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load cost profile: read %s: %w", path, err)
	}

	var profile CostProfile
	if err := yaml.Unmarshal(payload, &profile); err != nil {
		return nil, fmt.Errorf("load cost profile: decode %s: %w", path, err)
	}
	if profile.Platform == "" {
		return nil, fmt.Errorf("load cost profile: platform is required")
	}
	if profile.Name == "" {
		profile.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if profile.LicenseModel == "" {
		profile.LicenseModel = LicenseModelNone
	}

	return &profile, nil
}

// LoadCostProfilesDir loads all YAML cost profiles from a directory.
func LoadCostProfilesDir(path string) ([]*CostProfile, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("load cost profiles dir: read %s: %w", path, err)
	}

	profiles := make([]*CostProfile, 0)
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		profile, err := LoadCostProfile(filepath.Join(path, entry.Name()))
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func calculateLicenseCost(vm models.VirtualMachine, profile *CostProfile) float64 {
	if profile == nil {
		return 0
	}

	switch profile.LicenseModel {
	case LicenseModelPerSocket:
		return float64(estimatedSockets(vm.CPUCount)) * profile.LicenseCostPerSocketMonth
	case LicenseModelPerCore:
		return float64(vm.CPUCount) * profile.LicenseCostPerCoreMonth
	case LicenseModelPerVM:
		if profile.LicenseCostPerSocketMonth > 0 {
			return profile.LicenseCostPerSocketMonth
		}
		return profile.LicenseCostPerCoreMonth
	default:
		return 0
	}
}

func estimatedSockets(cpuCount int) int {
	if cpuCount <= 0 {
		return 1
	}

	return int(math.Ceil(float64(cpuCount) / 8.0))
}

func roundCurrency(value float64) float64 {
	return math.Round(value*100) / 100
}
