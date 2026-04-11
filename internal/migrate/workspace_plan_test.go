package migrate

import (
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestBuildStateFromInventory_SelectedWorkloadsPersisted_Expected(t *testing.T) {
	t.Parallel()

	state, err := BuildStateFromInventory(&MigrationSpec{
		Name: "pilot-plan",
		Source: SourceSpec{
			Address:  "examples/lab/kvm",
			Platform: models.PlatformKVM,
		},
		Target: TargetSpec{
			Address:        "https://pilot-proxmox.local:8006/api2/json",
			Platform:       models.PlatformProxmox,
			DefaultHost:    "pve-node-01",
			DefaultStorage: "local-lvm",
		},
		Workloads: []WorkloadSelector{
			{
				Match: MatchCriteria{NamePattern: "regex:^web-01$"},
				Overrides: WorkloadOverrides{
					NetworkMap: map[string]string{"default": "vmbr0"},
				},
			},
		},
		Options: MigrationOptions{
			Parallel: 1,
			Approval: ApprovalGate{Required: true},
		},
	}, &models.DiscoveryResult{
		Source:       "lab-kvm",
		Platform:     models.PlatformKVM,
		DiscoveredAt: time.Date(2026, time.April, 10, 13, 0, 0, 0, time.UTC),
		VMs: []models.VirtualMachine{
			{
				ID:         "vm-1",
				Name:       "web-01",
				Platform:   models.PlatformKVM,
				CPUCount:   2,
				MemoryMB:   4096,
				PowerState: models.PowerOn,
			},
			{
				ID:         "vm-2",
				Name:       "db-01",
				Platform:   models.PlatformKVM,
				CPUCount:   4,
				MemoryMB:   8192,
				PowerState: models.PowerOn,
			},
		},
	}, "migration-1", time.Date(2026, time.April, 10, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BuildStateFromInventory() error = %v", err)
	}
	if state.ID != "migration-1" || state.Phase != PhasePlan {
		t.Fatalf("unexpected migration state: %#v", state)
	}
	if state.Plan == nil || state.Plan.TotalWorkloads != 1 {
		t.Fatalf("unexpected migration plan: %#v", state.Plan)
	}
	if len(state.Workloads) != 1 || state.Workloads[0].VM.Name != "web-01" {
		t.Fatalf("unexpected workloads: %#v", state.Workloads)
	}
	if !state.PendingApproval {
		t.Fatal("PendingApproval = false, want true")
	}
}
