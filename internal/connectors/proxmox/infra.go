package proxmox

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

func discoverNetworks(ctx context.Context, client *ProxmoxClient) ([]models.NetworkInfo, error) {
	nodes, err := listNodes(ctx, client)
	if err != nil {
		return nil, err
	}

	networks := make([]models.NetworkInfo, 0)
	for _, node := range nodes {
		var payload []map[string]interface{}
		if err := client.decode(ctx, fmt.Sprintf("/nodes/%s/network", node), &payload); err != nil {
			return nil, fmt.Errorf("proxmox: discover networks for %s: %w", node, err)
		}

		for _, item := range payload {
			ifaceType := strings.ToLower(stringField(item, "type", ""))
			if ifaceType == "" {
				continue
			}

			vlanID := intField(item, "vlan-id", 0)
			if vlanID == 0 {
				if parts := strings.Split(stringField(item, "iface", ""), "."); len(parts) == 2 {
					if parsed, err := strconv.Atoi(parts[1]); err == nil {
						vlanID = parsed
					}
				}
			}

			networks = append(networks, models.NetworkInfo{
				ID:     fmt.Sprintf("%s/%s", node, stringField(item, "iface", "")),
				Name:   stringField(item, "iface", ""),
				Type:   ifaceType,
				VlanID: vlanID,
				Switch: stringField(item, "bridge_ports", node),
			})
		}
	}

	return networks, nil
}

func discoverStorage(ctx context.Context, client *ProxmoxClient) ([]models.DatastoreInfo, error) {
	nodes, err := listNodes(ctx, client)
	if err != nil {
		return nil, err
	}

	var storageList []map[string]interface{}
	if err := client.decode(ctx, "/storage", &storageList); err != nil {
		return nil, fmt.Errorf("proxmox: discover storage list: %w", err)
	}

	byName := map[string]*models.DatastoreInfo{}
	for _, item := range storageList {
		name := stringField(item, "storage", "")
		if name == "" {
			continue
		}

		entry, ok := byName[name]
		if !ok {
			entry = &models.DatastoreInfo{
				ID:   name,
				Name: name,
				Type: stringField(item, "type", "unknown"),
			}
			byName[name] = entry
		}

		if host := stringField(item, "node", ""); host != "" {
			entry.Hosts = appendIfMissing(entry.Hosts, host)
		}

		for _, node := range nodes {
			var status map[string]interface{}
			if err := client.decode(ctx, fmt.Sprintf("/nodes/%s/storage/%s/status", node, name), &status); err != nil {
				continue
			}

			entry.Hosts = appendIfMissing(entry.Hosts, node)
			if entry.CapacityMB == 0 {
				entry.CapacityMB = int64Field(status, "total", 0) / (1024 * 1024)
			}
			if entry.FreeMB == 0 {
				entry.FreeMB = int64Field(status, "avail", 0) / (1024 * 1024)
			}
		}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	datastores := make([]models.DatastoreInfo, 0, len(names))
	for _, name := range names {
		datastores = append(datastores, *byName[name])
	}

	return datastores, nil
}

func discoverNodes(ctx context.Context, client *ProxmoxClient) ([]models.HostInfo, error) {
	nodes, err := listNodes(ctx, client)
	if err != nil {
		return nil, err
	}

	hosts := make([]models.HostInfo, 0, len(nodes))
	for _, node := range nodes {
		var status map[string]interface{}
		if err := client.decode(ctx, fmt.Sprintf("/nodes/%s/status", node), &status); err != nil {
			return nil, fmt.Errorf("proxmox: discover node status for %s: %w", node, err)
		}

		cpuInfo, _ := status["cpuinfo"].(map[string]interface{})
		hosts = append(hosts, models.HostInfo{
			ID:              node,
			Name:            node,
			CPUCores:        intField(cpuInfo, "cpus", intField(status, "cpus", 0)),
			MemoryMB:        int64Field(status, "memory_total", int64Field(status, "memory", 0)/(1024*1024)),
			PowerState:      models.PowerOn,
			ConnectionState: stringField(status, "status", stringField(status, "online", "unknown")),
		})
	}

	return hosts, nil
}

func listNodes(ctx context.Context, client *ProxmoxClient) ([]string, error) {
	var payload []map[string]interface{}
	if err := client.decode(ctx, "/nodes", &payload); err != nil {
		return nil, fmt.Errorf("proxmox: list nodes: %w", err)
	}

	nodes := make([]string, 0, len(payload))
	for _, item := range payload {
		name := stringField(item, "node", "")
		if name != "" {
			nodes = append(nodes, name)
		}
	}

	sort.Strings(nodes)

	return nodes, nil
}

func appendIfMissing(items []string, candidate string) []string {
	for _, item := range items {
		if item == candidate {
			return items
		}
	}

	return append(items, candidate)
}
