package plugin

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

type testPluginServer struct {
	status      string
	platform    string
	connectOK   bool
	closeOK     bool
	emptyResult bool
	mu          sync.Mutex
	lastConfig  connectors.Config
}

func (s *testPluginServer) Connect(ctx context.Context, request *ConnectRequest) (*ConnectResponse, error) {
	if request != nil {
		s.mu.Lock()
		s.lastConfig = request.Config
		s.mu.Unlock()
	}
	return &ConnectResponse{OK: s.connectOK}, nil
}

func (s *testPluginServer) Discover(ctx context.Context, request *DiscoverRequest) (*DiscoverResponse, error) {
	if s.emptyResult {
		return &DiscoverResponse{}, nil
	}
	return &DiscoverResponse{Result: &models.DiscoveryResult{
		Source:   "plugin",
		Platform: models.Platform(s.platform),
		VMs: []models.VirtualMachine{
			{ID: "vm-1", Name: "plugin-vm", Platform: models.Platform(s.platform)},
		},
	}}, nil
}

func (s *testPluginServer) Platform(ctx context.Context, request *PlatformRequest) (*PlatformResponse, error) {
	return &PlatformResponse{Platform: s.platform}, nil
}

func (s *testPluginServer) Close(ctx context.Context, request *CloseRequest) (*CloseResponse, error) {
	return &CloseResponse{OK: s.closeOK}, nil
}

func (s *testPluginServer) Health(ctx context.Context, request *HealthRequest) (*HealthResponse, error) {
	return &HealthResponse{Status: s.status}, nil
}

func TestPluginHost_LoadPluginAndDiscover_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "ok",
		platform:  "example",
		connectOK: true,
		closeOK:   true,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()

	if err := host.LoadPlugin("grpc://" + address); err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	connector, err := host.GetConnector("example")
	if err != nil {
		t.Fatalf("GetConnector() error = %v", err)
	}
	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	result, err := connector.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.VMs) != 1 || result.VMs[0].Name != "plugin-vm" {
		t.Fatalf("unexpected discovery result: %#v", result)
	}
}

func TestPluginHost_LoadPlugin_HealthCheckRejectsUnhealthy(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "degraded",
		platform:  "example",
		connectOK: true,
		closeOK:   true,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()

	if err := host.LoadPlugin("grpc://" + address); err == nil {
		t.Fatal("LoadPlugin() error = nil, want health check failure")
	}
}

func TestPluginHost_LoadPlugin_EmptyPlatformRejected_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "ok",
		platform:  "",
		connectOK: true,
		closeOK:   true,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()

	if err := host.LoadPlugin("grpc://" + address); err == nil {
		t.Fatal("LoadPlugin() error = nil, want empty platform rejection")
	}
}

func TestPluginConnector_Connect_UnsuccessfulResponseRejected_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "ok",
		platform:  "example",
		connectOK: false,
		closeOK:   true,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()
	if err := host.LoadPlugin("grpc://" + address); err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	connector, err := host.GetConnector("example")
	if err != nil {
		t.Fatalf("GetConnector() error = %v", err)
	}
	if err := connector.Connect(context.Background()); err == nil {
		t.Fatal("Connect() error = nil, want unsuccessful initialization failure")
	}
}

func TestPluginConnector_Discover_EmptyResultRejected_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:      "ok",
		platform:    "example",
		connectOK:   true,
		closeOK:     true,
		emptyResult: true,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()
	if err := host.LoadPlugin("grpc://" + address); err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	connector, err := host.GetConnector("example")
	if err != nil {
		t.Fatalf("GetConnector() error = %v", err)
	}
	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if _, err := connector.Discover(context.Background()); err == nil {
		t.Fatal("Discover() error = nil, want empty result failure")
	}
}

func TestPluginConnector_Close_UnsuccessfulResponseRejected_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "ok",
		platform:  "example",
		connectOK: true,
		closeOK:   false,
	})
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()
	if err := host.LoadPlugin("grpc://" + address); err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	connector, err := host.GetConnector("example")
	if err != nil {
		t.Fatalf("GetConnector() error = %v", err)
	}
	if err := connector.Close(); err == nil {
		t.Fatal("Close() error = nil, want unsuccessful shutdown failure")
	}
}

func TestPluginHost_Factory_ForwardsConnectorConfig_Expected(t *testing.T) {
	t.Parallel()

	serverState := &testPluginServer{
		status:    "ok",
		platform:  "example",
		connectOK: true,
		closeOK:   true,
	}
	address, shutdown := startTestPluginServer(t, serverState)
	defer shutdown()

	host := NewPluginHost()
	defer host.ShutdownAll()
	if err := host.LoadPlugin("grpc://" + address); err != nil {
		t.Fatalf("LoadPlugin() error = %v", err)
	}

	factory, err := host.Factory("example")
	if err != nil {
		t.Fatalf("Factory() error = %v", err)
	}

	connector := factory(connectors.Config{
		Address:  "https://plugin.example.local",
		Username: "operator",
		Password: "secret",
		Insecure: true,
		Port:     9440,
	})
	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	serverState.mu.Lock()
	defer serverState.mu.Unlock()
	if serverState.lastConfig.Address != "https://plugin.example.local" {
		t.Fatalf("forwarded address = %q, want https://plugin.example.local", serverState.lastConfig.Address)
	}
	if serverState.lastConfig.Username != "operator" || serverState.lastConfig.Password != "secret" {
		t.Fatalf("forwarded credentials = %#v, want operator/secret", serverState.lastConfig)
	}
	if !serverState.lastConfig.Insecure || serverState.lastConfig.Port != 9440 {
		t.Fatalf("forwarded transport settings = %#v, want insecure=true and port=9440", serverState.lastConfig)
	}
}

func TestPluginHost_ConnectPlugin_IncompatibleHostVersionRejected_Expected(t *testing.T) {
	t.Parallel()

	address, shutdown := startTestPluginServer(t, &testPluginServer{
		status:    "ok",
		platform:  "example",
		connectOK: true,
		closeOK:   true,
	})
	defer shutdown()

	host := NewPluginHostWithVersion("v1.1.0")
	defer host.ShutdownAll()

	_, err := host.connectPlugin(nil, address, "grpc://"+address, &Manifest{
		Name:                  "Example Plugin",
		Platform:              "example",
		Version:               "1.0.0",
		ProtocolVersion:       "v1",
		MinimumViaductVersion: "v1.2.0",
	})
	if err == nil {
		t.Fatal("connectPlugin() error = nil, want incompatible host-version rejection")
	}
}

func startTestPluginServer(t *testing.T, plugin *testPluginServer) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	server := NewGRPCServer()
	RegisterConnectorPluginServer(server, plugin)
	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}
