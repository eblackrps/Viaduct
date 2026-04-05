package models

import "time"

// Tenant represents an isolated customer or workspace scope in the Viaduct API.
type Tenant struct {
	// ID is the stable tenant identifier used for routing and storage isolation.
	ID string `json:"id" yaml:"id"`
	// Name is the human-readable tenant name.
	Name string `json:"name" yaml:"name"`
	// APIKey is the tenant API credential used by middleware authentication.
	APIKey string `json:"api_key" yaml:"api_key"`
	// CreatedAt is when the tenant was created.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// Active reports whether the tenant may access the API.
	Active bool `json:"active" yaml:"active"`
	// Settings stores optional tenant-specific configuration values.
	Settings map[string]string `json:"settings,omitempty" yaml:"settings,omitempty"`
}
