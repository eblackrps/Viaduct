package hyperv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

const (
	listVMsCommand   = "Get-VM | Select-Object Name, VMId, State, ProcessorCount, MemoryAssigned, Path, Generation | ConvertTo-Json"
	listDisksCommand = "Get-VMHardDiskDrive -VMName * | Select-Object VMName, Path, DiskNumber, ControllerType | ConvertTo-Json"
	listNICsCommand  = "Get-VMNetworkAdapter -VMName * | Select-Object VMName, MacAddress, SwitchName, Connected, IPAddresses | ConvertTo-Json"
)

// HyperVConnector discovers VM inventory from Hyper-V over WinRM.
type HyperVConnector struct {
	config  connectors.Config
	client  *WinRMClient
	execute func(context.Context, string) (string, error)
}

// NewHyperVConnector creates a Hyper-V connector.
func NewHyperVConnector(cfg connectors.Config) *HyperVConnector {
	return &HyperVConnector{config: cfg}
}

// Connect creates a WinRM client and validates connectivity with a simple command.
func (c *HyperVConnector) Connect(ctx context.Context) error {
	client := NewWinRMClient(c.config)
	c.client = client
	c.execute = client.Execute

	if _, err := c.execute(ctx, "Write-Output 'viaduct'"); err != nil {
		return fmt.Errorf("hyperv: connect: %w", err)
	}

	return nil
}

// Discover gathers VM, disk, NIC, and network inventory from Hyper-V.
func (c *HyperVConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if c.execute == nil {
		return nil, fmt.Errorf("hyperv: not connected, call Connect first")
	}

	startedAt := time.Now()
	vmsPayload, err := c.execute(ctx, listVMsCommand)
	if err != nil {
		return nil, fmt.Errorf("hyperv: discover VMs: %w", err)
	}
	disksPayload, err := c.execute(ctx, listDisksCommand)
	if err != nil {
		return nil, fmt.Errorf("hyperv: discover disks: %w", err)
	}
	nicsPayload, err := c.execute(ctx, listNICsCommand)
	if err != nil {
		return nil, fmt.Errorf("hyperv: discover NICs: %w", err)
	}

	vms, err := decodeVMs(vmsPayload)
	if err != nil {
		return nil, fmt.Errorf("hyperv: decode VMs: %w", err)
	}
	disks, err := decodeDisks(disksPayload)
	if err != nil {
		return nil, fmt.Errorf("hyperv: decode disks: %w", err)
	}
	adapters, err := decodeAdapters(nicsPayload)
	if err != nil {
		return nil, fmt.Errorf("hyperv: decode NICs: %w", err)
	}

	disksByVM := make(map[string][]hyperVDisk)
	for _, disk := range disks {
		disksByVM[disk.VMName] = append(disksByVM[disk.VMName], disk)
	}
	adaptersByVM := make(map[string][]hyperVAdapter)
	networks := make([]models.NetworkInfo, 0)
	seenNetworks := make(map[string]struct{})
	for _, adapter := range adapters {
		adaptersByVM[adapter.VMName] = append(adaptersByVM[adapter.VMName], adapter)
		if _, ok := seenNetworks[adapter.SwitchName]; !ok && adapter.SwitchName != "" {
			seenNetworks[adapter.SwitchName] = struct{}{}
			networks = append(networks, models.NetworkInfo{Name: adapter.SwitchName, Type: "switch"})
		}
	}

	discoveredAt := time.Now().UTC()
	virtualMachines := make([]models.VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		virtualMachines = append(virtualMachines, models.VirtualMachine{
			ID:           vm.VMID,
			Name:         vm.Name,
			Platform:     models.PlatformHyperV,
			PowerState:   mapHyperVPowerState(vm.State),
			CPUCount:     vm.ProcessorCount,
			MemoryMB:     int(vm.MemoryAssigned / (1024 * 1024)),
			Disks:        mapHyperVDisks(disksByVM[vm.Name]),
			NICs:         mapHyperVNICs(adaptersByVM[vm.Name]),
			Host:         c.config.Address,
			DiscoveredAt: discoveredAt,
			SourceRef:    vm.VMID,
		})
	}

	return &models.DiscoveryResult{
		Source:       c.config.Address,
		Platform:     models.PlatformHyperV,
		VMs:          virtualMachines,
		Networks:     networks,
		DiscoveredAt: discoveredAt,
		Duration:     time.Since(startedAt),
	}, nil
}

// Platform returns the Hyper-V platform identifier.
func (c *HyperVConnector) Platform() models.Platform {
	return models.PlatformHyperV
}

// Close drops the WinRM client reference.
func (c *HyperVConnector) Close() error {
	c.client = nil
	c.execute = nil
	return nil
}

var _ connectors.Connector = (*HyperVConnector)(nil)

func decodeVMs(payload string) ([]hyperVVM, error) {
	var items []hyperVVM
	if err := decodeJSONList(payload, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeDisks(payload string) ([]hyperVDisk, error) {
	var items []hyperVDisk
	if err := decodeJSONList(payload, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeAdapters(payload string) ([]hyperVAdapter, error) {
	var items []hyperVAdapter
	if err := decodeJSONList(payload, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeJSONList[T any](payload string, out *[]T) error {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" || trimmed == "null" {
		*out = nil
		return nil
	}

	if strings.HasPrefix(trimmed, "[") {
		return json.Unmarshal([]byte(trimmed), out)
	}

	var single T
	if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
		return err
	}
	*out = []T{single}
	return nil
}
