package nutanix

import (
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func mapNutanixVM(raw map[string]interface{}) models.VirtualMachine {
	spec := mapValue(raw, "spec")
	status := mapValue(raw, "status")
	metadata := mapValue(raw, "metadata")
	specResources := mapValue(spec, "resources")
	statusResources := mapValue(status, "resources")

	name := stringValue(raw, "status.name")
	if name == "" {
		name = stringValue(raw, "spec.name")
	}

	uuid := stringValue(metadata, "uuid")
	if uuid == "" {
		uuid = stringValue(metadata, "id")
	}

	numSockets := intValue(specResources["num_sockets"])
	coresPerSocket := intValue(specResources["num_vcpus_per_socket"])
	cpuCount := numSockets * coresPerSocket
	if cpuCount == 0 {
		cpuCount = intValue(statusResources["num_vcpus"])
	}

	guestOS := stringValue(specResources, "guest_os")
	if guestOS == "" {
		guestOS = stringValue(statusResources, "guest_os")
	}

	createdAt := time.Time{}
	if created := stringValue(metadata, "creation_time"); created != "" {
		parsed, err := time.Parse(time.RFC3339, created)
		if err == nil {
			createdAt = parsed
		}
	}

	return models.VirtualMachine{
		ID:           uuid,
		Name:         name,
		Platform:     models.PlatformNutanix,
		PowerState:   mapNutanixPowerState(stringValue(statusResources, "power_state")),
		CPUCount:     cpuCount,
		MemoryMB:     int(int64Value(specResources["memory_size_mib"])),
		Disks:        mapNutanixDisks(specResources),
		NICs:         mapNutanixNICs(specResources),
		GuestOS:      guestOS,
		Host:         referenceName(statusResources, "host_reference"),
		Cluster:      referenceName(statusResources, "cluster_reference"),
		Tags:         mapCategories(metadata),
		CreatedAt:    createdAt,
		DiscoveredAt: time.Now().UTC(),
		SourceRef:    uuid,
	}
}

func mapNutanixPowerState(state string) models.PowerState {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "ON":
		return models.PowerOn
	case "OFF":
		return models.PowerOff
	case "SUSPENDED":
		return models.PowerSuspend
	default:
		return models.PowerUnknown
	}
}

func mapNutanixDisks(resources map[string]interface{}) []models.Disk {
	disks := listValue(resources["disk_list"])
	items := make([]models.Disk, 0, len(disks))
	for _, disk := range disks {
		storageConfig := mapValue(disk, "storage_config")
		deviceProperties := mapValue(disk, "device_properties")
		diskAddress := mapValue(deviceProperties, "disk_address")
		items = append(items, models.Disk{
			ID:             firstNonEmptyString(stringValue(disk, "uuid"), stringValue(diskAddress, "device_index")),
			Name:           firstNonEmptyString(referenceName(disk, "data_source_reference"), stringValue(diskAddress, "disk_label"), "disk"),
			SizeMB:         int(int64Value(disk["disk_size_bytes"]) / (1024 * 1024)),
			Thin:           true,
			StorageBackend: referenceName(storageConfig, "storage_container_reference"),
		})
	}
	return items
}

func mapNutanixNICs(resources map[string]interface{}) []models.NIC {
	nics := listValue(resources["nic_list"])
	items := make([]models.NIC, 0, len(nics))
	for _, nic := range nics {
		ipEndpoints := listValue(nic["ip_endpoint_list"])
		ipAddresses := make([]string, 0, len(ipEndpoints))
		for _, endpoint := range ipEndpoints {
			if ip := stringValue(endpoint, "ip"); ip != "" {
				ipAddresses = append(ipAddresses, ip)
			}
		}
		items = append(items, models.NIC{
			ID:          firstNonEmptyString(stringValue(nic, "uuid"), stringValue(nic, "nic_type")),
			Name:        firstNonEmptyString(stringValue(nic, "uuid"), stringValue(nic, "nic_type"), "nic"),
			MACAddress:  stringValue(nic, "mac_address"),
			Network:     referenceName(nic, "subnet_reference"),
			Connected:   true,
			IPAddresses: ipAddresses,
		})
	}
	return items
}

func mapCategories(metadata map[string]interface{}) map[string]string {
	if metadata == nil {
		return nil
	}

	categories := mapValue(metadata, "categories")
	if len(categories) == 0 {
		categories = mapValue(metadata, "categories_mapping")
	}
	if len(categories) == 0 {
		return nil
	}

	tags := make(map[string]string, len(categories))
	for key, value := range categories {
		switch typed := value.(type) {
		case string:
			tags[key] = typed
		case []interface{}:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			tags[key] = strings.Join(parts, ",")
		default:
			tags[key] = fmt.Sprintf("%v", typed)
		}
	}
	return tags
}

func referenceName(raw map[string]interface{}, key string) string {
	return stringValue(mapValue(raw, key), "name")
}

func mapValue(raw map[string]interface{}, path string) map[string]interface{} {
	current := raw
	for _, segment := range strings.Split(path, ".") {
		if current == nil {
			return nil
		}
		next, ok := current[segment].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func stringValue(raw map[string]interface{}, path string) string {
	if raw == nil {
		return ""
	}
	if !strings.Contains(path, ".") {
		value, _ := raw[path].(string)
		return value
	}

	parts := strings.Split(path, ".")
	current := raw
	for index, part := range parts {
		if index == len(parts)-1 {
			value, _ := current[part].(string)
			return value
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func int64Value(value interface{}) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func intValue(value interface{}) int {
	return int(int64Value(value))
}

func listValue(value interface{}) []map[string]interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	output := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]interface{}); ok {
			output = append(output, mapped)
		}
	}
	return output
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
