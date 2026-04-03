package connectors

import (
	"context"

	"github.com/eblackrps/viaduct/internal/models"
)

// Connector defines the behavior every hypervisor connector must implement.
type Connector interface {
	// Connect establishes a connection to the hypervisor management plane.
	Connect(ctx context.Context) error
	// Discover retrieves the full VM inventory from the connected platform.
	Discover(ctx context.Context) (*models.DiscoveryResult, error)
	// Platform returns the platform identifier for this connector.
	Platform() models.Platform
	// Close cleanly shuts down the connection.
	Close() error
}

// Config holds connection parameters for any connector.
type Config struct {
	// Address is the hostname, IP address, or URL of the source platform.
	Address string `json:"address" yaml:"address"`
	// Username is the username used for authentication.
	Username string `json:"username" yaml:"username"`
	// Password is the secret used for authentication.
	Password string `json:"-" yaml:"password"`
	// Insecure controls whether TLS verification should be skipped.
	Insecure bool `json:"insecure" yaml:"insecure"`
	// Port overrides the default platform API port when needed.
	Port int `json:"port,omitempty" yaml:"port,omitempty"`
}
