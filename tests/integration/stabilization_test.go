package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apipkg "github.com/eblackrps/viaduct/internal/api"
	"github.com/eblackrps/viaduct/internal/connectors/plugin"
	"github.com/eblackrps/viaduct/internal/connectors/veeam"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestWarmMigration_InterruptedResume_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	sourcePayload := append(bytes.Repeat([]byte("A"), 1024*1024), bytes.Repeat([]byte("B"), 1024*1024)...)
	if err := os.WriteFile(sourcePath, sourcePayload, 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	replicator := migratepkg.NewReplicator(migratepkg.ReplicationConfig{
		SourceDisk:         sourcePath,
		TargetDisk:         targetPath,
		BlockSizeKB:        1024,
		BandwidthLimitMBps: 1,
	}, stateStore)
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	if err := overwriteBytesAt(sourcePath, 0, bytes.Repeat([]byte{0x44}, 1024*1024)); err != nil {
		t.Fatalf("overwriteBytesAt(first block) error = %v", err)
	}
	if err := overwriteBytesAt(sourcePath, 1024*1024, bytes.Repeat([]byte{0x55}, 1024*1024)); err != nil {
		t.Fatalf("overwriteBytesAt(second block) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	if _, err := replicator.RunIncrementalSync(ctx); err == nil {
		t.Fatal("RunIncrementalSync() error = nil, want interruption")
	} else if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunIncrementalSync() error = %v, want context cancellation", err)
	}

	records, err := stateStore.ListMigrations(context.Background(), store.DefaultTenantID, 10)
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}
	if len(records) == 0 {
		t.Fatal("ListMigrations() returned no persisted replication state")
	}

	resumed := migratepkg.NewReplicator(migratepkg.ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: 1024,
	}, stateStore)
	if err := resumed.LoadState(context.Background(), records[0].ID); err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if _, err := resumed.RunIncrementalSync(context.Background()); err != nil {
		t.Fatalf("resumed RunIncrementalSync() error = %v", err)
	}

	targetPayload, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(target) error = %v", err)
	}
	currentSourcePayload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(source) error = %v", err)
	}
	if !bytes.Equal(currentSourcePayload, targetPayload) {
		t.Fatal("target payload does not match source after resumed integration sync")
	}
}

func TestWarmMigration_BootFailureRollback_Expected(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.qcow2")
	targetPath := filepath.Join(tempDir, "target.qcow2")
	if err := os.WriteFile(sourcePath, bytes.Repeat([]byte("C"), 8192), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	stateStore := store.NewMemoryStore()
	replicator := migratepkg.NewReplicator(migratepkg.ReplicationConfig{
		SourceDisk:  sourcePath,
		TargetDisk:  targetPath,
		BlockSizeKB: 4,
	}, stateStore)
	if err := replicator.StartInitialCopy(context.Background()); err != nil {
		t.Fatalf("StartInitialCopy() error = %v", err)
	}

	sourceVM := loadInventoryFixture(t, "mock_source_inventory.json").VMs[0]
	source := &mockIntegrationConnector{platform: models.PlatformVMware}
	target := &mockIntegrationConnector{
		platform:  models.PlatformProxmox,
		verifyErr: context.DeadlineExceeded,
	}
	rollback := migratepkg.NewRollbackManager(stateStore, source, target)
	coordinator := migratepkg.NewCutoverCoordinator(source, target, replicator, rollback, nil)

	report, err := coordinator.ExecuteCutover(context.Background(), &migratepkg.CutoverPlan{
		MigrationID:           "warm-cutover-failure",
		SourceVM:              sourceVM,
		TargetPlatform:        models.PlatformProxmox,
		BootTimeout:           25 * time.Millisecond,
		AutoRollbackOnFailure: true,
	})
	if err == nil {
		t.Fatal("ExecuteCutover() error = nil, want boot verification failure")
	}
	if report == nil || !report.RolledBack {
		t.Fatalf("unexpected cutover report: %#v", report)
	}
	if len(source.restored) != 1 {
		t.Fatalf("restored source VMs = %d, want 1", len(source.restored))
	}
}

func TestBackupPortability_VerificationAndRollback_Expected(t *testing.T) {
	t.Parallel()

	var created, deleted atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs":
			created.Add(1)
			writeLifecycleJSON(t, w, map[string]string{"id": "job-created-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/job-created-1/start":
			writeLifecycleJSON(t, w, map[string]string{"status": "started"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/jobs/job-created-1":
			deleted.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := veeam.NewVeeamClient(server.URL, true)
	manager := veeam.NewPortabilityManager(client)

	result, err := manager.ExecuteJobMigration(context.Background(), &veeam.JobMigrationPlan{
		Jobs: []veeam.BackupJobTemplate{
			{Name: "Daily web-01-target", TargetRepo: "target-repo", ProtectedVMs: []string{"web-01-target"}, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteJobMigration() error = %v", err)
	}
	if err := manager.RollbackJobMigration(context.Background(), result); err != nil {
		t.Fatalf("RollbackJobMigration() error = %v", err)
	}
	if created.Load() != 1 || deleted.Load() != 1 {
		t.Fatalf("created=%d deleted=%d, want 1/1", created.Load(), deleted.Load())
	}
}

func TestTenantIsolation_ConcurrentAccess_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	for _, tenant := range []models.Tenant{
		{ID: "tenant-a", Name: "Tenant A", APIKey: "tenant-a-key", Active: true},
		{ID: "tenant-b", Name: "Tenant B", APIKey: "tenant-b-key", Active: true},
	} {
		if err := stateStore.CreateTenant(context.Background(), tenant); err != nil {
			t.Fatalf("CreateTenant(%s) error = %v", tenant.ID, err)
		}
	}
	if _, err := stateStore.SaveDiscovery(context.Background(), "tenant-a", &models.DiscoveryResult{
		Source:       "tenant-a-source",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveDiscovery(tenant-a) error = %v", err)
	}
	if _, err := stateStore.SaveDiscovery(context.Background(), "tenant-b", &models.DiscoveryResult{
		Source:       "tenant-b-source",
		Platform:     models.PlatformProxmox,
		DiscoveredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveDiscovery(tenant-b) error = %v", err)
	}

	handler := apipkg.TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := stateStore.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(items); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
	}))

	var wg sync.WaitGroup
	errCh := make(chan error, 40)
	for _, tenant := range []struct {
		apiKey string
		source string
	}{
		{apiKey: "tenant-a-key", source: "tenant-a-source"},
		{apiKey: "tenant-b-key", source: "tenant-b-source"},
	} {
		tenant := tenant
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
				req.Header.Set("X-API-Key", tenant.apiKey)
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, req)
				if recorder.Code != http.StatusOK {
					errCh <- errors.New(recorder.Body.String())
					return
				}

				var items []store.SnapshotMeta
				if err := json.Unmarshal(recorder.Body.Bytes(), &items); err != nil {
					errCh <- err
					return
				}
				if len(items) != 1 || items[0].Source != tenant.source {
					errCh <- errors.New("tenant request observed cross-tenant snapshot data")
				}
			}()
		}
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent tenant isolation error: %v", err)
		}
	}
}

func TestPluginCrash_AfterLoad_DiscoverFails_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startIntegrationPluginServer(t, "ok")
	host := plugin.NewPluginHost()
	defer host.ShutdownAll()

	if err := host.LoadPlugin("grpc://" + address); err != nil {
		shutdown()
		t.Fatalf("LoadPlugin() error = %v", err)
	}
	connector, err := host.GetConnector("example")
	if err != nil {
		shutdown()
		t.Fatalf("GetConnector() error = %v", err)
	}

	shutdown()
	if _, err := connector.Discover(context.Background()); err == nil {
		t.Fatal("Discover() error = nil, want plugin crash failure")
	}
}

func TestTenantCompatibility_DefaultTenantLegacyDataRequiresExplicitCredentials_Expected(t *testing.T) {
	t.Parallel()

	stateStore := store.NewMemoryStore()
	if _, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, &models.DiscoveryResult{
		Source:       "legacy-default",
		Platform:     models.PlatformVMware,
		DiscoveredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SaveDiscovery(default) error = %v", err)
	}
	if err := stateStore.CreateTenant(context.Background(), models.Tenant{
		ID:     "tenant-disabled",
		Name:   "Disabled",
		APIKey: "disabled-key",
		Active: false,
	}); err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}
	defaultTenant, err := stateStore.GetTenant(context.Background(), store.DefaultTenantID)
	if err != nil {
		t.Fatalf("GetTenant(default) error = %v", err)
	}
	defaultTenant.APIKey = "default-key"
	if err := stateStore.UpdateTenant(context.Background(), *defaultTenant); err != nil {
		t.Fatalf("UpdateTenant(default) error = %v", err)
	}

	handler := apipkg.TenantAuthMiddleware(stateStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items, err := stateStore.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(items) != 1 || items[0].Source != "legacy-default" {
			http.Error(w, "default tenant compatibility failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots", nil)
	req.Header.Set("X-API-Key", "default-key")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

type integrationPluginServer struct {
	status string
}

func (s *integrationPluginServer) Connect(ctx context.Context, request *plugin.ConnectRequest) (*plugin.ConnectResponse, error) {
	return &plugin.ConnectResponse{OK: true}, nil
}

func (s *integrationPluginServer) Discover(ctx context.Context, request *plugin.DiscoverRequest) (*plugin.DiscoverResponse, error) {
	return &plugin.DiscoverResponse{Result: &models.DiscoveryResult{
		Source:   "integration-plugin",
		Platform: models.Platform("example"),
		VMs: []models.VirtualMachine{
			{ID: "plugin-vm", Name: "plugin-vm", Platform: models.Platform("example")},
		},
	}}, nil
}

func (s *integrationPluginServer) Platform(ctx context.Context, request *plugin.PlatformRequest) (*plugin.PlatformResponse, error) {
	return &plugin.PlatformResponse{Platform: "example"}, nil
}

func (s *integrationPluginServer) Close(ctx context.Context, request *plugin.CloseRequest) (*plugin.CloseResponse, error) {
	return &plugin.CloseResponse{OK: true}, nil
}

func (s *integrationPluginServer) Health(ctx context.Context, request *plugin.HealthRequest) (*plugin.HealthResponse, error) {
	return &plugin.HealthResponse{Status: s.status}, nil
}

func startIntegrationPluginServer(t *testing.T, status string) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	server := plugin.NewGRPCServer()
	plugin.RegisterConnectorPluginServer(server, &integrationPluginServer{status: status})
	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}

func overwriteBytesAt(path string, offset int64, payload []byte) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteAt(payload, offset)
	return err
}
