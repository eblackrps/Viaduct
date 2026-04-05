package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/eblackrps/viaduct/internal/connectors/plugin"
	"github.com/eblackrps/viaduct/internal/models"
)

type examplePluginServer struct{}

func (s *examplePluginServer) Connect(ctx context.Context, request *plugin.ConnectRequest) (*plugin.ConnectResponse, error) {
	return &plugin.ConnectResponse{OK: true}, nil
}

func (s *examplePluginServer) Discover(ctx context.Context, request *plugin.DiscoverRequest) (*plugin.DiscoverResponse, error) {
	return &plugin.DiscoverResponse{Result: &models.DiscoveryResult{
		Source:   "example-plugin",
		Platform: models.Platform("example"),
		VMs: []models.VirtualMachine{
			{ID: "example-1", Name: "example-vm", Platform: models.Platform("example"), PowerState: models.PowerOn},
		},
	}}, nil
}

func (s *examplePluginServer) Platform(ctx context.Context, request *plugin.PlatformRequest) (*plugin.PlatformResponse, error) {
	return &plugin.PlatformResponse{Platform: "example"}, nil
}

func (s *examplePluginServer) Close(ctx context.Context, request *plugin.CloseRequest) (*plugin.CloseResponse, error) {
	return &plugin.CloseResponse{OK: true}, nil
}

func (s *examplePluginServer) Health(ctx context.Context, request *plugin.HealthRequest) (*plugin.HealthResponse, error) {
	return &plugin.HealthResponse{Status: "ok"}, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	address := os.Getenv("VIADUCT_PLUGIN_ADDR")
	if address == "" {
		address = "127.0.0.1:50071"
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen plugin server: %w", err)
	}

	server := plugin.NewGRPCServer()
	plugin.RegisterConnectorPluginServer(server, &examplePluginServer{})
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("serve plugin server: %w", err)
	}
	return nil
}
