package proxmox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// ProxmoxConnector is the Proxmox VE implementation of the connector interface.
type ProxmoxConnector struct {
	config connectors.Config
	client *ProxmoxClient
}

// NewProxmoxConnector creates a Proxmox connector with the provided configuration.
func NewProxmoxConnector(cfg connectors.Config) *ProxmoxConnector {
	return &ProxmoxConnector{config: cfg}
}

// Connect establishes an authenticated connection to the Proxmox API.
func (c *ProxmoxConnector) Connect(ctx context.Context) error {
	client := NewProxmoxClient(c.config.Address, c.config.Insecure)

	if strings.Contains(c.config.Username, "!") {
		client.AuthenticateToken(c.config.Username, c.config.Password)
		c.client = client
		return nil
	}

	if err := client.Authenticate(ctx, c.config.Username, c.config.Password); err != nil {
		return fmt.Errorf("proxmox: connect: %w", err)
	}

	c.client = client
	return nil
}

// Discover retrieves VM, container, and infrastructure inventory from Proxmox VE.
func (c *ProxmoxConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("proxmox: not connected, call Connect first")
	}

	startedAt := time.Now()
	nodes, err := listNodes(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("proxmox: discover nodes: %w", err)
	}

	vms := make([]models.VirtualMachine, 0)
	discoveredAt := time.Now().UTC()

	for _, node := range nodes {
		var qemuVMs []map[string]interface{}
		if err := c.client.decode(ctx, fmt.Sprintf("/nodes/%s/qemu", node), &qemuVMs); err != nil {
			return nil, fmt.Errorf("proxmox: discover qemu list for %s: %w", node, err)
		}

		for _, vmStatus := range qemuVMs {
			vmID := stringField(vmStatus, "vmid", "")
			var vmConfig map[string]interface{}
			if err := c.client.decode(ctx, fmt.Sprintf("/nodes/%s/qemu/%s/config", node, vmID), &vmConfig); err != nil {
				return nil, fmt.Errorf("proxmox: discover qemu config for %s/%s: %w", node, vmID, err)
			}

			vm := mapQemuVM(node, vmStatus, vmConfig)
			vm.DiscoveredAt = discoveredAt
			vms = append(vms, vm)
		}

		var containers []map[string]interface{}
		if err := c.client.decode(ctx, fmt.Sprintf("/nodes/%s/lxc", node), &containers); err != nil {
			return nil, fmt.Errorf("proxmox: discover lxc list for %s: %w", node, err)
		}

		for _, ctStatus := range containers {
			vmID := stringField(ctStatus, "vmid", "")
			var ctConfig map[string]interface{}
			if err := c.client.decode(ctx, fmt.Sprintf("/nodes/%s/lxc/%s/config", node, vmID), &ctConfig); err != nil {
				return nil, fmt.Errorf("proxmox: discover lxc config for %s/%s: %w", node, vmID, err)
			}

			vm := mapLXCContainer(node, ctStatus, ctConfig)
			vm.DiscoveredAt = discoveredAt
			vms = append(vms, vm)
		}
	}

	networks, err := discoverNetworks(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("proxmox: discover infrastructure networks: %w", err)
	}

	datastores, err := discoverStorage(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("proxmox: discover infrastructure storage: %w", err)
	}

	hosts, err := discoverNodes(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("proxmox: discover infrastructure nodes: %w", err)
	}

	return &models.DiscoveryResult{
		Source:       c.config.Address,
		Platform:     models.PlatformProxmox,
		VMs:          vms,
		Networks:     networks,
		Datastores:   datastores,
		Hosts:        hosts,
		DiscoveredAt: discoveredAt,
		Duration:     time.Since(startedAt),
	}, nil
}

// Platform returns the Proxmox platform identifier.
func (c *ProxmoxConnector) Platform() models.Platform {
	return models.PlatformProxmox
}

// Close releases the Proxmox client reference.
func (c *ProxmoxConnector) Close() error {
	c.client = nil
	return nil
}

var _ connectors.Connector = (*ProxmoxConnector)(nil)
