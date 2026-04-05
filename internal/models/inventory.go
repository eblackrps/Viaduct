package models

import "time"

// Platform identifies a supported source or target virtualization platform.
type Platform string

const (
	// PlatformVMware identifies VMware vSphere and vCenter environments.
	PlatformVMware Platform = "vmware"
	// PlatformProxmox identifies Proxmox VE environments.
	PlatformProxmox Platform = "proxmox"
	// PlatformHyperV identifies Microsoft Hyper-V environments.
	PlatformHyperV Platform = "hyperv"
	// PlatformKVM identifies KVM and libvirt environments.
	PlatformKVM Platform = "kvm"
	// PlatformNutanix identifies Nutanix AHV environments.
	PlatformNutanix Platform = "nutanix"
)

// PowerState represents the runtime state of a virtual machine.
type PowerState string

const (
	// PowerOn indicates that a virtual machine is currently running.
	PowerOn PowerState = "on"
	// PowerOff indicates that a virtual machine is currently stopped.
	PowerOff PowerState = "off"
	// PowerSuspend indicates that a virtual machine is suspended.
	PowerSuspend PowerState = "suspended"
	// PowerUnknown indicates that the power state could not be determined.
	PowerUnknown PowerState = "unknown"
)

// Disk represents a virtual disk attached to a workload.
type Disk struct {
	// ID is the source platform identifier for the disk.
	ID string `json:"id"`
	// Name is the human-readable disk name.
	Name string `json:"name"`
	// SizeMB is the disk capacity in mebibytes.
	SizeMB int `json:"size_mb"`
	// Thin reports whether the disk uses thin provisioning.
	Thin bool `json:"thin"`
	// StorageBackend is the backing datastore or storage pool name.
	StorageBackend string `json:"storage_backend"`
}

// NIC represents a virtual network interface attached to a workload.
type NIC struct {
	// ID is the source platform identifier for the NIC.
	ID string `json:"id"`
	// Name is the human-readable NIC name.
	Name string `json:"name"`
	// MACAddress is the layer 2 address assigned to the NIC.
	MACAddress string `json:"mac_address"`
	// Network is the connected network or port group name.
	Network string `json:"network"`
	// Connected reports whether the NIC is currently connected.
	Connected bool `json:"connected"`
	// IPAddresses lists any IP addresses observed on the interface.
	IPAddresses []string `json:"ip_addresses"`
}

// Snapshot represents a point-in-time virtual machine snapshot.
type Snapshot struct {
	// ID is the source platform identifier for the snapshot.
	ID string `json:"id"`
	// Name is the human-readable snapshot name.
	Name string `json:"name"`
	// Description contains any platform-provided snapshot notes.
	Description string `json:"description"`
	// CreatedAt is when the snapshot was created on the source platform.
	CreatedAt time.Time `json:"created_at"`
	// SizeMB is the snapshot size in mebibytes when available.
	SizeMB int `json:"size_mb"`
}

// NetworkInfo represents a discovered virtual network or port group.
type NetworkInfo struct {
	// ID is the source platform identifier for the network.
	ID string `json:"id"`
	// Name is the human-readable network name.
	Name string `json:"name"`
	// Type identifies the network implementation such as standard, distributed, bridge, bond, or vlan.
	Type string `json:"type"`
	// VlanID is the configured VLAN ID when one is associated with the network.
	VlanID int `json:"vlan_id"`
	// Switch is the backing virtual switch, bridge, or fabric name.
	Switch string `json:"switch"`
}

// DatastoreInfo represents a discovered datastore or storage backend.
type DatastoreInfo struct {
	// ID is the source platform identifier for the datastore.
	ID string `json:"id"`
	// Name is the human-readable datastore name.
	Name string `json:"name"`
	// Type identifies the storage backend type such as vmfs, vsan, nfs, lvm, or ceph.
	Type string `json:"type"`
	// CapacityMB is the total datastore capacity in mebibytes.
	CapacityMB int64 `json:"capacity_mb"`
	// FreeMB is the currently available datastore capacity in mebibytes.
	FreeMB int64 `json:"free_mb"`
	// Hosts lists the hosts or nodes attached to the datastore.
	Hosts []string `json:"hosts"`
}

// HostInfo represents a discovered hypervisor host or node.
type HostInfo struct {
	// ID is the source platform identifier for the host.
	ID string `json:"id"`
	// Name is the human-readable host name.
	Name string `json:"name"`
	// Cluster is the cluster containing the host when applicable.
	Cluster string `json:"cluster"`
	// CPUCores is the number of physical CPU cores reported by the platform.
	CPUCores int `json:"cpu_cores"`
	// MemoryMB is the total host memory in mebibytes.
	MemoryMB int64 `json:"memory_mb"`
	// PowerState is the host power state when it can be normalized to the VM power state enum.
	PowerState PowerState `json:"power_state"`
	// ConnectionState is the platform-specific host connection state.
	ConnectionState string `json:"connection_state"`
}

// ClusterInfo represents a discovered compute cluster.
type ClusterInfo struct {
	// ID is the source platform identifier for the cluster.
	ID string `json:"id"`
	// Name is the human-readable cluster name.
	Name string `json:"name"`
	// Hosts lists host names that belong to the cluster.
	Hosts []string `json:"hosts"`
	// TotalCPUCores is the aggregate CPU core count for the cluster.
	TotalCPUCores int `json:"total_cpu_cores"`
	// TotalMemoryMB is the aggregate cluster memory in mebibytes.
	TotalMemoryMB int64 `json:"total_memory_mb"`
	// HAEnabled reports whether high availability is enabled.
	HAEnabled bool `json:"ha_enabled"`
	// DRSEnabled reports whether DRS or equivalent workload balancing is enabled.
	DRSEnabled bool `json:"drs_enabled"`
}

// ResourcePoolInfo represents a discovered resource pool or workload allocation boundary.
type ResourcePoolInfo struct {
	// ID is the source platform identifier for the resource pool.
	ID string `json:"id"`
	// Name is the human-readable resource pool name.
	Name string `json:"name"`
	// Cluster is the parent cluster name when applicable.
	Cluster string `json:"cluster"`
	// CPULimitMHz is the configured CPU limit in MHz when available.
	CPULimitMHz int64 `json:"cpu_limit_mhz"`
	// MemoryLimitMB is the configured memory limit in mebibytes when available.
	MemoryLimitMB int64 `json:"memory_limit_mb"`
}

// VirtualMachine is the universal representation of a VM across all hypervisors.
type VirtualMachine struct {
	// ID is the source platform identifier for the VM.
	ID string `json:"id"`
	// Name is the human-readable VM name.
	Name string `json:"name"`
	// Platform identifies the source platform that owns the VM.
	Platform Platform `json:"platform"`
	// PowerState is the current runtime state of the VM.
	PowerState PowerState `json:"power_state"`
	// CPUCount is the configured number of virtual CPUs.
	CPUCount int `json:"cpu_count"`
	// MemoryMB is the configured memory size in mebibytes.
	MemoryMB int `json:"memory_mb"`
	// Disks contains the VM's virtual disks.
	Disks []Disk `json:"disks"`
	// NICs contains the VM's network interfaces.
	NICs []NIC `json:"nics"`
	// GuestOS is the guest operating system reported by the platform.
	GuestOS string `json:"guest_os"`
	// Host is the hypervisor host currently running the VM.
	Host string `json:"host"`
	// Cluster is the cluster containing the VM when applicable.
	Cluster string `json:"cluster,omitempty"`
	// ResourcePool is the resource pool containing the VM when applicable.
	ResourcePool string `json:"resource_pool,omitempty"`
	// Folder is the source inventory folder path when applicable.
	Folder string `json:"folder,omitempty"`
	// Tags contains normalized key-value metadata tags.
	Tags map[string]string `json:"tags,omitempty"`
	// Snapshots contains any snapshots associated with the VM.
	Snapshots []Snapshot `json:"snapshots,omitempty"`
	// CreatedAt is when the VM was created on the source platform when known.
	CreatedAt time.Time `json:"created_at"`
	// DiscoveredAt is when the VM was observed by Viaduct.
	DiscoveredAt time.Time `json:"discovered_at"`
	// SourceRef is the platform-native reference for follow-up operations.
	SourceRef string `json:"source_ref"`
}

// DiscoveryResult captures the outcome of an inventory discovery run.
type DiscoveryResult struct {
	// Source is the source system or endpoint that was queried.
	Source string `json:"source"`
	// Platform identifies the source platform that was discovered.
	Platform Platform `json:"platform"`
	// VMs contains the normalized virtual machine inventory.
	VMs []VirtualMachine `json:"vms"`
	// Networks contains normalized virtual network inventory discovered alongside VMs.
	Networks []NetworkInfo `json:"networks,omitempty"`
	// Datastores contains normalized storage backend inventory discovered alongside VMs.
	Datastores []DatastoreInfo `json:"datastores,omitempty"`
	// Hosts contains normalized hypervisor host inventory discovered alongside VMs.
	Hosts []HostInfo `json:"hosts,omitempty"`
	// Clusters contains normalized compute cluster inventory discovered alongside VMs.
	Clusters []ClusterInfo `json:"clusters,omitempty"`
	// ResourcePools contains normalized resource pool inventory discovered alongside VMs.
	ResourcePools []ResourcePoolInfo `json:"resource_pools,omitempty"`
	// DiscoveredAt is when discovery completed.
	DiscoveredAt time.Time `json:"discovered_at"`
	// Duration is the total time spent collecting inventory.
	Duration time.Duration `json:"duration"`
	// Errors contains non-fatal discovery errors encountered during the run.
	Errors []string `json:"errors"`
}
