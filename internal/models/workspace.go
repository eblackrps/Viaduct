package models

import (
	"encoding/json"
	"time"
)

// PilotWorkspaceStatus identifies the current operator workflow state for a pilot workspace.
type PilotWorkspaceStatus string

// WorkspaceJobType identifies the type of background job persisted for a pilot workspace.
type WorkspaceJobType string

// WorkspaceJobStatus identifies the lifecycle state of a persisted workspace job.
type WorkspaceJobStatus string

// WorkspaceReadinessStatus identifies the operator-facing readiness posture for a workspace.
type WorkspaceReadinessStatus string

// WorkspaceApprovalStatus identifies the approval state recorded against a workspace stage.
type WorkspaceApprovalStatus string

// WorkspaceNoteKind identifies the category of note recorded against a workspace.
type WorkspaceNoteKind string

const (
	// PilotWorkspaceStatusDraft reports that a workspace has been created but not yet discovered.
	PilotWorkspaceStatusDraft PilotWorkspaceStatus = "draft"
	// PilotWorkspaceStatusDiscovered reports that discovery snapshots have been collected.
	PilotWorkspaceStatusDiscovered PilotWorkspaceStatus = "discovered"
	// PilotWorkspaceStatusGraphReady reports that dependency graph output has been generated.
	PilotWorkspaceStatusGraphReady PilotWorkspaceStatus = "graph-ready"
	// PilotWorkspaceStatusSimulated reports that simulation and readiness output has been generated.
	PilotWorkspaceStatusSimulated PilotWorkspaceStatus = "simulated"
	// PilotWorkspaceStatusPlanned reports that a migration plan has been saved for the workspace.
	PilotWorkspaceStatusPlanned PilotWorkspaceStatus = "planned"
	// PilotWorkspaceStatusReported reports that an exportable pilot report has been generated.
	PilotWorkspaceStatusReported PilotWorkspaceStatus = "reported"

	// WorkspaceJobTypeDiscovery reports a background discovery job.
	WorkspaceJobTypeDiscovery WorkspaceJobType = "discovery"
	// WorkspaceJobTypeGraph reports a dependency-graph generation job.
	WorkspaceJobTypeGraph WorkspaceJobType = "graph"
	// WorkspaceJobTypeSimulation reports a simulation and readiness job.
	WorkspaceJobTypeSimulation WorkspaceJobType = "simulation"
	// WorkspaceJobTypePlan reports a migration plan generation job.
	WorkspaceJobTypePlan WorkspaceJobType = "plan"

	// WorkspaceJobStatusQueued reports that a job has been accepted but not yet started.
	WorkspaceJobStatusQueued WorkspaceJobStatus = "queued"
	// WorkspaceJobStatusRunning reports that a job is currently executing.
	WorkspaceJobStatusRunning WorkspaceJobStatus = "running"
	// WorkspaceJobStatusSucceeded reports that a job completed successfully.
	WorkspaceJobStatusSucceeded WorkspaceJobStatus = "succeeded"
	// WorkspaceJobStatusFailed reports that a job completed with an error.
	WorkspaceJobStatusFailed WorkspaceJobStatus = "failed"

	// WorkspaceReadinessStatusReady reports that the current workspace state is ready for pilot handoff.
	WorkspaceReadinessStatusReady WorkspaceReadinessStatus = "ready"
	// WorkspaceReadinessStatusAttention reports that the workspace needs operator review before handoff.
	WorkspaceReadinessStatusAttention WorkspaceReadinessStatus = "attention"
	// WorkspaceReadinessStatusBlocked reports that the workspace has blocking issues.
	WorkspaceReadinessStatusBlocked WorkspaceReadinessStatus = "blocked"

	// WorkspaceApprovalStatusPending reports that approval has not yet been granted.
	WorkspaceApprovalStatusPending WorkspaceApprovalStatus = "pending"
	// WorkspaceApprovalStatusApproved reports that approval has been granted.
	WorkspaceApprovalStatusApproved WorkspaceApprovalStatus = "approved"
	// WorkspaceApprovalStatusRejected reports that approval has been denied.
	WorkspaceApprovalStatusRejected WorkspaceApprovalStatus = "rejected"

	// WorkspaceNoteKindOperator identifies a human-authored operator note.
	WorkspaceNoteKindOperator WorkspaceNoteKind = "operator"
	// WorkspaceNoteKindSystem identifies a system-authored note.
	WorkspaceNoteKindSystem WorkspaceNoteKind = "system"
)

// WorkspaceSourceConnection records a source endpoint and credential reference associated with a workspace.
type WorkspaceSourceConnection struct {
	// ID is the stable source-connection identifier.
	ID string `json:"id" yaml:"id"`
	// Name is the human-readable label shown in the operator UI.
	Name string `json:"name" yaml:"name"`
	// Platform identifies the source platform to discover from.
	Platform Platform `json:"platform" yaml:"platform"`
	// Address is the source API endpoint or fixture path.
	Address string `json:"address" yaml:"address"`
	// CredentialRef is the optional credential reference resolved by runtime configuration.
	CredentialRef string `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`
	// LastSnapshotID is the most recent saved discovery snapshot generated from this source.
	LastSnapshotID string `json:"last_snapshot_id,omitempty" yaml:"last_snapshot_id,omitempty"`
	// LastDiscoveredAt is when the most recent snapshot completed.
	LastDiscoveredAt time.Time `json:"last_discovered_at,omitempty" yaml:"last_discovered_at,omitempty"`
}

// WorkspaceSnapshot records a discovery snapshot associated with a workspace source.
type WorkspaceSnapshot struct {
	// SnapshotID is the persisted discovery snapshot identifier.
	SnapshotID string `json:"snapshot_id" yaml:"snapshot_id"`
	// SourceConnectionID identifies which workspace source produced the snapshot.
	SourceConnectionID string `json:"source_connection_id" yaml:"source_connection_id"`
	// Source is the source system label captured with the snapshot.
	Source string `json:"source" yaml:"source"`
	// Platform is the discovered platform for the snapshot.
	Platform Platform `json:"platform" yaml:"platform"`
	// VMCount is the number of workloads captured in the snapshot.
	VMCount int `json:"vm_count" yaml:"vm_count"`
	// DiscoveredAt is when the snapshot was collected.
	DiscoveredAt time.Time `json:"discovered_at" yaml:"discovered_at"`
}

// WorkspaceTargetAssumptions records the target environment assumptions used for simulation and plan generation.
type WorkspaceTargetAssumptions struct {
	// Platform identifies the planned target platform.
	Platform Platform `json:"platform,omitempty" yaml:"platform,omitempty"`
	// Address is the target endpoint address when one is known.
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
	// CredentialRef is the optional credential reference used for target validation later in the workflow.
	CredentialRef string `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`
	// DefaultHost is the default target host used during plan generation.
	DefaultHost string `json:"default_host,omitempty" yaml:"default_host,omitempty"`
	// DefaultStorage is the default target storage used during plan generation.
	DefaultStorage string `json:"default_storage,omitempty" yaml:"default_storage,omitempty"`
	// DefaultNetwork is the default target network mapping used during plan generation.
	DefaultNetwork string `json:"default_network,omitempty" yaml:"default_network,omitempty"`
	// Notes captures free-form target placement assumptions supplied by the operator.
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// WorkspacePlanSettings records declarative execution controls attached to a workspace plan.
type WorkspacePlanSettings struct {
	// Name is the migration spec name to generate when saving a plan.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Parallel is the requested parallelism for plan generation.
	Parallel int `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	// VerifyBoot reports whether boot verification should be enabled for the generated plan.
	VerifyBoot bool `json:"verify_boot,omitempty" yaml:"verify_boot,omitempty"`
	// ApprovalRequired reports whether approval should be required before execution.
	ApprovalRequired bool `json:"approval_required,omitempty" yaml:"approval_required,omitempty"`
	// ApprovedBy captures the operator who approved the plan when applicable.
	ApprovedBy string `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	// ApprovalTicket captures the change or approval ticket associated with the plan.
	ApprovalTicket string `json:"approval_ticket,omitempty" yaml:"approval_ticket,omitempty"`
	// WindowStart is the optional execution window start timestamp.
	WindowStart time.Time `json:"window_start,omitempty" yaml:"window_start,omitempty"`
	// WindowEnd is the optional execution window end timestamp.
	WindowEnd time.Time `json:"window_end,omitempty" yaml:"window_end,omitempty"`
	// WaveSize is the requested migration wave size.
	WaveSize int `json:"wave_size,omitempty" yaml:"wave_size,omitempty"`
	// DependencyAware reports whether waves should honor dependency ordering.
	DependencyAware bool `json:"dependency_aware,omitempty" yaml:"dependency_aware,omitempty"`
}

// WorkspaceGraphArtifact records persisted dependency graph output for a workspace.
type WorkspaceGraphArtifact struct {
	// GeneratedAt is when the graph was generated.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	// NodeCount is the number of graph nodes generated.
	NodeCount int `json:"node_count" yaml:"node_count"`
	// EdgeCount is the number of graph edges generated.
	EdgeCount int `json:"edge_count" yaml:"edge_count"`
	// RawJSON contains the serialized dependency graph payload.
	RawJSON json.RawMessage `json:"raw_json,omitempty" yaml:"raw_json,omitempty"`
}

// WorkspaceSimulationArtifact records persisted simulation output for a workspace.
type WorkspaceSimulationArtifact struct {
	// GeneratedAt is when the simulation completed.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	// TargetPlatform is the target platform simulated.
	TargetPlatform Platform `json:"target_platform,omitempty" yaml:"target_platform,omitempty"`
	// SelectedWorkloadIDs records which workloads were included in the simulation scope.
	SelectedWorkloadIDs []string `json:"selected_workload_ids,omitempty" yaml:"selected_workload_ids,omitempty"`
	// MovedVMs is the number of workloads simulated for movement.
	MovedVMs int `json:"moved_vms,omitempty" yaml:"moved_vms,omitempty"`
	// RawJSON contains the serialized simulation payload.
	RawJSON json.RawMessage `json:"raw_json,omitempty" yaml:"raw_json,omitempty"`
}

// WorkspaceReadinessResult records the latest operator-facing readiness summary for a workspace.
type WorkspaceReadinessResult struct {
	// GeneratedAt is when readiness was derived.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	// Status is the summarized readiness posture.
	Status WorkspaceReadinessStatus `json:"status" yaml:"status"`
	// SelectedWorkloadCount is the number of workloads currently in scope.
	SelectedWorkloadCount int `json:"selected_workload_count" yaml:"selected_workload_count"`
	// RecommendationCount is the number of active recommendations in scope.
	RecommendationCount int `json:"recommendation_count" yaml:"recommendation_count"`
	// PolicyViolationCount is the number of simulated policy violations in scope.
	PolicyViolationCount int `json:"policy_violation_count" yaml:"policy_violation_count"`
	// BlockingIssues contains operator-facing blocking issues.
	BlockingIssues []string `json:"blocking_issues,omitempty" yaml:"blocking_issues,omitempty"`
	// WarningIssues contains operator-facing warnings that still require review.
	WarningIssues []string `json:"warning_issues,omitempty" yaml:"warning_issues,omitempty"`
}

// WorkspaceSavedPlan records a persisted migration plan and the generated migration state payload.
type WorkspaceSavedPlan struct {
	// MigrationID is the identifier of the saved migration plan record.
	MigrationID string `json:"migration_id" yaml:"migration_id"`
	// GeneratedAt is when the plan was generated.
	GeneratedAt time.Time `json:"generated_at" yaml:"generated_at"`
	// SpecName is the generated migration spec name.
	SpecName string `json:"spec_name" yaml:"spec_name"`
	// SourcePlatform identifies the source platform used for the plan.
	SourcePlatform Platform `json:"source_platform,omitempty" yaml:"source_platform,omitempty"`
	// TargetPlatform identifies the target platform used for the plan.
	TargetPlatform Platform `json:"target_platform,omitempty" yaml:"target_platform,omitempty"`
	// WorkloadCount is the number of workloads included in the plan.
	WorkloadCount int `json:"workload_count" yaml:"workload_count"`
	// SelectedWorkloadIDs records which workloads were included in the saved plan.
	SelectedWorkloadIDs []string `json:"selected_workload_ids,omitempty" yaml:"selected_workload_ids,omitempty"`
	// SpecJSON contains the serialized migration spec used to build the plan.
	SpecJSON json.RawMessage `json:"spec_json,omitempty" yaml:"spec_json,omitempty"`
	// StateJSON contains the serialized migration state generated for the saved plan.
	StateJSON json.RawMessage `json:"state_json,omitempty" yaml:"state_json,omitempty"`
}

// WorkspaceApproval records an explicit approval decision associated with a workspace stage.
type WorkspaceApproval struct {
	// ID is the stable approval identifier.
	ID string `json:"id" yaml:"id"`
	// Stage identifies the stage that required approval.
	Stage string `json:"stage" yaml:"stage"`
	// Status identifies whether the approval is pending, approved, or rejected.
	Status WorkspaceApprovalStatus `json:"status" yaml:"status"`
	// ApprovedBy identifies who recorded the decision.
	ApprovedBy string `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	// Ticket records the change or review ticket tied to the decision.
	Ticket string `json:"ticket,omitempty" yaml:"ticket,omitempty"`
	// Notes captures free-form approval commentary.
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`
	// CreatedAt is when the approval decision was recorded.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// WorkspaceNote records a persisted operator or system note against a workspace.
type WorkspaceNote struct {
	// ID is the stable note identifier.
	ID string `json:"id" yaml:"id"`
	// Kind identifies whether the note was recorded by an operator or the system.
	Kind WorkspaceNoteKind `json:"kind" yaml:"kind"`
	// Author identifies who or what recorded the note.
	Author string `json:"author" yaml:"author"`
	// Body contains the note content.
	Body string `json:"body" yaml:"body"`
	// CreatedAt is when the note was recorded.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// WorkspaceReportArtifact records a persisted report export event for a workspace.
type WorkspaceReportArtifact struct {
	// ID is the stable report export identifier.
	ID string `json:"id" yaml:"id"`
	// Name identifies the exported report template.
	Name string `json:"name" yaml:"name"`
	// Format identifies the export format such as markdown or json.
	Format string `json:"format" yaml:"format"`
	// FileName is the suggested download filename.
	FileName string `json:"file_name" yaml:"file_name"`
	// CorrelationID records the request correlation ID associated with the export.
	CorrelationID string `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`
	// ExportedAt is when the report was generated.
	ExportedAt time.Time `json:"exported_at" yaml:"exported_at"`
}

// PilotWorkspace ties together discovery, analysis, planning, and operator review state for a pilot assessment.
type PilotWorkspace struct {
	// ID is the stable workspace identifier.
	ID string `json:"id" yaml:"id"`
	// TenantID is the tenant that owns the workspace.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// Name is the human-readable workspace name.
	Name string `json:"name" yaml:"name"`
	// Description captures the operator-facing purpose of the workspace.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Status identifies the current workflow state.
	Status PilotWorkspaceStatus `json:"status" yaml:"status"`
	// CreatedAt is when the workspace was created.
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	// UpdatedAt is when the workspace was last updated.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// SourceConnections contains the source endpoints and credential references attached to the workspace.
	SourceConnections []WorkspaceSourceConnection `json:"source_connections,omitempty" yaml:"source_connections,omitempty"`
	// Snapshots contains the discovery snapshots currently attached to the workspace.
	Snapshots []WorkspaceSnapshot `json:"snapshots,omitempty" yaml:"snapshots,omitempty"`
	// SelectedWorkloadIDs records the workloads currently selected for simulation and plan generation.
	SelectedWorkloadIDs []string `json:"selected_workload_ids,omitempty" yaml:"selected_workload_ids,omitempty"`
	// TargetAssumptions captures target-side planning assumptions.
	TargetAssumptions WorkspaceTargetAssumptions `json:"target_assumptions,omitempty" yaml:"target_assumptions,omitempty"`
	// PlanSettings captures declarative execution controls for generated plans.
	PlanSettings WorkspacePlanSettings `json:"plan_settings,omitempty" yaml:"plan_settings,omitempty"`
	// Graph captures the latest dependency graph output when one has been generated.
	Graph *WorkspaceGraphArtifact `json:"graph,omitempty" yaml:"graph,omitempty"`
	// Simulation captures the latest simulation output when one has been generated.
	Simulation *WorkspaceSimulationArtifact `json:"simulation,omitempty" yaml:"simulation,omitempty"`
	// Readiness captures the latest readiness summary when one has been generated.
	Readiness *WorkspaceReadinessResult `json:"readiness,omitempty" yaml:"readiness,omitempty"`
	// SavedPlan captures the latest saved migration plan for the workspace.
	SavedPlan *WorkspaceSavedPlan `json:"saved_plan,omitempty" yaml:"saved_plan,omitempty"`
	// Approvals records operator approval decisions captured against the workspace.
	Approvals []WorkspaceApproval `json:"approvals,omitempty" yaml:"approvals,omitempty"`
	// Notes records free-form operator or system notes captured against the workspace.
	Notes []WorkspaceNote `json:"notes,omitempty" yaml:"notes,omitempty"`
	// Reports records previously generated report exports for the workspace.
	Reports []WorkspaceReportArtifact `json:"reports,omitempty" yaml:"reports,omitempty"`
}

// WorkspaceJob records a persisted background job for discovery, graph generation, simulation, or plan generation.
type WorkspaceJob struct {
	// ID is the stable job identifier.
	ID string `json:"id" yaml:"id"`
	// TenantID is the tenant that owns the job.
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// WorkspaceID is the workspace associated with the job.
	WorkspaceID string `json:"workspace_id" yaml:"workspace_id"`
	// Type identifies the job action.
	Type WorkspaceJobType `json:"type" yaml:"type"`
	// Status identifies the current job lifecycle state.
	Status WorkspaceJobStatus `json:"status" yaml:"status"`
	// RequestedBy identifies who requested the job.
	RequestedBy string `json:"requested_by,omitempty" yaml:"requested_by,omitempty"`
	// RequestedAt is when the job was accepted.
	RequestedAt time.Time `json:"requested_at" yaml:"requested_at"`
	// StartedAt is when job execution began.
	StartedAt time.Time `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	// UpdatedAt is when the job was last updated.
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// CompletedAt is when the job finished.
	CompletedAt time.Time `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	// CorrelationID captures the request correlation ID associated with the job.
	CorrelationID string `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`
	// Message contains operator-facing progress detail for the latest update.
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	// Error contains the terminal error message when the job failed.
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
	// Retryable reports whether the failure is safe to retry.
	Retryable bool `json:"retryable,omitempty" yaml:"retryable,omitempty"`
	// InputJSON contains the serialized job request payload.
	InputJSON json.RawMessage `json:"input_json,omitempty" yaml:"input_json,omitempty"`
	// OutputJSON contains the serialized job result payload.
	OutputJSON json.RawMessage `json:"output_json,omitempty" yaml:"output_json,omitempty"`
}
