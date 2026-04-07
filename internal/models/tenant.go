package models

import "time"

// TenantRole identifies the authorization level granted within a tenant boundary.
type TenantRole string

const (
	// TenantRoleViewer grants read-only access to tenant-scoped APIs.
	TenantRoleViewer TenantRole = "viewer"
	// TenantRoleOperator grants read/write access to operational migration workflows.
	TenantRoleOperator TenantRole = "operator"
	// TenantRoleAdmin grants full tenant-scoped administrative access.
	TenantRoleAdmin TenantRole = "admin"
)

// ServiceAccount represents a scoped non-human credential within a tenant.
type ServiceAccount struct {
	// ID is the stable service-account identifier.
	ID string `json:"id" yaml:"id"`
	// Name is the human-readable service-account name.
	Name string `json:"name" yaml:"name"`
	// Description explains the intended purpose of the credential.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// APIKey is the bearer credential used for API authentication.
	APIKey string `json:"api_key" yaml:"api_key"`
	// Role is the tenant-scoped authorization level granted to the service account.
	Role TenantRole `json:"role" yaml:"role"`
	// Active reports whether the service account may authenticate.
	Active bool `json:"active" yaml:"active"`
	// CreatedAt is when the service account was created.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// LastRotatedAt is when the service-account key was last rotated.
	LastRotatedAt time.Time `json:"last_rotated_at,omitempty" yaml:"last_rotated_at,omitempty"`
	// ExpiresAt is when the service account should stop being accepted.
	ExpiresAt time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	// Metadata stores optional operator-supplied labels and ownership details.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// TenantQuota defines soft isolation and fairness limits for a tenant.
type TenantQuota struct {
	// RequestsPerMinute caps tenant-scoped API throughput when greater than zero.
	RequestsPerMinute int `json:"requests_per_minute,omitempty" yaml:"requests_per_minute,omitempty"`
	// MaxSnapshots caps the number of persisted discovery snapshots when greater than zero.
	MaxSnapshots int `json:"max_snapshots,omitempty" yaml:"max_snapshots,omitempty"`
	// MaxMigrations caps the number of persisted migrations when greater than zero.
	MaxMigrations int `json:"max_migrations,omitempty" yaml:"max_migrations,omitempty"`
}

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
	// Quotas stores tenant-scoped capacity and rate limits.
	Quotas TenantQuota `json:"quotas,omitempty" yaml:"quotas,omitempty"`
	// ServiceAccounts stores scoped machine credentials for the tenant.
	ServiceAccounts []ServiceAccount `json:"service_accounts,omitempty" yaml:"service_accounts,omitempty"`
}

// Allows reports whether the current role satisfies the required role.
func (r TenantRole) Allows(required TenantRole) bool {
	return tenantRoleRank(r) >= tenantRoleRank(required)
}

// EffectiveRequestsPerMinute returns the configured request budget or the supplied default.
func (q TenantQuota) EffectiveRequestsPerMinute(defaultLimit int) int {
	if q.RequestsPerMinute > 0 {
		return q.RequestsPerMinute
	}
	return defaultLimit
}

func tenantRoleRank(role TenantRole) int {
	switch role {
	case TenantRoleAdmin:
		return 3
	case TenantRoleOperator:
		return 2
	case TenantRoleViewer:
		return 1
	default:
		return 0
	}
}
