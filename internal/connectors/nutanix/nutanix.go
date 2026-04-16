package nutanix

import (
	"context"
	"fmt"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// NutanixConnector discovers VM inventory from Prism Central v3.
type NutanixConnector struct {
	config connectors.Config
	client *PrismClient
}

// NewNutanixConnector creates a Nutanix connector from shared connector configuration.
func NewNutanixConnector(cfg connectors.Config) *NutanixConnector {
	return &NutanixConnector{config: cfg}
}

// Connect validates Prism Central access using the current credentials.
func (c *NutanixConnector) Connect(ctx context.Context) error {
	client := NewPrismClient(c.config.Address, c.config.Username, c.config.Password, c.config.Insecure, c.config.RequestID)
	if _, err := client.Get(ctx, "/api/nutanix/v3/users/me"); err != nil {
		return fmt.Errorf("nutanix: connect: %w", err)
	}
	c.client = client
	return nil
}

// Discover retrieves VMs, clusters, hosts, and subnets from Prism Central.
func (c *NutanixConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("nutanix: not connected, call Connect first")
	}

	startedAt := time.Now()
	vmItems, err := c.client.ListAll(ctx, "/api/nutanix/v3/vms/list", map[string]interface{}{"kind": "vm"})
	if err != nil {
		return nil, fmt.Errorf("nutanix: list VMs: %w", err)
	}
	clusterItems, err := c.client.ListAll(ctx, "/api/nutanix/v3/clusters/list", map[string]interface{}{"kind": "cluster"})
	if err != nil {
		return nil, fmt.Errorf("nutanix: list clusters: %w", err)
	}
	hostItems, err := c.client.ListAll(ctx, "/api/nutanix/v3/hosts/list", map[string]interface{}{"kind": "host"})
	if err != nil {
		return nil, fmt.Errorf("nutanix: list hosts: %w", err)
	}
	subnetItems, err := c.client.ListAll(ctx, "/api/nutanix/v3/subnets/list", map[string]interface{}{"kind": "subnet"})
	if err != nil {
		return nil, fmt.Errorf("nutanix: list subnets: %w", err)
	}

	discoveredAt := time.Now().UTC()
	virtualMachines := make([]models.VirtualMachine, 0, len(vmItems))
	for _, item := range vmItems {
		vm := mapNutanixVM(item)
		vm.DiscoveredAt = discoveredAt
		virtualMachines = append(virtualMachines, vm)
	}

	clusters := make([]models.ClusterInfo, 0, len(clusterItems))
	for _, item := range clusterItems {
		clusters = append(clusters, models.ClusterInfo{
			ID:            stringValue(mapValue(item, "metadata"), "uuid"),
			Name:          stringValue(mapValue(item, "status"), "name"),
			TotalCPUCores: intValue(mapValue(item, "status.resources")["num_cpu_cores"]),
			TotalMemoryMB: int64Value(mapValue(item, "status.resources")["memory_capacity_mib"]),
			HAEnabled:     true,
			DRSEnabled:    true,
		})
	}

	hosts := make([]models.HostInfo, 0, len(hostItems))
	for _, item := range hostItems {
		statusResources := mapValue(mapValue(item, "status"), "resources")
		hosts = append(hosts, models.HostInfo{
			ID:              stringValue(mapValue(item, "metadata"), "uuid"),
			Name:            stringValue(mapValue(item, "status"), "name"),
			Cluster:         referenceName(statusResources, "cluster_reference"),
			CPUCores:        intValue(statusResources["num_cpu_cores"]),
			MemoryMB:        int64Value(statusResources["memory_capacity_mib"]),
			PowerState:      models.PowerOn,
			ConnectionState: "connected",
		})
	}

	networks := make([]models.NetworkInfo, 0, len(subnetItems))
	for _, item := range subnetItems {
		statusResources := mapValue(mapValue(item, "status"), "resources")
		networks = append(networks, models.NetworkInfo{
			ID:     stringValue(mapValue(item, "metadata"), "uuid"),
			Name:   stringValue(mapValue(item, "status"), "name"),
			Type:   "subnet",
			VlanID: intValue(statusResources["vlan_id"]),
			Switch: stringValue(statusResources, "virtual_switch_name"),
		})
	}

	return &models.DiscoveryResult{
		Source:       c.config.Address,
		Platform:     models.PlatformNutanix,
		VMs:          virtualMachines,
		Networks:     networks,
		Hosts:        hosts,
		Clusters:     clusters,
		DiscoveredAt: discoveredAt,
		Duration:     time.Since(startedAt),
	}, nil
}

// Platform returns the Nutanix platform identifier.
func (c *NutanixConnector) Platform() models.Platform {
	return models.PlatformNutanix
}

// Close clears the Prism client reference.
func (c *NutanixConnector) Close() error {
	c.client = nil
	return nil
}

var _ connectors.Connector = (*NutanixConnector)(nil)
