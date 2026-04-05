package vmware

import (
	"context"
	"path"
	"strconv"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// mapPowerState maps a VMware power state into the universal Viaduct power state.
func mapPowerState(state types.VirtualMachinePowerState) models.PowerState {
	switch state {
	case types.VirtualMachinePowerStatePoweredOn:
		return models.PowerOn
	case types.VirtualMachinePowerStatePoweredOff:
		return models.PowerOff
	case types.VirtualMachinePowerStateSuspended:
		return models.PowerSuspend
	default:
		return models.PowerUnknown
	}
}

// mapDisks converts VMware virtual disk devices into the universal disk schema.
func mapDisks(devices []types.BaseVirtualDevice) []models.Disk {
	if len(devices) == 0 {
		return nil
	}

	disks := make([]models.Disk, 0)
	for _, device := range devices {
		disk, ok := device.(*types.VirtualDisk)
		if !ok {
			continue
		}

		virtualDevice := disk.GetVirtualDevice()
		name := descriptionLabel(virtualDevice.DeviceInfo)
		thin, fileName, datastore := diskBackingDetails(disk.Backing)
		if name == "" {
			name = fileName
		}

		disks = append(disks, models.Disk{
			ID:             strconv.Itoa(int(disk.Key)),
			Name:           name,
			SizeMB:         int(disk.CapacityInKB / 1024),
			Thin:           thin,
			StorageBackend: datastore,
		})
	}

	return disks
}

// mapNICs converts VMware virtual NIC devices and guest network data into the universal NIC schema.
func mapNICs(devices []types.BaseVirtualDevice, guestNics []types.GuestNicInfo) []models.NIC {
	if len(devices) == 0 {
		return nil
	}

	guestByMAC := make(map[string]types.GuestNicInfo, len(guestNics))
	for _, guestNIC := range guestNics {
		guestByMAC[strings.ToUpper(guestNIC.MacAddress)] = guestNIC
	}

	nics := make([]models.NIC, 0)
	for _, device := range devices {
		card, ok := device.(types.BaseVirtualEthernetCard)
		if !ok {
			continue
		}

		ethernet := card.GetVirtualEthernetCard()
		virtualDevice := ethernet.GetVirtualDevice()
		mac := strings.ToUpper(ethernet.MacAddress)
		guestNIC, hasGuestNIC := guestByMAC[mac]

		network := nicNetworkName(virtualDevice.Backing)
		if network == "" && hasGuestNIC {
			network = guestNIC.Network
		}

		ipAddresses := []string(nil)
		if hasGuestNIC {
			ipAddresses = append(ipAddresses, guestNIC.IpAddress...)
		}

		connected := true
		if virtualDevice.Connectable != nil {
			connected = virtualDevice.Connectable.Connected
		}

		nics = append(nics, models.NIC{
			ID:          strconv.Itoa(int(virtualDevice.Key)),
			Name:        descriptionLabel(virtualDevice.DeviceInfo),
			MACAddress:  mac,
			Network:     network,
			Connected:   connected,
			IPAddresses: ipAddresses,
		})
	}

	return nics
}

// extractClusterName resolves the cluster or compute resource that owns the supplied host reference.
func extractClusterName(ctx context.Context, client *govmomi.Client, hostRef types.ManagedObjectReference) string {
	if client == nil || hostRef.Value == "" {
		return ""
	}

	var host mo.HostSystem
	if err := property.DefaultCollector(client.Client).RetrieveOne(ctx, hostRef, []string{"parent"}, &host); err != nil {
		return ""
	}

	return managedObjectNamePtr(ctx, client, host.Parent)
}

// extractFolderPath walks VMware inventory parents to build a folder path for the VM.
func extractFolderPath(ctx context.Context, client *govmomi.Client, vm mo.VirtualMachine) string {
	if client == nil || vm.Parent == nil || vm.Parent.Value == "" {
		return ""
	}

	segments := make([]string, 0)
	current := *vm.Parent
	for current.Value != "" {
		if current.Type == "Datacenter" {
			break
		}

		var entity mo.ManagedEntity
		if err := property.DefaultCollector(client.Client).RetrieveOne(ctx, current, []string{"name", "parent"}, &entity); err != nil {
			break
		}

		if entity.Name != "" && entity.Name != "vm" {
			segments = append([]string{entity.Name}, segments...)
		}

		if entity.Parent == nil {
			break
		}

		current = *entity.Parent
	}

	if len(segments) == 0 {
		return ""
	}

	return "/" + path.Join(segments...)
}

func diskBackingDetails(backing types.BaseVirtualDeviceBackingInfo) (bool, string, string) {
	switch typed := backing.(type) {
	case *types.VirtualDiskFlatVer2BackingInfo:
		return typed.ThinProvisioned != nil && *typed.ThinProvisioned, typed.FileName, datastoreFromFileName(typed.FileName)
	case *types.VirtualDiskFlatVer1BackingInfo:
		return false, typed.FileName, datastoreFromFileName(typed.FileName)
	case *types.VirtualDiskSparseVer2BackingInfo:
		return true, typed.FileName, datastoreFromFileName(typed.FileName)
	case *types.VirtualDiskSeSparseBackingInfo:
		return true, typed.FileName, datastoreFromFileName(typed.FileName)
	case *types.VirtualDiskRawDiskMappingVer1BackingInfo:
		return false, typed.FileName, datastoreFromFileName(typed.FileName)
	default:
		return false, "", ""
	}
}

func descriptionLabel(info types.BaseDescription) string {
	if info == nil {
		return ""
	}

	description := info.GetDescription()
	if description == nil {
		return ""
	}

	if description.Label != "" {
		return description.Label
	}

	return description.Summary
}

func datastoreFromFileName(fileName string) string {
	if !strings.HasPrefix(fileName, "[") {
		return ""
	}

	end := strings.Index(fileName, "]")
	if end <= 1 {
		return ""
	}

	return strings.TrimSpace(fileName[1:end])
}

func nicNetworkName(backing types.BaseVirtualDeviceBackingInfo) string {
	switch typed := backing.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		return typed.DeviceName
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		return typed.Port.PortgroupKey
	case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
		return typed.OpaqueNetworkId
	}

	return ""
}
