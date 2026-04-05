package models

import "time"

// AuditOutcome describes the result of an auditable operation.
type AuditOutcome string

const (
	// AuditOutcomeSuccess indicates that the audited operation completed successfully.
	AuditOutcomeSuccess AuditOutcome = "success"
	// AuditOutcomeFailure indicates that the audited operation failed.
	AuditOutcomeFailure AuditOutcome = "failure"
)

// AuditEvent records a tenant-scoped operational event for security and support workflows.
type AuditEvent struct {
	// ID is the unique audit event identifier.
	ID string `json:"id"`
	// TenantID is the tenant that owns the event.
	TenantID string `json:"tenant_id"`
	// Actor identifies who or what initiated the event.
	Actor string `json:"actor"`
	// RequestID correlates the event with an API request when available.
	RequestID string `json:"request_id,omitempty"`
	// Category groups related events such as migration, admin, or backup portability.
	Category string `json:"category"`
	// Action is the specific action that occurred within the category.
	Action string `json:"action"`
	// Resource identifies the affected object or resource.
	Resource string `json:"resource,omitempty"`
	// Outcome records whether the action succeeded or failed.
	Outcome AuditOutcome `json:"outcome"`
	// Message contains a concise human-readable description of the event.
	Message string `json:"message"`
	// Details carries additional structured context for the event.
	Details map[string]string `json:"details,omitempty"`
	// CreatedAt is when the event was recorded.
	CreatedAt time.Time `json:"created_at"`
}
