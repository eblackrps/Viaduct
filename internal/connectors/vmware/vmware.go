package vmware

import (
	"context"
	"fmt"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// VMwareConnector is the VMware implementation of the connector interface.
type VMwareConnector struct {
	config    connectors.Config
	connected bool
}

// NewVMwareConnector creates a VMware connector with the provided configuration.
func NewVMwareConnector(cfg connectors.Config) *VMwareConnector {
	return &VMwareConnector{config: cfg}
}

// Connect establishes a logical connection to the VMware endpoint.
func (c *VMwareConnector) Connect(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("vmware: connect: %w", ctx.Err())
	default:
	}

	c.connected = true
	return nil
}

// Discover retrieves inventory from VMware once the connector has been connected.
func (c *VMwareConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("vmware: discover: %w", ctx.Err())
	default:
	}

	if !c.connected {
		return nil, fmt.Errorf("vmware: not connected, call Connect first")
	}

	return nil, fmt.Errorf("vmware: discovery not yet implemented")
}

// Platform returns the VMware platform identifier.
func (c *VMwareConnector) Platform() models.Platform {
	return models.PlatformVMware
}

// Close resets the connector connection state.
func (c *VMwareConnector) Close() error {
	c.connected = false
	return nil
}

var _ connectors.Connector = (*VMwareConnector)(nil)
