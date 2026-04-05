//go:build soak

package soak

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

type soakConnector struct {
	platform models.Platform
	address  string
	vms      []models.VirtualMachine
}

func (c *soakConnector) Connect(ctx context.Context) error { return nil }

func (c *soakConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	return &models.DiscoveryResult{
		Source:       c.address,
		Platform:     c.platform,
		VMs:          append([]models.VirtualMachine(nil), c.vms...),
		DiscoveredAt: time.Now().UTC(),
	}, nil
}

func (c *soakConnector) Platform() models.Platform { return c.platform }

func (c *soakConnector) Close() error { return nil }

func TestMigrationSoak_LargeWavePlanAndExecution_Completes(t *testing.T) {
	stateStore := store.NewMemoryStore()
	sourceVMs := make([]models.VirtualMachine, 0, 120)
	for i := 0; i < 120; i++ {
		sourceVMs = append(sourceVMs, models.VirtualMachine{
			ID:           fmt.Sprintf("vm-%03d", i),
			Name:         fmt.Sprintf("app-%03d", i),
			Platform:     models.PlatformVMware,
			CPUCount:     2,
			MemoryMB:     4096,
			PowerState:   models.PowerOn,
			DiscoveredAt: time.Now().UTC(),
		})
	}

	orchestrator := migratepkg.NewOrchestrator(
		&soakConnector{platform: models.PlatformVMware, address: "vcsa.lab.local", vms: sourceVMs},
		&soakConnector{platform: models.PlatformProxmox, address: "pve.lab.local"},
		stateStore,
		nil,
	)

	spec := &migratepkg.MigrationSpec{
		Name: "soak",
		Source: migratepkg.SourceSpec{
			Address:  "vcsa.lab.local",
			Platform: models.PlatformVMware,
		},
		Target: migratepkg.TargetSpec{
			Address:  "pve.lab.local",
			Platform: models.PlatformProxmox,
		},
		Workloads: []migratepkg.WorkloadSelector{
			{Match: migratepkg.MatchCriteria{NamePattern: "app-*"}},
		},
		Options: migratepkg.MigrationOptions{
			Parallel: 4,
			Waves: migratepkg.WaveStrategy{
				Size:            25,
				DependencyAware: true,
			},
		},
	}

	state, err := orchestrator.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Phase != migratepkg.PhaseComplete {
		t.Fatalf("state.Phase = %s, want %s", state.Phase, migratepkg.PhaseComplete)
	}

	record, err := stateStore.GetMigration(context.Background(), store.DefaultTenantID, state.ID)
	if err != nil {
		t.Fatalf("GetMigration() error = %v", err)
	}

	var persisted migratepkg.MigrationState
	if err := json.Unmarshal(record.RawJSON, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(persisted.Checkpoints) == 0 {
		t.Fatal("persisted checkpoints are empty")
	}
}

var _ connectors.Connector = (*soakConnector)(nil)
