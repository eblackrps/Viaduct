package hyperv

import (
	"path/filepath"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

type hyperVVM struct {
	Name           string `json:"Name"`
	VMID           string `json:"VMId"`
	State          string `json:"State"`
	ProcessorCount int    `json:"ProcessorCount"`
	MemoryAssigned int64  `json:"MemoryAssigned"`
	Path           string `json:"Path"`
	Generation     int    `json:"Generation"`
}

type hyperVDisk struct {
	VMName         string `json:"VMName"`
	Path           string `json:"Path"`
	DiskNumber     int    `json:"DiskNumber"`
	ControllerType string `json:"ControllerType"`
}

type hyperVAdapter struct {
	VMName      string   `json:"VMName"`
	MacAddress  string   `json:"MacAddress"`
	SwitchName  string   `json:"SwitchName"`
	Connected   bool     `json:"Connected"`
	IPAddresses []string `json:"IPAddresses"`
}

func mapHyperVPowerState(state string) models.PowerState {
	switch strings.ToLower(state) {
	case "running":
		return models.PowerOn
	case "off", "stopped":
		return models.PowerOff
	case "saved", "paused":
		return models.PowerSuspend
	default:
		return models.PowerUnknown
	}
}

func mapHyperVDisks(disks []hyperVDisk) []models.Disk {
	mapped := make([]models.Disk, 0, len(disks))
	for _, disk := range disks {
		mapped = append(mapped, models.Disk{
			ID:             disk.Path,
			Name:           filepath.Base(disk.Path),
			SizeMB:         0,
			Thin:           strings.EqualFold(filepath.Ext(disk.Path), ".vhdx"),
			StorageBackend: filepath.Dir(disk.Path),
		})
	}
	return mapped
}

func mapHyperVNICs(adapters []hyperVAdapter) []models.NIC {
	mapped := make([]models.NIC, 0, len(adapters))
	for _, adapter := range adapters {
		mapped = append(mapped, models.NIC{
			ID:          adapter.MacAddress,
			Name:        adapter.VMName + "-" + adapter.SwitchName,
			MACAddress:  adapter.MacAddress,
			Network:     adapter.SwitchName,
			Connected:   adapter.Connected,
			IPAddresses: append([]string(nil), adapter.IPAddresses...),
		})
	}
	return mapped
}
