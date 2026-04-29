package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/eblackrps/viaduct/internal/api"
	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/connectors/kvm"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

const (
	e2eTenantID             = "tenant-e2e"
	e2eTenantKey            = "tenant-e2e-key"
	e2eServiceAccountID     = "sa-e2e"
	e2eServiceAccountName   = "E2E Operator"
	e2eServiceAccountKey    = "sa-e2e-key"
	e2eWorkspaceID          = "workspace-e2e"
	e2eWorkspaceSourceID    = "source-e2e-lab"
	e2eWorkspaceName        = "E2E Lab Workspace"
	e2eSavedMigrationID     = "migration-e2e-plan"
	e2eCompletedMigrationID = "migration-e2e-complete"
)

func main() {
	var (
		host         string
		port         int
		webDir       string
		localRuntime bool
	)

	flag.StringVar(&host, "host", "127.0.0.1", "Host interface to bind")
	flag.IntVar(&port, "port", 4173, "Port to bind")
	flag.StringVar(&webDir, "web-dir", filepath.Join("web", "dist"), "Path to built dashboard assets")
	flag.BoolVar(&localRuntime, "local-runtime", false, "Seed the fixture under the default tenant and enable keyless local sessions")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, host, port, webDir, localRuntime); err != nil {
		fmt.Fprintf(os.Stderr, "e2e server: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, host string, port int, webDir string, localRuntime bool) error {
	stateStore := store.NewMemoryStore()
	if err := seedState(ctx, stateStore, localRuntime); err != nil {
		return err
	}

	server, err := api.NewServer(nil, stateStore, port, nil)
	if err != nil {
		return fmt.Errorf("create api server: %w", err)
	}
	server.SetBuildInfo("e2e", "fixture", time.Now().UTC().Format(time.RFC3339))
	server.SetBindHost(host)
	server.SetDashboardDir(resolvePath(webDir))
	server.SetLocalRuntimeMode(localRuntime)
	return server.Start(ctx)
}

func seedState(ctx context.Context, stateStore store.Store, localRuntime bool) error {
	now := time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)
	tenantID := e2eTenantID
	if localRuntime {
		tenantID = store.DefaultTenantID
	} else {
		tenant := models.Tenant{
			ID:        e2eTenantID,
			Name:      "E2E Tenant",
			APIKey:    e2eTenantKey,
			CreatedAt: now.Add(-2 * time.Hour),
			Active:    true,
			Quotas: models.TenantQuota{
				RequestsPerMinute: 5000,
				MaxSnapshots:      25,
				MaxMigrations:     25,
			},
			ServiceAccounts: []models.ServiceAccount{
				{
					ID:     e2eServiceAccountID,
					Name:   e2eServiceAccountName,
					APIKey: e2eServiceAccountKey,
					Role:   models.TenantRoleAdmin,
					Permissions: []models.TenantPermission{
						models.TenantPermissionInventoryRead,
						models.TenantPermissionReportsRead,
						models.TenantPermissionLifecycleRead,
						models.TenantPermissionMigrationManage,
						models.TenantPermissionTenantRead,
						models.TenantPermissionTenantManage,
					},
					Active:        true,
					CreatedAt:     now.Add(-90 * time.Minute),
					LastRotatedAt: now.Add(-45 * time.Minute),
				},
			},
		}
		if err := stateStore.CreateTenant(ctx, tenant); err != nil {
			return fmt.Errorf("seed tenant: %w", err)
		}
	}

	discoveryResult, err := discoverFixtures()
	if err != nil {
		return err
	}
	discoveryResult.DiscoveredAt = now.Add(-30 * time.Minute)
	discoveryResult.Duration = 850 * time.Millisecond

	snapshotID, err := stateStore.SaveDiscovery(ctx, tenantID, discoveryResult)
	if err != nil {
		return fmt.Errorf("seed discovery snapshot: %w", err)
	}

	olderSnapshot := cloneDiscoveryResult(discoveryResult)
	olderSnapshot.DiscoveredAt = now.Add(-3 * time.Hour)
	if _, err := stateStore.SaveDiscovery(ctx, tenantID, olderSnapshot); err != nil {
		return fmt.Errorf("seed historical discovery snapshot: %w", err)
	}

	if err := seedWorkspace(ctx, stateStore, tenantID, now, snapshotID, discoveryResult); err != nil {
		return err
	}
	if err := seedMigrations(ctx, stateStore, tenantID, now, discoveryResult); err != nil {
		return err
	}

	return nil
}

func discoverFixtures() (*models.DiscoveryResult, error) {
	fixturePath := resolvePath(filepath.Join("examples", "lab", "kvm"))
	connector := kvm.NewKVMConnector(connectors.Config{Address: fixturePath})
	if err := connector.Connect(context.Background()); err != nil {
		return nil, fmt.Errorf("connect kvm fixtures: %w", err)
	}
	defer connector.Close()

	result, err := connector.Discover(context.Background())
	if err != nil {
		return nil, fmt.Errorf("discover kvm fixtures: %w", err)
	}
	return result, nil
}

func seedWorkspace(ctx context.Context, stateStore store.Store, tenantID string, now time.Time, snapshotID string, discoveryResult *models.DiscoveryResult) error {
	workspace := models.PilotWorkspace{
		ID:          e2eWorkspaceID,
		TenantID:    tenantID,
		Name:        e2eWorkspaceName,
		Description: "Browser fixture workspace for the Viaduct operator console",
		Status:      models.PilotWorkspaceStatusDiscovered,
		CreatedAt:   now.Add(-20 * time.Minute),
		UpdatedAt:   now.Add(-15 * time.Minute),
		SourceConnections: []models.WorkspaceSourceConnection{
			{
				ID:               e2eWorkspaceSourceID,
				Name:             "Lab KVM",
				Platform:         models.PlatformKVM,
				Address:          filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
				CredentialRef:    "lab-kvm",
				LastSnapshotID:   snapshotID,
				LastDiscoveredAt: discoveryResult.DiscoveredAt,
			},
		},
		Snapshots: []models.WorkspaceSnapshot{
			{
				SnapshotID:         snapshotID,
				SourceConnectionID: e2eWorkspaceSourceID,
				Source:             discoveryResult.Source,
				Platform:           discoveryResult.Platform,
				VMCount:            len(discoveryResult.VMs),
				DiscoveredAt:       discoveryResult.DiscoveredAt,
			},
		},
		SelectedWorkloadIDs: selectedWorkloadIDs(discoveryResult.VMs),
		TargetAssumptions: models.WorkspaceTargetAssumptions{
			Platform:       models.PlatformKVM,
			Address:        filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
			DefaultHost:    "lab-host-01",
			DefaultStorage: "default",
			DefaultNetwork: "default",
			Notes:          "Local fixture target for Playwright coverage.",
		},
		PlanSettings: models.WorkspacePlanSettings{
			Name:             "e2e-lab-plan",
			Parallel:         2,
			VerifyBoot:       true,
			ApprovalRequired: false,
			WaveSize:         1,
			DependencyAware:  true,
		},
		Notes: []models.WorkspaceNote{
			{
				ID:        "note-e2e",
				Kind:      models.WorkspaceNoteKindOperator,
				Author:    "fixture",
				Body:      "Seeded workspace for end-to-end coverage.",
				CreatedAt: now.Add(-10 * time.Minute),
			},
		},
	}
	if err := stateStore.CreateWorkspace(ctx, tenantID, workspace); err != nil {
		return fmt.Errorf("seed workspace: %w", err)
	}
	return nil
}

func seedMigrations(ctx context.Context, stateStore store.Store, tenantID string, now time.Time, discoveryResult *models.DiscoveryResult) error {
	plannedSpecName := "e2e-kvm-plan"
	plannedState := migratepkg.MigrationState{
		ID:             e2eSavedMigrationID,
		SpecName:       plannedSpecName,
		SourceAddress:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
		SourcePlatform: models.PlatformKVM,
		TargetAddress:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
		TargetPlatform: models.PlatformKVM,
		Phase:          migratepkg.PhasePlan,
		Plan: &migratepkg.MigrationPlan{
			GeneratedAt:       now.Add(-12 * time.Minute),
			TotalWorkloads:    1,
			ApprovalSatisfied: true,
			WaveStrategy:      migratepkg.WaveStrategy{Size: 1, DependencyAware: true},
			Waves: []migratepkg.MigrationWave{
				{
					Index:      1,
					Reason:     "fixture",
					Dependency: true,
					Workloads: []migratepkg.PlannedWorkload{
						{
							VMID:          discoveryResult.VMs[0].ID,
							Name:          discoveryResult.VMs[0].Name,
							TargetHost:    "lab-host-01",
							TargetStorage: "default",
						},
					},
				},
			},
		},
		Workloads: []migratepkg.WorkloadMigration{
			{VM: discoveryResult.VMs[0], Phase: migratepkg.PhasePlan},
		},
		StartedAt: now.Add(-12 * time.Minute),
		UpdatedAt: now.Add(-11 * time.Minute),
	}
	plannedPayload, err := json.Marshal(plannedState)
	if err != nil {
		return fmt.Errorf("marshal planned migration: %w", err)
	}
	if err := stateStore.SaveMigration(ctx, tenantID, store.MigrationRecord{
		ID:        e2eSavedMigrationID,
		TenantID:  tenantID,
		SpecName:  plannedSpecName,
		Phase:     string(migratepkg.PhasePlan),
		StartedAt: plannedState.StartedAt,
		UpdatedAt: plannedState.UpdatedAt,
		RawJSON:   plannedPayload,
	}); err != nil {
		return fmt.Errorf("seed planned migration: %w", err)
	}

	completedSpecName := "e2e-kvm-complete"
	completedAt := now.Add(-90 * time.Minute)
	completedState := migratepkg.MigrationState{
		ID:             e2eCompletedMigrationID,
		SpecName:       completedSpecName,
		SourceAddress:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
		SourcePlatform: models.PlatformKVM,
		TargetAddress:  filepath.ToSlash(filepath.Join("examples", "lab", "kvm")),
		TargetPlatform: models.PlatformKVM,
		Phase:          migratepkg.PhaseComplete,
		Workloads: []migratepkg.WorkloadMigration{
			{VM: discoveryResult.VMs[1], Phase: migratepkg.PhaseComplete},
		},
		StartedAt:   now.Add(-2 * time.Hour),
		UpdatedAt:   completedAt,
		CompletedAt: completedAt,
	}
	completedPayload, err := json.Marshal(completedState)
	if err != nil {
		return fmt.Errorf("marshal completed migration: %w", err)
	}
	if err := stateStore.SaveMigration(ctx, tenantID, store.MigrationRecord{
		ID:          e2eCompletedMigrationID,
		TenantID:    tenantID,
		SpecName:    completedSpecName,
		Phase:       string(migratepkg.PhaseComplete),
		StartedAt:   completedState.StartedAt,
		UpdatedAt:   completedState.UpdatedAt,
		CompletedAt: completedState.CompletedAt,
		RawJSON:     completedPayload,
	}); err != nil {
		return fmt.Errorf("seed completed migration: %w", err)
	}

	return nil
}

func selectedWorkloadIDs(vms []models.VirtualMachine) []string {
	ids := make([]string, 0, len(vms))
	for _, vm := range vms {
		primary := vm.SourceRef
		if primary == "" {
			primary = vm.ID
		}
		if primary == "" {
			primary = vm.Name
		}
		ids = append(ids, fmt.Sprintf("%s:%s", vm.Platform, primary))
	}
	return ids
}

func cloneDiscoveryResult(result *models.DiscoveryResult) *models.DiscoveryResult {
	if result == nil {
		return &models.DiscoveryResult{}
	}

	cloned := *result
	cloned.VMs = append([]models.VirtualMachine(nil), result.VMs...)
	cloned.Networks = append([]models.NetworkInfo(nil), result.Networks...)
	cloned.Datastores = append([]models.DatastoreInfo(nil), result.Datastores...)
	cloned.Hosts = append([]models.HostInfo(nil), result.Hosts...)
	cloned.Clusters = append([]models.ClusterInfo(nil), result.Clusters...)
	cloned.ResourcePools = append([]models.ResourcePoolInfo(nil), result.ResourcePools...)
	cloned.Errors = append([]string(nil), result.Errors...)
	return &cloned
}

func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
		return filepath.Join(repoRoot, path)
	}
	return path
}
