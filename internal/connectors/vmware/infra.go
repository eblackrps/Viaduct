package vmware

import (
	"context"
	"fmt"
	"sort"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func discoverNetworks(ctx context.Context, client *govmomi.Client) ([]models.NetworkInfo, error) {
	manager := view.NewManager(client.Client)

	standardView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"Network"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create standard network view: %w", err)
	}
	defer func() {
		_ = standardView.Destroy(ctx)
	}()

	var standardNetworks []mo.Network
	if err := standardView.Retrieve(ctx, []string{"Network"}, []string{"name", "summary", "parent"}, &standardNetworks); err != nil {
		return nil, fmt.Errorf("vmware: retrieve standard networks: %w", err)
	}

	networks := make([]models.NetworkInfo, 0, len(standardNetworks))
	for _, network := range standardNetworks {
		networks = append(networks, models.NetworkInfo{
			ID:     network.Reference().Value,
			Name:   network.Name,
			Type:   "standard",
			VlanID: 0,
			Switch: managedObjectNamePtr(ctx, client, network.Parent),
		})
	}

	distributedView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"DistributedVirtualPortgroup"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create distributed network view: %w", err)
	}
	defer func() {
		_ = distributedView.Destroy(ctx)
	}()

	var distributedNetworks []mo.DistributedVirtualPortgroup
	if err := distributedView.Retrieve(ctx, []string{"DistributedVirtualPortgroup"}, []string{"name", "config", "parent"}, &distributedNetworks); err != nil {
		return nil, fmt.Errorf("vmware: retrieve distributed networks: %w", err)
	}

	for _, network := range distributedNetworks {
		networks = append(networks, models.NetworkInfo{
			ID:     network.Reference().Value,
			Name:   network.Name,
			Type:   "distributed",
			VlanID: distributedVLANID(network.Config),
			Switch: managedObjectNamePtr(ctx, client, network.Parent),
		})
	}

	sort.Slice(networks, func(i, j int) bool {
		if networks[i].Type == networks[j].Type {
			return networks[i].Name < networks[j].Name
		}

		return networks[i].Type < networks[j].Type
	})

	return networks, nil
}

func discoverDatastores(ctx context.Context, client *govmomi.Client) ([]models.DatastoreInfo, error) {
	manager := view.NewManager(client.Client)
	containerView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create datastore view: %w", err)
	}
	defer func() {
		_ = containerView.Destroy(ctx)
	}()

	var datastores []mo.Datastore
	if err := containerView.Retrieve(ctx, []string{"Datastore"}, []string{"name", "summary", "host"}, &datastores); err != nil {
		return nil, fmt.Errorf("vmware: retrieve datastores: %w", err)
	}

	results := make([]models.DatastoreInfo, 0, len(datastores))
	for _, datastore := range datastores {
		hosts := make([]string, 0, len(datastore.Host))
		for _, host := range datastore.Host {
			hosts = append(hosts, managedObjectName(ctx, client, host.Key))
		}

		results = append(results, models.DatastoreInfo{
			ID:         datastore.Reference().Value,
			Name:       datastore.Name,
			Type:       datastore.Summary.Type,
			CapacityMB: datastore.Summary.Capacity / (1024 * 1024),
			FreeMB:     datastore.Summary.FreeSpace / (1024 * 1024),
			Hosts:      hosts,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	return results, nil
}

func discoverHosts(ctx context.Context, client *govmomi.Client) ([]models.HostInfo, error) {
	manager := view.NewManager(client.Client)
	containerView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"HostSystem"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create host view: %w", err)
	}
	defer func() {
		_ = containerView.Destroy(ctx)
	}()

	var hosts []mo.HostSystem
	if err := containerView.Retrieve(ctx, []string{"HostSystem"}, []string{"name", "summary", "parent"}, &hosts); err != nil {
		return nil, fmt.Errorf("vmware: retrieve hosts: %w", err)
	}

	results := make([]models.HostInfo, 0, len(hosts))
	for _, host := range hosts {
		results = append(results, models.HostInfo{
			ID:              host.Reference().Value,
			Name:            host.Name,
			Cluster:         managedObjectNamePtr(ctx, client, host.Parent),
			CPUCores:        int(host.Summary.Hardware.NumCpuCores),
			MemoryMB:        int64(host.Summary.Hardware.MemorySize / (1024 * 1024)),
			PowerState:      hostPowerState(host.Summary.Runtime.PowerState),
			ConnectionState: string(host.Summary.Runtime.ConnectionState),
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	return results, nil
}

func discoverClusters(ctx context.Context, client *govmomi.Client) ([]models.ClusterInfo, error) {
	manager := view.NewManager(client.Client)
	containerView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"ClusterComputeResource"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create cluster view: %w", err)
	}
	defer func() {
		_ = containerView.Destroy(ctx)
	}()

	var clusters []mo.ClusterComputeResource
	if err := containerView.Retrieve(ctx, []string{"ClusterComputeResource"}, []string{"name", "host", "summary", "configurationEx"}, &clusters); err != nil {
		return nil, fmt.Errorf("vmware: retrieve clusters: %w", err)
	}

	results := make([]models.ClusterInfo, 0, len(clusters))
	for _, cluster := range clusters {
		totalCPUCores := 0
		totalMemoryMB := int64(0)
		hostNames := make([]string, 0, len(cluster.Host))
		for _, hostRef := range cluster.Host {
			hostNames = append(hostNames, managedObjectName(ctx, client, hostRef))

			var host mo.HostSystem
			if err := property.DefaultCollector(client.Client).RetrieveOne(ctx, hostRef, []string{"summary.hardware"}, &host); err == nil {
				totalCPUCores += int(host.Summary.Hardware.NumCpuCores)
				totalMemoryMB += int64(host.Summary.Hardware.MemorySize / (1024 * 1024))
			}
		}

		haEnabled := false
		drsEnabled := false
		if configuration, ok := cluster.ConfigurationEx.(*types.ClusterConfigInfoEx); ok {
			haEnabled = configuration.DasConfig.Enabled != nil && *configuration.DasConfig.Enabled
			drsEnabled = configuration.DrsConfig.Enabled != nil && *configuration.DrsConfig.Enabled
		}

		results = append(results, models.ClusterInfo{
			ID:            cluster.Reference().Value,
			Name:          cluster.Name,
			Hosts:         hostNames,
			TotalCPUCores: totalCPUCores,
			TotalMemoryMB: totalMemoryMB,
			HAEnabled:     haEnabled,
			DRSEnabled:    drsEnabled,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	return results, nil
}

func discoverResourcePools(ctx context.Context, client *govmomi.Client) ([]models.ResourcePoolInfo, error) {
	manager := view.NewManager(client.Client)
	containerView, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"ResourcePool"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: create resource pool view: %w", err)
	}
	defer func() {
		_ = containerView.Destroy(ctx)
	}()

	var pools []mo.ResourcePool
	if err := containerView.Retrieve(ctx, []string{"ResourcePool"}, []string{"name", "parent", "summary"}, &pools); err != nil {
		return nil, fmt.Errorf("vmware: retrieve resource pools: %w", err)
	}

	results := make([]models.ResourcePoolInfo, 0, len(pools))
	for _, pool := range pools {
		clusterName := resolveClusterFromParent(ctx, client, pool.Parent)

		cpuLimit := int64(-1)
		memoryLimit := int64(-1)
		if summary, ok := pool.Summary.(*types.ResourcePoolSummary); ok {
			if summary.Config.CpuAllocation.Limit != nil {
				cpuLimit = *summary.Config.CpuAllocation.Limit
			}
			if summary.Config.MemoryAllocation.Limit != nil {
				memoryLimit = *summary.Config.MemoryAllocation.Limit
			}
		}

		results = append(results, models.ResourcePoolInfo{
			ID:            pool.Reference().Value,
			Name:          pool.Name,
			Cluster:       clusterName,
			CPULimitMHz:   cpuLimit,
			MemoryLimitMB: memoryLimit,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	return results, nil
}

func hostPowerState(state types.HostSystemPowerState) models.PowerState {
	switch state {
	case types.HostSystemPowerStatePoweredOn:
		return models.PowerOn
	case types.HostSystemPowerStatePoweredOff:
		return models.PowerOff
	case types.HostSystemPowerStateStandBy:
		return models.PowerSuspend
	default:
		return models.PowerUnknown
	}
}

func distributedVLANID(config types.DVPortgroupConfigInfo) int {
	if config.DefaultPortConfig == nil {
		return 0
	}

	portConfig, ok := config.DefaultPortConfig.(*types.VMwareDVSPortSetting)
	if !ok || portConfig.Vlan == nil {
		return 0
	}

	switch vlan := portConfig.Vlan.(type) {
	case *types.VmwareDistributedVirtualSwitchVlanIdSpec:
		return int(vlan.VlanId)
	case *types.VmwareDistributedVirtualSwitchTrunkVlanSpec:
		if len(vlan.VlanId) == 0 {
			return 0
		}

		return int(vlan.VlanId[0].Start)
	default:
		return 0
	}
}

func managedObjectName(ctx context.Context, client *govmomi.Client, ref types.ManagedObjectReference) string {
	if ref.Value == "" {
		return ""
	}

	var entity mo.ManagedEntity
	if err := property.DefaultCollector(client.Client).RetrieveOne(ctx, ref, []string{"name"}, &entity); err != nil {
		return ""
	}

	return entity.Name
}

func managedObjectNamePtr(ctx context.Context, client *govmomi.Client, ref *types.ManagedObjectReference) string {
	if ref == nil {
		return ""
	}

	return managedObjectName(ctx, client, *ref)
}

func resolveClusterFromParent(ctx context.Context, client *govmomi.Client, ref *types.ManagedObjectReference) string {
	if ref == nil || ref.Value == "" {
		return ""
	}

	current := *ref
	for current.Value != "" {
		if current.Type == "ClusterComputeResource" || current.Type == "ComputeResource" {
			return managedObjectName(ctx, client, current)
		}

		var entity mo.ManagedEntity
		if err := property.DefaultCollector(client.Client).RetrieveOne(ctx, current, []string{"parent"}, &entity); err != nil || entity.Parent == nil {
			return ""
		}

		current = *entity.Parent
	}

	return ""
}
