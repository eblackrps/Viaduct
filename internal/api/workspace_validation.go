package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

type requestValidationError struct {
	message     string
	fieldErrors []apiFieldError
}

func (e requestValidationError) Error() string {
	return e.message
}

func validateWorkspace(workspace models.PilotWorkspace) error {
	fieldErrors := make([]apiFieldError, 0)
	if strings.TrimSpace(workspace.Name) == "" {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "name", Message: "workspace name is required"})
	}

	sourceIDs := make(map[string]struct{}, len(workspace.SourceConnections))
	for index, source := range workspace.SourceConnections {
		pathPrefix := fmt.Sprintf("source_connections[%d]", index)
		if strings.TrimSpace(source.Name) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".name", Message: "source connection name is required"})
		}
		if !source.Platform.Valid() {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".platform", Message: "source connection platform must be one of vmware, proxmox, hyperv, kvm, or nutanix"})
		}
		if strings.TrimSpace(source.Address) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".address", Message: "source connection address is required"})
		}
		if sourceID := strings.TrimSpace(source.ID); sourceID != "" {
			if _, exists := sourceIDs[sourceID]; exists {
				fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".id", Message: "source connection IDs must be unique within a workspace"})
			} else {
				sourceIDs[sourceID] = struct{}{}
			}
		}
	}

	if workspace.TargetAssumptions.Platform != "" && !workspace.TargetAssumptions.Platform.Valid() {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "target_assumptions.platform", Message: "target platform must be one of vmware, proxmox, hyperv, kvm, or nutanix"})
	}
	if workspace.PlanSettings.Parallel < 0 {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "plan_settings.parallel", Message: "plan parallelism cannot be negative"})
	}
	if workspace.PlanSettings.WaveSize < 0 {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "plan_settings.wave_size", Message: "plan wave size cannot be negative"})
	}
	if !workspace.PlanSettings.WindowStart.IsZero() && !workspace.PlanSettings.WindowEnd.IsZero() && workspace.PlanSettings.WindowEnd.Before(workspace.PlanSettings.WindowStart) {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "plan_settings.window_end", Message: "plan window end must be after the window start"})
	}

	selectedWorkloads := make(map[string]struct{}, len(workspace.SelectedWorkloadIDs))
	for index, workloadID := range workspace.SelectedWorkloadIDs {
		path := fmt.Sprintf("selected_workload_ids[%d]", index)
		trimmed := strings.TrimSpace(workloadID)
		if trimmed == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "selected workload identifiers cannot be blank"})
			continue
		}
		if _, exists := selectedWorkloads[trimmed]; exists {
			fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "selected workload identifiers must be unique"})
			continue
		}
		selectedWorkloads[trimmed] = struct{}{}
	}

	for index, approval := range workspace.Approvals {
		pathPrefix := fmt.Sprintf("approvals[%d]", index)
		if strings.TrimSpace(approval.Stage) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".stage", Message: "approval stage is required"})
		}
		switch approval.Status {
		case "", models.WorkspaceApprovalStatusPending, models.WorkspaceApprovalStatusApproved, models.WorkspaceApprovalStatusRejected:
		default:
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".status", Message: "approval status must be pending, approved, or rejected"})
		}
	}

	for index, note := range workspace.Notes {
		pathPrefix := fmt.Sprintf("notes[%d]", index)
		switch note.Kind {
		case "", models.WorkspaceNoteKindOperator, models.WorkspaceNoteKindSystem:
		default:
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".kind", Message: "note kind must be operator or system"})
		}
		if strings.TrimSpace(note.Author) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".author", Message: "note author is required"})
		}
		if strings.TrimSpace(note.Body) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: pathPrefix + ".body", Message: "note body is required"})
		}
	}

	if len(fieldErrors) > 0 {
		return requestValidationError{
			message:     "workspace request contains invalid fields",
			fieldErrors: fieldErrors,
		}
	}
	return nil
}

func validateWorkspaceJobRequest(workspace models.PilotWorkspace, request workspaceJobCreateRequest) error {
	fieldErrors := make([]apiFieldError, 0)
	switch request.Type {
	case models.WorkspaceJobTypeDiscovery, models.WorkspaceJobTypeGraph, models.WorkspaceJobTypeSimulation, models.WorkspaceJobTypePlan:
	case "":
		fieldErrors = append(fieldErrors, apiFieldError{Path: "type", Message: "workspace job type is required"})
	default:
		fieldErrors = append(fieldErrors, apiFieldError{Path: "type", Message: "workspace job type must be discovery, graph, simulation, or plan"})
	}

	if len(request.SourceConnectionIDs) > 0 {
		knownSourceConnections := make(map[string]struct{}, len(workspace.SourceConnections))
		for _, source := range workspace.SourceConnections {
			if sourceID := strings.TrimSpace(source.ID); sourceID != "" {
				knownSourceConnections[sourceID] = struct{}{}
			}
		}
		for index, sourceID := range request.SourceConnectionIDs {
			path := fmt.Sprintf("source_connection_ids[%d]", index)
			trimmed := strings.TrimSpace(sourceID)
			if trimmed == "" {
				fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "source connection identifiers cannot be blank"})
				continue
			}
			if _, exists := knownSourceConnections[trimmed]; !exists {
				fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "source connection must belong to the workspace"})
			}
		}
	}

	selectedIDs := make(map[string]struct{}, len(request.SelectedWorkloadIDs))
	for index, workloadID := range request.SelectedWorkloadIDs {
		path := fmt.Sprintf("selected_workload_ids[%d]", index)
		trimmed := strings.TrimSpace(workloadID)
		if trimmed == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "selected workload identifiers cannot be blank"})
			continue
		}
		if _, exists := selectedIDs[trimmed]; exists {
			fieldErrors = append(fieldErrors, apiFieldError{Path: path, Message: "selected workload identifiers must be unique"})
			continue
		}
		selectedIDs[trimmed] = struct{}{}
	}

	if request.Simulation != nil && request.Simulation.TargetPlatform != "" && !request.Simulation.TargetPlatform.Valid() {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "simulation.target_platform", Message: "simulation target platform must be one of vmware, proxmox, hyperv, kvm, or nutanix"})
	}

	switch request.Type {
	case models.WorkspaceJobTypeDiscovery:
		if len(workspace.SourceConnections) == 0 {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "source_connection_ids", Message: "workspace discovery requires at least one source connection"})
		}
	case models.WorkspaceJobTypeGraph:
		if len(workspace.Snapshots) == 0 {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "type", Message: "graph generation requires at least one saved discovery snapshot"})
		}
	case models.WorkspaceJobTypeSimulation:
		if len(workspace.Snapshots) == 0 {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "type", Message: "simulation requires at least one saved discovery snapshot"})
		}
		if workspace.TargetAssumptions.Platform == "" && (request.Simulation == nil || request.Simulation.TargetPlatform == "") {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "simulation.target_platform", Message: "simulation requires a target platform in the workspace or job request"})
		}
	case models.WorkspaceJobTypePlan:
		if len(workspace.Snapshots) == 0 {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "type", Message: "plan generation requires at least one saved discovery snapshot"})
		}
		if !workspace.TargetAssumptions.Platform.Valid() {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "target_assumptions.platform", Message: "plan generation requires a valid target platform on the workspace"})
		}
		if strings.TrimSpace(workspace.TargetAssumptions.Address) == "" {
			fieldErrors = append(fieldErrors, apiFieldError{Path: "target_assumptions.address", Message: "plan generation requires a target address on the workspace"})
		}
	}

	if len(fieldErrors) > 0 {
		return requestValidationError{
			message:     "workspace job request contains invalid fields",
			fieldErrors: fieldErrors,
		}
	}
	return nil
}

func validateWorkspaceReportExportRequest(request workspaceReportExportRequest) error {
	fieldErrors := make([]apiFieldError, 0)
	if request.Name != "" && strings.TrimSpace(request.Name) == "" {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "name", Message: "report name cannot be blank"})
	}

	format := strings.ToLower(strings.TrimSpace(request.Format))
	if format != "" && format != "markdown" && format != "json" {
		fieldErrors = append(fieldErrors, apiFieldError{Path: "format", Message: "supported report formats are markdown and json"})
	}

	if len(fieldErrors) > 0 {
		return requestValidationError{
			message:     "workspace report export request contains invalid fields",
			fieldErrors: fieldErrors,
		}
	}
	return nil
}

func writeValidationFailure(w http.ResponseWriter, r *http.Request, err error) bool {
	validationErr, ok := err.(requestValidationError)
	if !ok {
		return false
	}
	writeAPIError(w, r, http.StatusBadRequest, "invalid_request", validationErr.message, apiErrorOptions{
		FieldErrors: validationErr.fieldErrors,
	})
	return true
}
