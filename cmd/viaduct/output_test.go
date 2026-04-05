package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/models"
	"gopkg.in/yaml.v3"
)

func TestFormatTable_BasicOutput(t *testing.T) {
	t.Parallel()

	result := &discovery.MergedResult{
		Sources: []models.DiscoveryResult{
			{
				Source:   "vcenter",
				Platform: models.PlatformVMware,
				VMs: []models.VirtualMachine{
					{Name: "web-01", Platform: models.PlatformVMware, PowerState: models.PowerOn, CPUCount: 2, MemoryMB: 2048, Host: "esxi-01", Cluster: "cluster-a"},
				},
			},
			{
				Source:   "proxmox",
				Platform: models.PlatformProxmox,
				VMs: []models.VirtualMachine{
					{Name: "db-01", Platform: models.PlatformProxmox, PowerState: models.PowerOff, CPUCount: 4, MemoryMB: 4096, Host: "pve-01", Cluster: "cluster-b"},
					{Name: "cache-01", Platform: models.PlatformProxmox, PowerState: models.PowerOn, CPUCount: 1, MemoryMB: 1024, Host: "pve-02", Cluster: "cluster-b"},
				},
			},
		},
		TotalVMs:      3,
		TotalCPU:      7,
		TotalMemoryMB: 7168,
		ByPlatform: map[models.Platform]discovery.PlatformSummary{
			models.PlatformVMware:  {VMCount: 1, TotalCPU: 2, TotalMemoryMB: 2048, Source: "vcenter"},
			models.PlatformProxmox: {VMCount: 2, TotalCPU: 5, TotalMemoryMB: 5120, Source: "proxmox"},
		},
	}

	table := FormatTable(result, false)
	for _, want := range []string{"web-01", "db-01", "cache-01", "VMWARE", "PROXMOX"} {
		if !strings.Contains(table, want) {
			t.Fatalf("FormatTable() output missing %q:\n%s", want, table)
		}
	}
}

func TestFormatTable_EmptyResult(t *testing.T) {
	t.Parallel()

	table := FormatTable(&discovery.MergedResult{}, false)
	if !strings.Contains(table, "No VMs discovered") {
		t.Fatalf("FormatTable() = %q, want empty message", table)
	}
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	t.Parallel()

	payload, err := FormatJSON(&discovery.MergedResult{TotalVMs: 2})
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func TestFormatYAML_ValidYAML(t *testing.T) {
	t.Parallel()

	payload, err := FormatYAML(&discovery.MergedResult{TotalVMs: 2})
	if err != nil {
		t.Fatalf("FormatYAML() error = %v", err)
	}

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
}

func TestFormatTable_SummaryLine(t *testing.T) {
	t.Parallel()

	table := FormatTable(&discovery.MergedResult{
		Sources:       []models.DiscoveryResult{{Platform: models.PlatformVMware}},
		TotalVMs:      2,
		TotalCPU:      6,
		TotalMemoryMB: 8192,
		ByPlatform: map[models.Platform]discovery.PlatformSummary{
			models.PlatformVMware: {VMCount: 2, TotalCPU: 6, TotalMemoryMB: 8192, Source: "vcenter"},
		},
	}, false)

	if !strings.Contains(table, "Discovered 2 VMs across 1 sources") {
		t.Fatalf("FormatTable() summary missing expected text:\n%s", table)
	}
}
