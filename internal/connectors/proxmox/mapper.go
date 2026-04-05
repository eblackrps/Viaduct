package proxmox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

var (
	diskKeyPattern = regexp.MustCompile(`^(?:scsi|virtio|ide|sata|unused)\d+$`)
	nicKeyPattern  = regexp.MustCompile(`^net\d+$`)
)

func mapQemuVM(node string, vmStatus map[string]interface{}, vmConfig map[string]interface{}) models.VirtualMachine {
	return models.VirtualMachine{
		ID:           stringField(vmStatus, "vmid", stringField(vmConfig, "vmid", "")),
		Name:         stringField(vmStatus, "name", stringField(vmConfig, "name", "")),
		Platform:     models.PlatformProxmox,
		PowerState:   mapProxmoxPowerState(stringField(vmStatus, "status", "")),
		CPUCount:     intField(vmConfig, "cores", intField(vmStatus, "cpus", 0)),
		MemoryMB:     intField(vmConfig, "memory", bytesToMB(int64Field(vmStatus, "maxmem", 0))),
		Disks:        parseProxmoxDisks(vmConfig),
		NICs:         parseProxmoxNICs(vmConfig),
		GuestOS:      stringField(vmConfig, "ostype", stringField(vmStatus, "template", "")),
		Host:         node,
		CreatedAt:    timeField(vmStatus, "uptime"),
		DiscoveredAt: timeField(vmStatus, ""),
		SourceRef:    fmt.Sprintf("%s/qemu/%s", node, stringField(vmStatus, "vmid", stringField(vmConfig, "vmid", ""))),
	}
}

func mapLXCContainer(node string, ctStatus map[string]interface{}, ctConfig map[string]interface{}) models.VirtualMachine {
	name := stringField(ctStatus, "name", stringField(ctConfig, "hostname", ""))
	if name == "" {
		name = stringField(ctConfig, "hostname", stringField(ctStatus, "vmid", ""))
	}

	return models.VirtualMachine{
		ID:         stringField(ctStatus, "vmid", stringField(ctConfig, "vmid", "")),
		Name:       name,
		Platform:   models.PlatformProxmox,
		PowerState: mapProxmoxPowerState(stringField(ctStatus, "status", "")),
		CPUCount:   intField(ctConfig, "cores", intField(ctStatus, "cpus", 0)),
		MemoryMB:   intField(ctConfig, "memory", bytesToMB(int64Field(ctStatus, "maxmem", 0))),
		Disks:      parseProxmoxDisks(ctConfig),
		NICs:       parseProxmoxNICs(ctConfig),
		GuestOS:    "lxc",
		Host:       node,
		SourceRef:  fmt.Sprintf("%s/lxc/%s", node, stringField(ctStatus, "vmid", stringField(ctConfig, "vmid", ""))),
	}
}

func parseProxmoxDisks(config map[string]interface{}) []models.Disk {
	keys := make([]string, 0, len(config))
	for key := range config {
		if key == "rootfs" || diskKeyPattern.MatchString(key) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	disks := make([]models.Disk, 0, len(keys))
	for _, key := range keys {
		raw, ok := config[key].(string)
		if !ok || raw == "" {
			continue
		}

		parts := strings.Split(raw, ",")
		storage := ""
		volume := key
		if len(parts) > 0 {
			storagePart := parts[0]
			before, after, found := strings.Cut(storagePart, ":")
			if found {
				storage = before
				volume = after
			} else {
				volume = storagePart
			}
		}

		sizeMB := 0
		thin := false
		for _, part := range parts[1:] {
			name, value, found := strings.Cut(part, "=")
			if !found {
				continue
			}

			switch strings.TrimSpace(name) {
			case "size":
				sizeMB = sizeToMB(value)
			case "thin":
				thin = value == "1" || strings.EqualFold(value, "true")
			}
		}

		disks = append(disks, models.Disk{
			ID:             key,
			Name:           volume,
			SizeMB:         sizeMB,
			Thin:           thin,
			StorageBackend: storage,
		})
	}

	return disks
}

func parseProxmoxNICs(config map[string]interface{}) []models.NIC {
	keys := make([]string, 0, len(config))
	for key := range config {
		if nicKeyPattern.MatchString(key) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	nics := make([]models.NIC, 0, len(keys))
	for _, key := range keys {
		raw, ok := config[key].(string)
		if !ok || raw == "" {
			continue
		}

		attributes := parseAttributeList(raw)
		modelPart := strings.SplitN(raw, ",", 2)[0]
		model, mac, _ := strings.Cut(modelPart, "=")
		if hwAddr := attributes["hwaddr"]; hwAddr != "" {
			mac = hwAddr
		}
		if name := attributes["name"]; name != "" {
			model = name
		}
		network := attributes["bridge"]
		if network == "" {
			network = attributes["network"]
		}

		nics = append(nics, models.NIC{
			ID:         key,
			Name:       model,
			MACAddress: strings.ToUpper(mac),
			Network:    network,
			Connected:  attributes["link_down"] != "1",
		})
	}

	return nics
}

func mapProxmoxPowerState(status string) models.PowerState {
	switch strings.ToLower(status) {
	case "running":
		return models.PowerOn
	case "stopped":
		return models.PowerOff
	case "paused", "suspended":
		return models.PowerSuspend
	default:
		return models.PowerUnknown
	}
}

func parseAttributeList(raw string) map[string]string {
	attrs := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		name, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}

		attrs[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}

	return attrs
}

func sizeToMB(raw string) int {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	if raw == "" {
		return 0
	}

	multiplier := float64(1)
	switch {
	case strings.HasSuffix(raw, "T"):
		multiplier = 1024 * 1024
		raw = strings.TrimSuffix(raw, "T")
	case strings.HasSuffix(raw, "G"):
		multiplier = 1024
		raw = strings.TrimSuffix(raw, "G")
	case strings.HasSuffix(raw, "M"):
		raw = strings.TrimSuffix(raw, "M")
	case strings.HasSuffix(raw, "K"):
		multiplier = 1.0 / 1024
		raw = strings.TrimSuffix(raw, "K")
	case strings.HasSuffix(raw, "B"):
		multiplier = 1.0 / (1024 * 1024)
		raw = strings.TrimSuffix(raw, "B")
	}

	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}

	return int(value * multiplier)
}

func stringField(data map[string]interface{}, key, fallback string) string {
	if key == "" {
		return fallback
	}

	value, ok := data[key]
	if !ok || value == nil {
		return fallback
	}

	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fallback
	}
}

func intField(data map[string]interface{}, key string, fallback int) int {
	return int(int64Field(data, key, int64(fallback)))
}

func int64Field(data map[string]interface{}, key string, fallback int64) int64 {
	value, ok := data[key]
	if !ok || value == nil {
		return fallback
	}

	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err == nil {
			return parsed
		}
	}

	return fallback
}

func bytesToMB(raw int64) int {
	if raw <= 0 {
		return 0
	}

	return int(raw / (1024 * 1024))
}

func timeField(data map[string]interface{}, key string) time.Time {
	if key == "" {
		return time.Time{}
	}

	seconds := int64Field(data, key, 0)
	if seconds <= 0 {
		return time.Time{}
	}

	return time.Unix(seconds, 0).UTC()
}
