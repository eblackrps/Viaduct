package vmware

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// VMwareConnector is the VMware vCenter implementation of the connector interface.
type VMwareConnector struct {
	config connectors.Config
	client *govmomi.Client
	ctx    context.Context
}

// NewVMwareConnector creates a VMware connector with the provided configuration.
func NewVMwareConnector(cfg connectors.Config) *VMwareConnector {
	return &VMwareConnector{config: cfg}
}

// Connect authenticates to vCenter and stores the resulting govmomi client.
func (c *VMwareConnector) Connect(ctx context.Context) error {
	endpoint, err := soap.ParseURL(buildVCenterURL(c.config))
	if err != nil {
		return fmt.Errorf("vmware: parse endpoint: %w", err)
	}

	client, err := govmomi.NewClient(ctx, endpoint, c.config.Insecure)
	if err != nil {
		return fmt.Errorf("vmware: connect: %w", err)
	}

	c.client = client
	c.ctx = ctx
	return nil
}

// Discover retrieves VMware VM and infrastructure inventory and normalizes it into the shared schema.
func (c *VMwareConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("vmware: not connected, call Connect first")
	}

	startedAt := time.Now()
	viewManager := view.NewManager(c.client.Client)
	containerView, err := viewManager.CreateContainerView(ctx, c.client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create VM container view: %w", err)
	}
	defer func() {
		_ = containerView.Destroy(ctx)
	}()

	var vms []mo.VirtualMachine
	properties := []string{
		"name",
		"config",
		"guest",
		"runtime.powerState",
		"runtime.host",
		"summary.config.vmPathName",
		"resourcePool",
		"parent",
	}
	if err := containerView.Retrieve(ctx, []string{"VirtualMachine"}, properties, &vms); err != nil {
		return nil, fmt.Errorf("vmware: retrieve VMs: %w", err)
	}

	discoveredAt := time.Now().UTC()
	virtualMachines := make([]models.VirtualMachine, 0, len(vms))
	for _, vm := range vms {
		cpuCount := 0
		memoryMB := 0
		devices := []types.BaseVirtualDevice(nil)
		guestOS := ""
		createdAt := time.Time{}

		if vm.Config != nil {
			cpuCount = int(vm.Config.Hardware.NumCPU)
			memoryMB = int(vm.Config.Hardware.MemoryMB)
			devices = vm.Config.Hardware.Device
			guestOS = vm.Config.GuestFullName
			if vm.Config.CreateDate != nil {
				createdAt = *vm.Config.CreateDate
			}
		}

		if vm.Guest != nil && vm.Guest.GuestFullName != "" {
			guestOS = vm.Guest.GuestFullName
		}

		hostName := ""
		clusterName := ""
		if vm.Runtime.Host != nil {
			hostName = managedObjectName(ctx, c.client, *vm.Runtime.Host)
			clusterName = extractClusterName(ctx, c.client, *vm.Runtime.Host)
		}

		resourcePool := ""
		if vm.ResourcePool != nil {
			resourcePool = managedObjectName(ctx, c.client, *vm.ResourcePool)
		}

		guestNics := []types.GuestNicInfo(nil)
		if vm.Guest != nil {
			guestNics = vm.Guest.Net
		}

		virtualMachines = append(virtualMachines, models.VirtualMachine{
			ID:           vm.Reference().Value,
			Name:         vm.Name,
			Platform:     models.PlatformVMware,
			PowerState:   mapPowerState(vm.Runtime.PowerState),
			CPUCount:     cpuCount,
			MemoryMB:     memoryMB,
			Disks:        mapDisks(devices),
			NICs:         mapNICs(devices, guestNics),
			GuestOS:      guestOS,
			Host:         hostName,
			Cluster:      clusterName,
			ResourcePool: resourcePool,
			Folder:       extractFolderPath(ctx, c.client, vm),
			CreatedAt:    createdAt,
			DiscoveredAt: discoveredAt,
			SourceRef:    vm.Reference().String(),
		})
	}

	networks, err := discoverNetworks(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("vmware: discover networks: %w", err)
	}

	datastores, err := discoverDatastores(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("vmware: discover datastores: %w", err)
	}

	hosts, err := discoverHosts(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("vmware: discover hosts: %w", err)
	}

	clusters, err := discoverClusters(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("vmware: discover clusters: %w", err)
	}

	resourcePools, err := discoverResourcePools(ctx, c.client)
	if err != nil {
		return nil, fmt.Errorf("vmware: discover resource pools: %w", err)
	}

	return &models.DiscoveryResult{
		Source:        c.config.Address,
		Platform:      models.PlatformVMware,
		VMs:           virtualMachines,
		Networks:      networks,
		Datastores:    datastores,
		Hosts:         hosts,
		Clusters:      clusters,
		ResourcePools: resourcePools,
		DiscoveredAt:  discoveredAt,
		Duration:      time.Since(startedAt),
	}, nil
}

// Platform returns the VMware platform identifier.
func (c *VMwareConnector) Platform() models.Platform {
	return models.PlatformVMware
}

// Close logs out of vCenter and clears the cached client reference.
func (c *VMwareConnector) Close() error {
	if c.client == nil {
		return nil
	}

	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if err := c.client.Logout(ctx); err != nil {
		return fmt.Errorf("vmware: close: %w", err)
	}

	c.client = nil
	return nil
}

func buildVCenterURL(cfg connectors.Config) string {
	address := strings.TrimSpace(cfg.Address)
	if address == "" {
		return "https:///sdk"
	}

	if !strings.Contains(address, "://") {
		address = "https://" + address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return address
	}

	if cfg.Port != 0 {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(cfg.Port))
	}

	if parsed.User == nil && cfg.Username != "" {
		parsed.User = url.UserPassword(cfg.Username, cfg.Password)
	}

	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/sdk"
	}

	return parsed.String()
}

var _ connectors.Connector = (*VMwareConnector)(nil)
