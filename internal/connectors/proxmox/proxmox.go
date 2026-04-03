package proxmox

import (
	"context"
	"fmt"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// ProxmoxConnector is the Proxmox implementation of the connector interface.
type ProxmoxConnector struct {
	config    connectors.Config
	connected bool
}

// NewProxmoxConnector creates a Proxmox connector with the provided configuration.
func NewProxmoxConnector(cfg connectors.Config) *ProxmoxConnector {
	return &ProxmoxConnector{config: cfg}
}

// Connect establishes a logical connection to the Proxmox endpoint.
func (c *ProxmoxConnector) Connect(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("proxmox: connect: %w", ctx.Err())
	default:
	}

	c.connected = true
	return nil
}

// Discover retrieves inventory from Proxmox once the connector has been connected.
func (c *ProxmoxConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("proxmox: discover: %w", ctx.Err())
	default:
	}

	if !c.connected {
		return nil, fmt.Errorf("proxmox: not connected, call Connect first")
	}

	return nil, fmt.Errorf("proxmox: discovery not yet implemented")
}

// Platform returns the Proxmox platform identifier.
func (c *ProxmoxConnector) Platform() models.Platform {
	return models.PlatformProxmox
}

// Close resets the connector connection state.
func (c *ProxmoxConnector) Close() error {
	c.connected = false
	return nil
}

var _ connectors.Connector = (*ProxmoxConnector)(nil)
