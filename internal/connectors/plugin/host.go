package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

const connectorPluginServiceName = "viaduct.plugin.v1.ConnectorPlugin"

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

type jsonCodec struct{}

func (jsonCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func (jsonCodec) Name() string {
	return "json"
}

// ConnectRequest requests connector initialization inside a plugin process.
type ConnectRequest struct {
	// Config is the connector configuration the plugin should apply.
	Config connectors.Config `json:"config"`
}

// MarshalJSON preserves redacted connector defaults for general JSON output while still carrying
// the password field across the plugin RPC boundary.
func (r ConnectRequest) MarshalJSON() ([]byte, error) {
	type jsonConfig struct {
		Address   string `json:"address"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Insecure  bool   `json:"insecure"`
		Port      int    `json:"port,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	return json.Marshal(struct {
		Config jsonConfig `json:"config"`
	}{
		Config: jsonConfig{
			Address:   r.Config.Address,
			Username:  r.Config.Username,
			Password:  r.Config.Password,
			Insecure:  r.Config.Insecure,
			Port:      r.Config.Port,
			RequestID: r.Config.RequestID,
		},
	})
}

// UnmarshalJSON decodes a plugin connection request and restores the shared connector config.
func (r *ConnectRequest) UnmarshalJSON(data []byte) error {
	type jsonConfig struct {
		Address   string `json:"address"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Insecure  bool   `json:"insecure"`
		Port      int    `json:"port,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	var payload struct {
		Config jsonConfig `json:"config"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	r.Config = connectors.Config{
		Address:   payload.Config.Address,
		Username:  payload.Config.Username,
		Password:  payload.Config.Password,
		Insecure:  payload.Config.Insecure,
		Port:      payload.Config.Port,
		RequestID: payload.Config.RequestID,
	}
	return nil
}

// ConnectResponse reports whether plugin-side initialization succeeded.
type ConnectResponse struct {
	// OK reports plugin-side success.
	OK bool `json:"ok"`
}

// DiscoverRequest requests discovery from the plugin connector.
type DiscoverRequest struct{}

// DiscoverResponse carries normalized inventory returned by the plugin connector.
type DiscoverResponse struct {
	// Result is the normalized inventory result.
	Result *models.DiscoveryResult `json:"result"`
}

// PlatformRequest requests the plugin platform identifier.
type PlatformRequest struct{}

// PlatformResponse returns the platform served by a plugin.
type PlatformResponse struct {
	// Platform is the plugin platform identifier.
	Platform string `json:"platform"`
}

// CloseRequest requests connector shutdown.
type CloseRequest struct{}

// CloseResponse reports whether shutdown completed successfully.
type CloseResponse struct {
	// OK reports plugin-side success.
	OK bool `json:"ok"`
}

// HealthRequest requests the plugin health state.
type HealthRequest struct{}

// HealthResponse reports plugin liveness and readiness.
type HealthResponse struct {
	// Status is the health status string.
	Status string `json:"status"`
}

// ConnectorPluginServer defines the gRPC service exported by community connector plugins.
type ConnectorPluginServer interface {
	// Connect initializes the plugin connector with the supplied configuration.
	Connect(context.Context, *ConnectRequest) (*ConnectResponse, error)
	// Discover returns normalized inventory from the plugin connector.
	Discover(context.Context, *DiscoverRequest) (*DiscoverResponse, error)
	// Platform returns the platform identifier served by the plugin.
	Platform(context.Context, *PlatformRequest) (*PlatformResponse, error)
	// Close shuts down plugin connector resources.
	Close(context.Context, *CloseRequest) (*CloseResponse, error)
	// Health reports plugin readiness.
	Health(context.Context, *HealthRequest) (*HealthResponse, error)
}

type connectorPluginClient struct {
	connection *grpc.ClientConn
}

func newConnectorPluginClient(connection *grpc.ClientConn) *connectorPluginClient {
	return &connectorPluginClient{connection: connection}
}

func (c *connectorPluginClient) Connect(ctx context.Context, request *ConnectRequest) (*ConnectResponse, error) {
	response := &ConnectResponse{}
	if err := c.connection.Invoke(ctx, "/"+connectorPluginServiceName+"/Connect", request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *connectorPluginClient) Discover(ctx context.Context, request *DiscoverRequest) (*DiscoverResponse, error) {
	response := &DiscoverResponse{}
	if err := c.connection.Invoke(ctx, "/"+connectorPluginServiceName+"/Discover", request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *connectorPluginClient) Platform(ctx context.Context, request *PlatformRequest) (*PlatformResponse, error) {
	response := &PlatformResponse{}
	if err := c.connection.Invoke(ctx, "/"+connectorPluginServiceName+"/Platform", request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *connectorPluginClient) Close(ctx context.Context, request *CloseRequest) (*CloseResponse, error) {
	response := &CloseResponse{}
	if err := c.connection.Invoke(ctx, "/"+connectorPluginServiceName+"/Close", request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *connectorPluginClient) Health(ctx context.Context, request *HealthRequest) (*HealthResponse, error) {
	response := &HealthResponse{}
	if err := c.connection.Invoke(ctx, "/"+connectorPluginServiceName+"/Health", request, response); err != nil {
		return nil, err
	}
	return response, nil
}

// RegisterConnectorPluginServer registers a community connector plugin service with a gRPC server.
func RegisterConnectorPluginServer(server grpc.ServiceRegistrar, implementation ConnectorPluginServer) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: connectorPluginServiceName,
		HandlerType: (*ConnectorPluginServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "Connect", Handler: unaryHandler(func(ctx context.Context, request *ConnectRequest, impl ConnectorPluginServer) (any, error) {
				return impl.Connect(ctx, request)
			})},
			{MethodName: "Discover", Handler: unaryHandler(func(ctx context.Context, request *DiscoverRequest, impl ConnectorPluginServer) (any, error) {
				return impl.Discover(ctx, request)
			})},
			{MethodName: "Platform", Handler: unaryHandler(func(ctx context.Context, request *PlatformRequest, impl ConnectorPluginServer) (any, error) {
				return impl.Platform(ctx, request)
			})},
			{MethodName: "Close", Handler: unaryHandler(func(ctx context.Context, request *CloseRequest, impl ConnectorPluginServer) (any, error) {
				return impl.Close(ctx, request)
			})},
			{MethodName: "Health", Handler: unaryHandler(func(ctx context.Context, request *HealthRequest, impl ConnectorPluginServer) (any, error) {
				return impl.Health(ctx, request)
			})},
		},
	}, implementation)
}

type unaryCall[T any] func(context.Context, *T, ConnectorPluginServer) (any, error)

func unaryHandler[T any](call unaryCall[T]) grpc.MethodHandler {
	return func(service any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		request := new(T)
		if err := decode(request); err != nil {
			return nil, err
		}

		implementation := service.(ConnectorPluginServer)
		if interceptor == nil {
			return call(ctx, request, implementation)
		}

		info := &grpc.UnaryServerInfo{
			Server:     service,
			FullMethod: "/" + connectorPluginServiceName,
		}
		return interceptor(ctx, request, info, func(currentCtx context.Context, currentRequest any) (any, error) {
			return call(currentCtx, currentRequest.(*T), implementation)
		})
	}
}

// NewGRPCServer creates a JSON-coded gRPC server suitable for connector plugins.
func NewGRPCServer() *grpc.Server {
	return grpc.NewServer(grpc.ForceServerCodec(jsonCodec{}))
}

// PluginProcess represents a running community connector plugin.
type PluginProcess struct {
	path     string
	address  string
	platform string
	manifest *Manifest
	command  *exec.Cmd
	conn     *grpc.ClientConn
	client   *connectorPluginClient
}

// PluginHost manages the lifecycle of community connector plugins.
type PluginHost struct {
	mu          sync.RWMutex
	hostVersion string
	plugins     map[string]*PluginProcess
}

// NewPluginHost creates an empty plugin host.
func NewPluginHost() *PluginHost {
	return NewPluginHostWithVersion(inferHostVersion())
}

// NewPluginHostWithVersion creates an empty plugin host with an explicit host-version marker.
func NewPluginHostWithVersion(version string) *PluginHost {
	return &PluginHost{
		hostVersion: strings.TrimSpace(version),
		plugins:     make(map[string]*PluginProcess),
	}
}

// LoadPlugin starts or connects to a plugin and registers it by platform.
func (h *PluginHost) LoadPlugin(path string) error {
	if h == nil {
		return fmt.Errorf("load plugin: host is nil")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("load plugin: path is required")
	}

	process, err := h.startPlugin(path)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	platformKey := strings.ToLower(strings.TrimSpace(process.platform))
	if platformKey == "" {
		// The plugin cannot be used without a platform; cleanup is best effort because we are already failing load.
		_ = closePluginProcess(process)
		return fmt.Errorf("load plugin: plugin platform is empty")
	}
	if existing, ok := h.plugins[platformKey]; ok {
		if err := closePluginProcess(existing); err != nil {
			log.Printf("component=plugin platform=%s message=%q", platformKey, fmt.Sprintf("failed to close replaced plugin: %v", err))
		}
	}
	h.plugins[platformKey] = process
	return nil
}

// GetConnector returns a connector backed by a loaded plugin.
func (h *PluginHost) GetConnector(platform string) (connectors.Connector, error) {
	factory, err := h.Factory(platform)
	if err != nil {
		return nil, err
	}
	return factory(connectors.Config{}), nil
}

// Factory returns a connector factory backed by a loaded plugin.
func (h *PluginHost) Factory(platform string) (connectors.Factory, error) {
	if h == nil {
		return nil, fmt.Errorf("get connector: host is nil")
	}

	h.mu.RLock()
	process, ok := h.plugins[strings.ToLower(platform)]
	h.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("get connector: plugin for platform %q not found", platform)
	}

	return func(cfg connectors.Config) connectors.Connector {
		return &PluginConnector{
			process:  process,
			platform: models.Platform(process.platform),
			config:   cfg,
		}
	}, nil
}

// ShutdownAll closes all loaded plugins and stops spawned processes.
func (h *PluginHost) ShutdownAll() {
	if h == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for key, process := range h.plugins {
		if process == nil {
			continue
		}
		if err := closePluginProcess(process); err != nil {
			log.Printf("component=plugin platform=%s message=%q", key, fmt.Sprintf("failed to shut down plugin: %v", err))
		}
		delete(h.plugins, key)
	}
}

// PluginConnector adapts a plugin process to the core Connector interface.
type PluginConnector struct {
	process  *PluginProcess
	platform models.Platform
	config   connectors.Config
}

// Connect initializes the plugin connector.
func (c *PluginConnector) Connect(ctx context.Context) error {
	if c == nil || c.process == nil || c.process.client == nil {
		return fmt.Errorf("plugin connector: process is unavailable")
	}
	response, err := c.process.client.Connect(ctx, &ConnectRequest{Config: c.config})
	if err != nil {
		return fmt.Errorf("plugin connector: connect: %w", err)
	}
	if response == nil || !response.OK {
		return fmt.Errorf("plugin connector: connect: plugin reported unsuccessful initialization")
	}
	return nil
}

// Discover retrieves normalized inventory from the plugin connector.
func (c *PluginConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if c == nil || c.process == nil || c.process.client == nil {
		return nil, fmt.Errorf("plugin connector: process is unavailable")
	}
	response, err := c.process.client.Discover(ctx, &DiscoverRequest{})
	if err != nil {
		return nil, fmt.Errorf("plugin connector: discover: %w", err)
	}
	if response == nil || response.Result == nil {
		return nil, fmt.Errorf("plugin connector: discover: plugin returned an empty discovery result")
	}
	return response.Result, nil
}

// Platform returns the plugin platform identifier.
func (c *PluginConnector) Platform() models.Platform {
	return c.platform
}

// Close closes plugin-side connector state.
func (c *PluginConnector) Close() error {
	if c == nil || c.process == nil || c.process.client == nil {
		return nil
	}
	response, err := c.process.client.Close(context.Background(), &CloseRequest{})
	if err != nil {
		return err
	}
	if response == nil || !response.OK {
		return fmt.Errorf("plugin connector: close: plugin reported unsuccessful shutdown")
	}
	return nil
}

func (h *PluginHost) startPlugin(path string) (*PluginProcess, error) {
	if strings.HasPrefix(path, "grpc://") {
		return h.connectPlugin(nil, strings.TrimPrefix(path, "grpc://"), path, nil)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}

	pluginAddress, err := reserveLoopbackAddress()
	if err != nil {
		return nil, fmt.Errorf("load plugin: reserve address: %w", err)
	}

	command := exec.Command(path)
	command.Dir = filepath.Dir(path)
	command.Env = append(os.Environ(), "VIADUCT_PLUGIN_ADDR="+pluginAddress)
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("load plugin: start %s: %w", path, err)
	}

	process, err := h.connectPlugin(command, pluginAddress, path, manifest)
	if err != nil {
		// The child process may already be exiting; terminate and reap it on a best-effort basis before returning.
		_ = command.Process.Kill()
		// Wait is best effort here because the process may already have been reaped while we unwind the failed launch.
		_, _ = command.Process.Wait()
		return nil, err
	}
	return process, nil
}

func (h *PluginHost) connectPlugin(command *exec.Cmd, address, path string, manifest *Manifest) (*PluginProcess, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connection, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		return nil, fmt.Errorf("load plugin: dial %s: %w", address, err)
	}
	connection.Connect()

	client := newConnectorPluginClient(connection)
	health, err := client.Health(ctx, &HealthRequest{})
	if err != nil {
		// Connection teardown is best effort during plugin handshake failures.
		_ = connection.Close()
		return nil, fmt.Errorf("load plugin: health check: %w", err)
	}
	if !strings.EqualFold(health.Status, "ok") && !strings.EqualFold(health.Status, "healthy") {
		// Connection teardown is best effort during plugin handshake failures.
		_ = connection.Close()
		return nil, fmt.Errorf("load plugin: unhealthy plugin status %q", health.Status)
	}

	platformResponse, err := client.Platform(ctx, &PlatformRequest{})
	if err != nil {
		// Connection teardown is best effort during plugin handshake failures.
		_ = connection.Close()
		return nil, fmt.Errorf("load plugin: platform lookup: %w", err)
	}
	if manifest != nil && !strings.EqualFold(strings.TrimSpace(manifest.Platform), strings.TrimSpace(platformResponse.Platform)) {
		// Connection teardown is best effort during plugin handshake failures.
		_ = connection.Close()
		return nil, fmt.Errorf("load plugin: manifest platform %q does not match plugin platform %q", manifest.Platform, platformResponse.Platform)
	}
	if err := manifestSupportsHost(manifest, h.hostVersion); err != nil {
		// Connection teardown is best effort during plugin handshake failures.
		_ = connection.Close()
		return nil, fmt.Errorf("load plugin: %w", err)
	}

	return &PluginProcess{
		path:     path,
		address:  address,
		platform: platformResponse.Platform,
		manifest: manifest,
		command:  command,
		conn:     connection,
		client:   client,
	}, nil
}

func reserveLoopbackAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", err
	}
	return address, nil
}

func closePluginProcess(process *PluginProcess) error {
	if process == nil {
		return nil
	}

	var closeErr error
	if process.client != nil {
		if _, err := process.client.Close(context.Background(), &CloseRequest{}); err != nil && !isIgnorablePluginCloseError(err) {
			closeErr = err
		}
	}
	if process.conn != nil {
		if err := process.conn.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if process.command != nil && process.command.Process != nil {
		if err := process.command.Process.Kill(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "process has already exited") && closeErr == nil {
			closeErr = err
		}
		if _, err := process.command.Process.Wait(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func isIgnorablePluginCloseError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "code = unavailable") ||
		strings.Contains(message, "actively refused") ||
		strings.Contains(message, "connection error")
}

var _ connectors.Connector = (*PluginConnector)(nil)

func inferHostVersion() string {
	if version := strings.TrimSpace(os.Getenv("VIADUCT_VERSION")); version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info != nil && strings.TrimSpace(info.Main.Version) != "" {
		return info.Main.Version
	}
	return "dev"
}
