//go:build libvirt

package kvm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"libvirt.org/go/libvirt"
)

// Connect opens a live libvirt connection.
func (c *KVMConnector) Connect(ctx context.Context) error {
	_ = ctx
	conn, err := libvirt.NewConnect(buildLibvirtURI(c.config.Address))
	if err != nil {
		return fmt.Errorf("kvm: connect: %w", err)
	}
	c.connected = true
	c.config.Address = buildLibvirtURI(c.config.Address)
	c.liveConn = conn
	return nil
}

// Discover retrieves domain, storage pool, and network inventory from a live libvirt daemon.
func (c *KVMConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if !c.connected {
		return nil, fmt.Errorf("kvm: not connected, call Connect first")
	}
	return c.discoverLive(ctx)
}

// Close releases a live libvirt connection.
func (c *KVMConnector) Close() error {
	conn := c.libvirtConn()
	if conn == nil {
		c.connected = false
		return nil
	}

	if _, err := conn.Close(); err != nil {
		return fmt.Errorf("kvm: close: %w", err)
	}
	c.liveConn = nil
	c.connected = false
	return nil
}

func buildLibvirtURI(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return "qemu:///system"
	}
	if strings.Contains(address, "://") {
		return address
	}
	return "qemu+tcp://" + address + "/system"
}

func (c *KVMConnector) discoverLive(ctx context.Context) (*models.DiscoveryResult, error) {
	conn := c.libvirtConn()
	if conn == nil {
		return nil, fmt.Errorf("kvm: libvirt connection is unavailable")
	}

	startedAt := time.Now()
	host, _ := conn.GetHostname()
	discoveredAt := time.Now().UTC()

	domains, err := conn.ListAllDomains(0)
	if err != nil {
		return nil, fmt.Errorf("kvm: list domains: %w", err)
	}

	virtualMachines := make([]models.VirtualMachine, 0, len(domains))
	for _, domain := range domains {
		xmlDesc, err := domain.GetXMLDesc(0)
		if err != nil {
			return nil, fmt.Errorf("kvm: get domain XML: %w", err)
		}
		parsed, err := parseDomainXML(xmlDesc)
		if err != nil {
			return nil, err
		}
		state, _, err := domain.GetState()
		if err != nil {
			return nil, fmt.Errorf("kvm: get domain state: %w", err)
		}

		vm := mapDomainToVM(parsed, mapLibvirtDomainState(state), host)
		vm.DiscoveredAt = discoveredAt
		virtualMachines = append(virtualMachines, vm)
	}

	storagePools, err := conn.ListAllStoragePools(0)
	if err != nil {
		return nil, fmt.Errorf("kvm: list storage pools: %w", err)
	}
	datastores := make([]models.DatastoreInfo, 0, len(storagePools))
	for _, pool := range storagePools {
		xmlDesc, err := pool.GetXMLDesc(0)
		if err != nil {
			return nil, fmt.Errorf("kvm: get storage pool XML: %w", err)
		}
		parsed, err := parseStoragePoolXML(xmlDesc)
		if err != nil {
			return nil, err
		}
		info, err := pool.GetInfo()
		if err != nil {
			return nil, fmt.Errorf("kvm: get storage pool info: %w", err)
		}
		name, err := pool.GetName()
		if err != nil {
			return nil, fmt.Errorf("kvm: get storage pool name: %w", err)
		}
		datastores = append(datastores, models.DatastoreInfo{
			ID:         name,
			Name:       name,
			Type:       parsed.Type,
			CapacityMB: int64(info.Capacity / (1024 * 1024)),
			FreeMB:     int64(info.Available / (1024 * 1024)),
			Hosts:      []string{host},
		})
	}

	networks, err := conn.ListAllNetworks(0)
	if err != nil {
		return nil, fmt.Errorf("kvm: list networks: %w", err)
	}
	networkItems := make([]models.NetworkInfo, 0, len(networks))
	for _, network := range networks {
		name, err := network.GetName()
		if err != nil {
			return nil, fmt.Errorf("kvm: get network name: %w", err)
		}
		bridge, _ := network.GetBridgeName()
		uuid, _ := network.GetUUIDString()
		networkItems = append(networkItems, models.NetworkInfo{
			ID:     firstNonEmpty(uuid, name),
			Name:   name,
			Type:   "bridge",
			Switch: bridge,
		})
	}

	return &models.DiscoveryResult{
		Source:       c.config.Address,
		Platform:     models.PlatformKVM,
		VMs:          virtualMachines,
		Networks:     networkItems,
		Datastores:   datastores,
		DiscoveredAt: discoveredAt,
		Duration:     time.Since(startedAt),
	}, nil
}

func mapLibvirtDomainState(state libvirt.DomainState) models.PowerState {
	switch state {
	case libvirt.DOMAIN_RUNNING, libvirt.DOMAIN_BLOCKED:
		return models.PowerOn
	case libvirt.DOMAIN_PAUSED, libvirt.DOMAIN_PMSUSPENDED:
		return models.PowerSuspend
	case libvirt.DOMAIN_SHUTOFF, libvirt.DOMAIN_SHUTDOWN, libvirt.DOMAIN_CRASHED:
		return models.PowerOff
	default:
		return models.PowerUnknown
	}
}

func (c *KVMConnector) libvirtConn() *libvirt.Connect {
	conn, _ := c.liveConn.(*libvirt.Connect)
	return conn
}
