package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectorcatalog"
	"github.com/eblackrps/viaduct/internal/deps"
	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/lifecycle"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/google/uuid"
)

type workspaceCreateRequest struct {
	ID                  string                             `json:"id,omitempty"`
	Name                string                             `json:"name"`
	Description         string                             `json:"description,omitempty"`
	SourceConnections   []models.WorkspaceSourceConnection `json:"source_connections,omitempty"`
	SelectedWorkloadIDs []string                           `json:"selected_workload_ids,omitempty"`
	TargetAssumptions   models.WorkspaceTargetAssumptions  `json:"target_assumptions,omitempty"`
	PlanSettings        models.WorkspacePlanSettings       `json:"plan_settings,omitempty"`
	Approvals           []models.WorkspaceApproval         `json:"approvals,omitempty"`
	Notes               []models.WorkspaceNote             `json:"notes,omitempty"`
}

type workspaceUpdateRequest struct {
	Name                *string                             `json:"name,omitempty"`
	Description         *string                             `json:"description,omitempty"`
	SourceConnections   *[]models.WorkspaceSourceConnection `json:"source_connections,omitempty"`
	SelectedWorkloadIDs *[]string                           `json:"selected_workload_ids,omitempty"`
	TargetAssumptions   *models.WorkspaceTargetAssumptions  `json:"target_assumptions,omitempty"`
	PlanSettings        *models.WorkspacePlanSettings       `json:"plan_settings,omitempty"`
	Approvals           *[]models.WorkspaceApproval         `json:"approvals,omitempty"`
	Notes               *[]models.WorkspaceNote             `json:"notes,omitempty"`
}

type workspaceJobCreateRequest struct {
	Type                models.WorkspaceJobType      `json:"type"`
	RequestedBy         string                       `json:"requested_by,omitempty"`
	SourceConnectionIDs []string                     `json:"source_connection_ids,omitempty"`
	SelectedWorkloadIDs []string                     `json:"selected_workload_ids,omitempty"`
	Simulation          *lifecycle.SimulationRequest `json:"simulation,omitempty"`
}

type workspaceReportExportRequest struct {
	Name   string `json:"name,omitempty"`
	Format string `json:"format,omitempty"`
}

type workspaceReportDocument struct {
	Workspace  models.PilotWorkspace `json:"workspace"`
	Jobs       []models.WorkspaceJob `json:"jobs"`
	ExportedAt time.Time             `json:"exported_at"`
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		limit := parseLimitQuery(r, 100)
		items, err := s.store.ListWorkspaces(r.Context(), tenantID, limit)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		defer r.Body.Close()

		var request workspaceCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode workspace request: %w", err).Error(), apiErrorOptions{})
			return
		}
		workspace := models.PilotWorkspace{
			ID:                  strings.TrimSpace(request.ID),
			TenantID:            tenantID,
			Name:                strings.TrimSpace(request.Name),
			Description:         strings.TrimSpace(request.Description),
			Status:              models.PilotWorkspaceStatusDraft,
			SourceConnections:   append([]models.WorkspaceSourceConnection(nil), request.SourceConnections...),
			SelectedWorkloadIDs: append([]string(nil), request.SelectedWorkloadIDs...),
			TargetAssumptions:   request.TargetAssumptions,
			PlanSettings:        request.PlanSettings,
			Approvals:           append([]models.WorkspaceApproval(nil), request.Approvals...),
			Notes:               append([]models.WorkspaceNote(nil), request.Notes...),
		}
		if workspace.ID == "" {
			workspace.ID = uuid.NewString()
		}
		if err := validateWorkspace(workspace); err != nil {
			if writeValidationFailure(w, r, err) {
				return
			}
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}
		if err := s.store.CreateWorkspace(r.Context(), tenantID, workspace); err != nil {
			status := http.StatusBadRequest
			code := "invalid_request"
			if strings.Contains(strings.ToLower(err.Error()), "already exists") {
				status = http.StatusConflict
				code = "conflict"
			}
			writeAPIError(w, r, status, code, err.Error(), apiErrorOptions{})
			return
		}

		created, err := s.store.GetWorkspace(r.Context(), tenantID, workspace.ID)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "workspace",
			Action:   "create",
			Resource: created.ID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "pilot workspace created",
		})
		writeJSON(w, http.StatusCreated, created)
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleWorkspaceByID(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "workspace ID is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "workspace_id", Message: "workspace ID is required"}},
		})
		return
	}

	workspaceID := parts[0]
	if len(parts) == 1 {
		s.handleWorkspaceDocument(w, r, tenantID, workspaceID)
		return
	}

	switch parts[1] {
	case "jobs":
		if len(parts) == 2 {
			s.handleWorkspaceJobs(w, r, tenantID, workspaceID)
			return
		}
		if len(parts) == 3 {
			s.handleWorkspaceJobByID(w, r, tenantID, workspaceID, parts[2])
			return
		}
	case "reports":
		if len(parts) == 3 && parts[2] == "export" {
			s.handleWorkspaceReportExport(w, r, tenantID, workspaceID)
			return
		}
	}

	writeAPIError(w, r, http.StatusNotFound, "invalid_request", "not found", apiErrorOptions{})
}

func (s *Server) handleWorkspaceDocument(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	switch r.Method {
	case http.MethodGet:
		workspace, err := s.store.GetWorkspace(r.Context(), tenantID, workspaceID)
		if err != nil {
			writeAPIError(w, r, http.StatusNotFound, "workspace_not_found", err.Error(), apiErrorOptions{
				Details: map[string]any{"workspace_id": workspaceID},
			})
			return
		}
		writeJSON(w, http.StatusOK, workspace)
	case http.MethodPatch:
		workspace, err := s.store.GetWorkspace(r.Context(), tenantID, workspaceID)
		if err != nil {
			writeAPIError(w, r, http.StatusNotFound, "workspace_not_found", err.Error(), apiErrorOptions{
				Details: map[string]any{"workspace_id": workspaceID},
			})
			return
		}

		defer r.Body.Close()
		var request workspaceUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode workspace update: %w", err).Error(), apiErrorOptions{})
			return
		}

		if request.Name != nil {
			workspace.Name = strings.TrimSpace(*request.Name)
		}
		if request.Description != nil {
			workspace.Description = strings.TrimSpace(*request.Description)
		}
		if request.SourceConnections != nil {
			workspace.SourceConnections = append([]models.WorkspaceSourceConnection(nil), (*request.SourceConnections)...)
		}
		if request.SelectedWorkloadIDs != nil {
			workspace.SelectedWorkloadIDs = append([]string(nil), (*request.SelectedWorkloadIDs)...)
		}
		if request.TargetAssumptions != nil {
			workspace.TargetAssumptions = *request.TargetAssumptions
		}
		if request.PlanSettings != nil {
			workspace.PlanSettings = *request.PlanSettings
		}
		if request.Approvals != nil {
			workspace.Approvals = append([]models.WorkspaceApproval(nil), (*request.Approvals)...)
		}
		if request.Notes != nil {
			workspace.Notes = append([]models.WorkspaceNote(nil), (*request.Notes)...)
		}
		if err := validateWorkspace(*workspace); err != nil {
			if writeValidationFailure(w, r, err) {
				return
			}
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		if err := s.store.UpdateWorkspace(r.Context(), tenantID, *workspace); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		updated, err := s.store.GetWorkspace(r.Context(), tenantID, workspaceID)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "workspace",
			Action:   "update",
			Resource: workspaceID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "pilot workspace updated",
		})
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.store.DeleteWorkspace(r.Context(), tenantID, workspaceID); err != nil {
			writeAPIError(w, r, http.StatusNotFound, "workspace_not_found", err.Error(), apiErrorOptions{
				Details: map[string]any{"workspace_id": workspaceID},
			})
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "workspace",
			Action:   "delete",
			Resource: workspaceID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "pilot workspace deleted",
		})
		w.WriteHeader(http.StatusNoContent)
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleWorkspaceJobs(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	switch r.Method {
	case http.MethodGet:
		limit := parseLimitQuery(r, 25)
		items, err := s.store.ListWorkspaceJobs(r.Context(), tenantID, workspaceID, limit)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		workspace, err := s.store.GetWorkspace(r.Context(), tenantID, workspaceID)
		if err != nil {
			writeAPIError(w, r, http.StatusNotFound, "workspace_not_found", err.Error(), apiErrorOptions{
				Details: map[string]any{"workspace_id": workspaceID},
			})
			return
		}

		defer r.Body.Close()
		var request workspaceJobCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode workspace job request: %w", err).Error(), apiErrorOptions{})
			return
		}
		if err := validateWorkspaceJobRequest(*workspace, request); err != nil {
			if writeValidationFailure(w, r, err) {
				return
			}
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		payload, err := json.Marshal(request)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		now := time.Now().UTC()
		job := models.WorkspaceJob{
			ID:            uuid.NewString(),
			TenantID:      tenantID,
			WorkspaceID:   workspaceID,
			Type:          request.Type,
			Status:        models.WorkspaceJobStatusQueued,
			RequestedBy:   firstNonEmpty(strings.TrimSpace(request.RequestedBy), actorFromContext(r.Context())),
			RequestedAt:   now,
			UpdatedAt:     now,
			CorrelationID: responseRequestID(nil, r),
			Message:       workspaceJobQueuedMessage(request.Type),
			InputJSON:     payload,
		}
		if err := s.store.SaveWorkspaceJob(r.Context(), tenantID, job); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		s.recordAuditEvent(r, models.AuditEvent{
			Category: "workspace",
			Action:   "enqueue-job",
			Resource: job.ID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  fmt.Sprintf("workspace %s job queued", request.Type),
			Details: map[string]string{
				"workspace_id": workspace.ID,
				"job_type":     string(request.Type),
			},
		})

		go s.runWorkspaceJob(tenantID, workspace.ID, job.ID)
		writeJSON(w, http.StatusAccepted, job)
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleWorkspaceJobByID(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, jobID string) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	job, err := s.store.GetWorkspaceJob(r.Context(), tenantID, workspaceID, jobID)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, "workspace_job_not_found", err.Error(), apiErrorOptions{
			Details: map[string]any{
				"workspace_id": workspaceID,
				"job_id":       jobID,
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleWorkspaceReportExport(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	if r.Method != http.MethodPost {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	workspace, err := s.store.GetWorkspace(r.Context(), tenantID, workspaceID)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, "workspace_not_found", err.Error(), apiErrorOptions{
			Details: map[string]any{"workspace_id": workspaceID},
		})
		return
	}

	defer r.Body.Close()
	var request workspaceReportExportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode workspace report export request: %w", err).Error(), apiErrorOptions{})
		return
	}
	if err := validateWorkspaceReportExportRequest(request); err != nil {
		if writeValidationFailure(w, r, err) {
			return
		}
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
		return
	}

	format := strings.ToLower(strings.TrimSpace(request.Format))
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "json" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unsupported report format %q", format), apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "format", Message: "supported formats are markdown and json"}},
		})
		return
	}

	jobs, err := s.store.ListWorkspaceJobs(r.Context(), tenantID, workspaceID, 50)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	document := workspaceReportDocument{
		Workspace:  *workspace,
		Jobs:       jobs,
		ExportedAt: time.Now().UTC(),
	}
	reportName := firstNonEmpty(strings.TrimSpace(request.Name), "pilot-workspace-report")
	reportID := uuid.NewString()
	fileName := fmt.Sprintf("%s-%s.%s", workspaceReportSlug(*workspace), reportID[:8], workspaceReportExtension(format))

	workspace.Reports = append(workspace.Reports, models.WorkspaceReportArtifact{
		ID:            reportID,
		Name:          reportName,
		Format:        format,
		FileName:      fileName,
		CorrelationID: responseRequestID(nil, r),
		ExportedAt:    document.ExportedAt,
	})
	workspace.Status = advanceWorkspaceStatus(workspace.Status, models.PilotWorkspaceStatusReported)
	if err := s.store.UpdateWorkspace(r.Context(), tenantID, *workspace); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	s.recordAuditEvent(r, models.AuditEvent{
		Category: "workspace",
		Action:   "export-report",
		Resource: workspaceID,
		Outcome:  models.AuditOutcomeSuccess,
		Message:  "pilot workspace report exported",
		Details: map[string]string{
			"format": format,
			"name":   reportName,
		},
	})

	if format == "json" {
		payload, err := json.MarshalIndent(document, "", "  ")
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderWorkspaceReportMarkdown(document)))
}

func (s *Server) runWorkspaceJob(tenantID, workspaceID, jobID string) {
	storeCtx := store.ContextWithTenantID(context.Background(), tenantID)

	job, err := s.store.GetWorkspaceJob(storeCtx, tenantID, workspaceID, jobID)
	if err != nil {
		return
	}
	workspace, err := s.store.GetWorkspace(storeCtx, tenantID, workspaceID)
	if err != nil {
		_ = s.failWorkspaceJob(storeCtx, tenantID, *job, false, err)
		return
	}

	job.Status = models.WorkspaceJobStatusRunning
	job.StartedAt = time.Now().UTC()
	job.UpdatedAt = job.StartedAt
	job.Message = workspaceJobRunningMessage(job.Type)
	if err := s.store.SaveWorkspaceJob(storeCtx, tenantID, *job); err != nil {
		return
	}

	request, err := decodeWorkspaceJobRequest(job.InputJSON)
	if err != nil {
		_ = s.failWorkspaceJob(storeCtx, tenantID, *job, false, err)
		return
	}

	var (
		output  json.RawMessage
		message string
	)
	execCtx := storeCtx
	cancel := func() {}
	if s != nil && s.workspaceJobTimeout > 0 {
		execCtx, cancel = context.WithTimeout(storeCtx, s.workspaceJobTimeout)
	}
	execCtx = contextWithConnectorRequestID(execCtx, job.CorrelationID)
	defer cancel()
	switch job.Type {
	case models.WorkspaceJobTypeDiscovery:
		output, message, err = s.executeWorkspaceDiscoveryJob(execCtx, tenantID, workspace, request)
	case models.WorkspaceJobTypeGraph:
		output, message, err = s.executeWorkspaceGraphJob(execCtx, tenantID, workspace)
	case models.WorkspaceJobTypeSimulation:
		output, message, err = s.executeWorkspaceSimulationJob(execCtx, tenantID, workspace, request)
	case models.WorkspaceJobTypePlan:
		output, message, err = s.executeWorkspacePlanJob(execCtx, tenantID, workspace, request)
	default:
		err = fmt.Errorf("unsupported workspace job type %q", job.Type)
	}

	if err != nil {
		_ = s.failWorkspaceJob(storeCtx, tenantID, *job, isRetryableWorkspaceJobError(err), err)
		return
	}

	job.Status = models.WorkspaceJobStatusSucceeded
	job.Message = message
	job.OutputJSON = output
	job.UpdatedAt = time.Now().UTC()
	job.CompletedAt = job.UpdatedAt
	_ = s.store.SaveWorkspaceJob(storeCtx, tenantID, *job)
}

func (s *Server) executeWorkspaceDiscoveryJob(ctx context.Context, tenantID string, workspace *models.PilotWorkspace, request workspaceJobCreateRequest) (json.RawMessage, string, error) {
	sourceConnections := workspaceSourceConnections(*workspace, request.SourceConnectionIDs)
	if len(sourceConnections) == 0 {
		return nil, "", fmt.Errorf("workspace discovery: at least one source connection is required")
	}

	catalog, err := s.workspaceConnectorCatalog()
	if err != nil {
		return nil, "", fmt.Errorf("workspace discovery: %w", err)
	}

	engine := discovery.NewEngine()
	addressToConnectionID := make(map[string]string, len(sourceConnections))
	for _, source := range sourceConnections {
		connector, err := s.buildConnector(ctx, catalog, source.Platform, source.Address, source.CredentialRef)
		if err != nil {
			return nil, "", fmt.Errorf("workspace discovery: build %s connector: %w", source.Platform, err)
		}
		engine.AddSource(source.ID, connector)
		addressToConnectionID[strings.ToLower(strings.TrimSpace(source.Address))] = source.ID
	}

	merged, runErr := engine.RunAll(ctx)
	if merged == nil {
		return nil, "", fmt.Errorf("workspace discovery: no discovery results returned")
	}
	snapshots := make([]models.WorkspaceSnapshot, 0, len(merged.Sources))
	snapshotIDs := make([]string, 0, len(merged.Sources))
	for _, result := range merged.Sources {
		resultCopy := result
		snapshotID, err := s.store.SaveDiscovery(ctx, tenantID, &resultCopy)
		if err != nil {
			return nil, "", fmt.Errorf("workspace discovery: save discovery snapshot: %w", err)
		}
		sourceConnectionID := addressToConnectionID[strings.ToLower(strings.TrimSpace(result.Source))]
		snapshotIDs = append(snapshotIDs, snapshotID)
		snapshots = append(snapshots, models.WorkspaceSnapshot{
			SnapshotID:         snapshotID,
			SourceConnectionID: sourceConnectionID,
			Source:             result.Source,
			Platform:           result.Platform,
			VMCount:            len(result.VMs),
			DiscoveredAt:       result.DiscoveredAt,
		})
	}

	for _, snapshot := range snapshots {
		upsertWorkspaceSnapshot(workspace, snapshot)
		updateWorkspaceSourceConnectionDiscovery(workspace, snapshot)
	}
	if len(snapshots) > 0 {
		workspace.Status = advanceWorkspaceStatus(workspace.Status, models.PilotWorkspaceStatusDiscovered)
	}
	if err := s.store.UpdateWorkspace(ctx, tenantID, *workspace); err != nil {
		return nil, "", fmt.Errorf("workspace discovery: update workspace: %w", err)
	}

	output, err := json.Marshal(map[string]any{
		"snapshot_ids": snapshotIDs,
		"sources":      snapshots,
		"errors":       merged.Errors,
	})
	if err != nil {
		return nil, "", fmt.Errorf("workspace discovery: marshal output: %w", err)
	}
	if runErr != nil {
		return output, fmt.Sprintf("Discovery saved %d snapshot(s) with warnings.", len(snapshotIDs)), fmt.Errorf("workspace discovery: %w", runErr)
	}
	return output, fmt.Sprintf("Discovery saved %d snapshot(s).", len(snapshotIDs)), nil
}

func (s *Server) executeWorkspaceGraphJob(ctx context.Context, tenantID string, workspace *models.PilotWorkspace) (json.RawMessage, string, error) {
	inventory, err := s.workspaceInventory(ctx, tenantID, *workspace)
	if err != nil {
		return nil, "", fmt.Errorf("workspace graph: %w", err)
	}

	graph := deps.BuildGraph(inventory, s.backups)
	output, err := json.Marshal(graph)
	if err != nil {
		return nil, "", fmt.Errorf("workspace graph: %w", err)
	}

	workspace.Graph = &models.WorkspaceGraphArtifact{
		GeneratedAt: time.Now().UTC(),
		NodeCount:   len(graph.Nodes),
		EdgeCount:   len(graph.Edges),
		RawJSON:     output,
	}
	workspace.Status = advanceWorkspaceStatus(workspace.Status, models.PilotWorkspaceStatusGraphReady)
	if err := s.store.UpdateWorkspace(ctx, tenantID, *workspace); err != nil {
		return nil, "", fmt.Errorf("workspace graph: update workspace: %w", err)
	}
	return output, fmt.Sprintf("Dependency graph generated with %d nodes and %d edges.", len(graph.Nodes), len(graph.Edges)), nil
}

func (s *Server) executeWorkspaceSimulationJob(ctx context.Context, tenantID string, workspace *models.PilotWorkspace, request workspaceJobCreateRequest) (json.RawMessage, string, error) {
	inventory, err := s.workspaceInventory(ctx, tenantID, *workspace)
	if err != nil {
		return nil, "", fmt.Errorf("workspace simulation: %w", err)
	}

	selectedIDs := workspaceSelectedWorkloadIDs(*workspace, request.SelectedWorkloadIDs)
	simulationRequest := lifecycle.SimulationRequest{
		TargetPlatform: workspace.TargetAssumptions.Platform,
		VMIDs:          append([]string(nil), selectedIDs...),
		IncludeAll:     len(selectedIDs) == 0,
	}
	if request.Simulation != nil {
		simulationRequest = *request.Simulation
		if len(request.SelectedWorkloadIDs) > 0 {
			simulationRequest.VMIDs = append([]string(nil), request.SelectedWorkloadIDs...)
			simulationRequest.IncludeAll = false
		}
	}
	if simulationRequest.TargetPlatform == "" {
		return nil, "", fmt.Errorf("workspace simulation: target platform is required")
	}

	result, err := s.recommendationEngine.Simulate(inventory, simulationRequest)
	if err != nil {
		return nil, "", fmt.Errorf("workspace simulation: %w", err)
	}
	output, err := json.Marshal(result)
	if err != nil {
		return nil, "", fmt.Errorf("workspace simulation: %w", err)
	}

	if len(request.SelectedWorkloadIDs) > 0 {
		workspace.SelectedWorkloadIDs = append([]string(nil), request.SelectedWorkloadIDs...)
	} else if len(workspace.SelectedWorkloadIDs) == 0 && !simulationRequest.IncludeAll {
		workspace.SelectedWorkloadIDs = append([]string(nil), simulationRequest.VMIDs...)
	}
	workspace.Simulation = &models.WorkspaceSimulationArtifact{
		GeneratedAt:         time.Now().UTC(),
		TargetPlatform:      result.TargetPlatform,
		SelectedWorkloadIDs: append([]string(nil), simulationRequest.VMIDs...),
		MovedVMs:            result.MovedVMs,
		RawJSON:             output,
	}
	workspace.Readiness = deriveWorkspaceReadiness(result, len(simulationRequest.VMIDs), simulationRequest.IncludeAll, len(inventory.VMs))
	workspace.Status = advanceWorkspaceStatus(workspace.Status, models.PilotWorkspaceStatusSimulated)
	if err := s.store.UpdateWorkspace(ctx, tenantID, *workspace); err != nil {
		return nil, "", fmt.Errorf("workspace simulation: update workspace: %w", err)
	}

	return output, fmt.Sprintf("Simulation completed for %d workload(s).", result.MovedVMs), nil
}

func (s *Server) executeWorkspacePlanJob(ctx context.Context, tenantID string, workspace *models.PilotWorkspace, request workspaceJobCreateRequest) (json.RawMessage, string, error) {
	inventory, err := s.workspaceInventory(ctx, tenantID, *workspace)
	if err != nil {
		return nil, "", fmt.Errorf("workspace plan: %w", err)
	}

	selectedIDs := workspaceSelectedWorkloadIDs(*workspace, request.SelectedWorkloadIDs)
	spec, err := buildWorkspaceMigrationSpec(*workspace, inventory, selectedIDs)
	if err != nil {
		return nil, "", fmt.Errorf("workspace plan: %w", err)
	}

	migrationID := uuid.NewString()
	generatedAt := time.Now().UTC()
	state, err := migratepkg.BuildStateFromInventory(spec, inventory, migrationID, generatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("workspace plan: %w", err)
	}

	statePayload, err := json.Marshal(state)
	if err != nil {
		return nil, "", fmt.Errorf("workspace plan: marshal migration state: %w", err)
	}
	if err := s.store.SaveMigration(ctx, tenantID, store.MigrationRecord{
		ID:        migrationID,
		TenantID:  tenantID,
		SpecName:  state.SpecName,
		Phase:     string(state.Phase),
		StartedAt: state.StartedAt,
		UpdatedAt: state.UpdatedAt,
		RawJSON:   statePayload,
	}); err != nil {
		return nil, "", fmt.Errorf("workspace plan: save migration record: %w", err)
	}

	specPayload, err := json.Marshal(spec)
	if err != nil {
		return nil, "", fmt.Errorf("workspace plan: marshal migration spec: %w", err)
	}

	if len(selectedIDs) > 0 {
		workspace.SelectedWorkloadIDs = append([]string(nil), selectedIDs...)
	}
	workspace.SavedPlan = &models.WorkspaceSavedPlan{
		MigrationID:         migrationID,
		GeneratedAt:         generatedAt,
		SpecName:            spec.Name,
		SourcePlatform:      spec.Source.Platform,
		TargetPlatform:      spec.Target.Platform,
		WorkloadCount:       len(state.Workloads),
		SelectedWorkloadIDs: append([]string(nil), selectedIDs...),
		SpecJSON:            specPayload,
		StateJSON:           statePayload,
	}
	syncWorkspacePlanApproval(workspace, generatedAt)
	workspace.Status = advanceWorkspaceStatus(workspace.Status, models.PilotWorkspaceStatusPlanned)
	if err := s.store.UpdateWorkspace(ctx, tenantID, *workspace); err != nil {
		return nil, "", fmt.Errorf("workspace plan: update workspace: %w", err)
	}

	return statePayload, fmt.Sprintf("Saved migration plan %s with %d workload(s).", migrationID, len(state.Workloads)), nil
}

func (s *Server) failWorkspaceJob(ctx context.Context, tenantID string, job models.WorkspaceJob, retryable bool, failure error) error {
	job.Status = models.WorkspaceJobStatusFailed
	job.Retryable = retryable
	job.Error = failure.Error()
	job.Message = job.Error
	job.UpdatedAt = time.Now().UTC()
	job.CompletedAt = job.UpdatedAt
	return s.store.SaveWorkspaceJob(ctx, tenantID, job)
}

func (s *Server) recoverWorkspaceJobs(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}

	tenants, err := s.store.ListTenants(ctx)
	if err != nil {
		return fmt.Errorf("recover workspace jobs: list tenants: %w", err)
	}

	for _, tenant := range tenants {
		tenantCtx := store.ContextWithTenantID(ctx, tenant.ID)
		workspaces, err := s.store.ListWorkspaces(tenantCtx, tenant.ID, 0)
		if err != nil {
			return fmt.Errorf("recover workspace jobs: list workspaces for tenant %s: %w", tenant.ID, err)
		}

		for _, workspace := range workspaces {
			jobs, err := s.store.ListWorkspaceJobs(tenantCtx, tenant.ID, workspace.ID, 0)
			if err != nil {
				return fmt.Errorf("recover workspace jobs: list jobs for workspace %s: %w", workspace.ID, err)
			}
			for _, job := range jobs {
				if job.Status != models.WorkspaceJobStatusQueued && job.Status != models.WorkspaceJobStatusRunning {
					continue
				}
				job.Status = models.WorkspaceJobStatusQueued
				job.StartedAt = time.Time{}
				job.CompletedAt = time.Time{}
				job.UpdatedAt = time.Now().UTC()
				job.Message = "Recovered queued workspace job after API restart."
				job.Error = ""
				job.Retryable = false
				if err := s.store.SaveWorkspaceJob(tenantCtx, tenant.ID, job); err != nil {
					return fmt.Errorf("recover workspace jobs: save recovered job %s: %w", job.ID, err)
				}
				go s.runWorkspaceJob(tenant.ID, workspace.ID, job.ID)
			}
		}
	}
	return nil
}

func (s *Server) workspaceConnectorCatalog() (*connectorcatalog.Catalog, error) {
	if s != nil && s.catalog != nil {
		return s.catalog, nil
	}
	catalog, err := connectorcatalog.New(nil)
	if err != nil {
		return nil, fmt.Errorf("open connector catalog: %w", err)
	}
	return catalog, nil
}

func (s *Server) workspaceInventory(ctx context.Context, tenantID string, workspace models.PilotWorkspace) (*models.DiscoveryResult, error) {
	if len(workspace.Snapshots) == 0 {
		return &models.DiscoveryResult{DiscoveredAt: time.Now().UTC()}, nil
	}

	items := append([]models.WorkspaceSnapshot(nil), workspace.Snapshots...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].DiscoveredAt.After(items[j].DiscoveredAt)
	})

	merged := &models.DiscoveryResult{
		VMs:           make([]models.VirtualMachine, 0),
		Networks:      make([]models.NetworkInfo, 0),
		Datastores:    make([]models.DatastoreInfo, 0),
		Hosts:         make([]models.HostInfo, 0),
		Clusters:      make([]models.ClusterInfo, 0),
		ResourcePools: make([]models.ResourcePoolInfo, 0),
		DiscoveredAt:  items[0].DiscoveredAt,
	}

	for _, snapshot := range items {
		result, err := s.store.GetSnapshot(ctx, tenantID, snapshot.SnapshotID)
		if err != nil {
			return nil, fmt.Errorf("load workspace snapshot %s: %w", snapshot.SnapshotID, err)
		}
		merged.VMs = append(merged.VMs, result.VMs...)
		merged.Networks = append(merged.Networks, result.Networks...)
		merged.Datastores = append(merged.Datastores, result.Datastores...)
		merged.Hosts = append(merged.Hosts, result.Hosts...)
		merged.Clusters = append(merged.Clusters, result.Clusters...)
		merged.ResourcePools = append(merged.ResourcePools, result.ResourcePools...)
	}

	return merged, nil
}

func decodeWorkspaceJobRequest(payload json.RawMessage) (workspaceJobCreateRequest, error) {
	if len(payload) == 0 {
		return workspaceJobCreateRequest{}, nil
	}

	var request workspaceJobCreateRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return workspaceJobCreateRequest{}, fmt.Errorf("decode workspace job input: %w", err)
	}
	return request, nil
}

func buildWorkspaceMigrationSpec(workspace models.PilotWorkspace, inventory *models.DiscoveryResult, selectedIDs []string) (*migratepkg.MigrationSpec, error) {
	if inventory == nil {
		return nil, fmt.Errorf("inventory is required")
	}
	if len(workspace.SourceConnections) == 0 {
		return nil, fmt.Errorf("at least one source connection is required")
	}
	if workspace.TargetAssumptions.Platform == "" {
		return nil, fmt.Errorf("target assumptions platform is required")
	}
	if strings.TrimSpace(workspace.TargetAssumptions.Address) == "" {
		return nil, fmt.Errorf("target assumptions address is required")
	}

	source := workspace.SourceConnections[0]
	selectors, err := buildWorkspaceSelectors(workspace, inventory, selectedIDs)
	if err != nil {
		return nil, err
	}

	planName := strings.TrimSpace(workspace.PlanSettings.Name)
	if planName == "" {
		planName = workspace.Name
	}
	parallel := workspace.PlanSettings.Parallel
	if parallel <= 0 {
		parallel = 1
	}
	waveSize := workspace.PlanSettings.WaveSize
	if waveSize <= 0 {
		waveSize = parallel
	}

	return &migratepkg.MigrationSpec{
		Name: planName,
		Source: migratepkg.SourceSpec{
			Address:       source.Address,
			Platform:      source.Platform,
			CredentialRef: source.CredentialRef,
		},
		Target: migratepkg.TargetSpec{
			Address:        workspace.TargetAssumptions.Address,
			Platform:       workspace.TargetAssumptions.Platform,
			CredentialRef:  workspace.TargetAssumptions.CredentialRef,
			DefaultHost:    workspace.TargetAssumptions.DefaultHost,
			DefaultStorage: workspace.TargetAssumptions.DefaultStorage,
		},
		Workloads: selectors,
		Options: migratepkg.MigrationOptions{
			DryRun:     true,
			Parallel:   parallel,
			VerifyBoot: workspace.PlanSettings.VerifyBoot,
			Window: migratepkg.ExecutionWindow{
				NotBefore: workspace.PlanSettings.WindowStart,
				NotAfter:  workspace.PlanSettings.WindowEnd,
			},
			Approval: migratepkg.ApprovalGate{
				Required:   workspace.PlanSettings.ApprovalRequired,
				ApprovedBy: workspace.PlanSettings.ApprovedBy,
				Ticket:     workspace.PlanSettings.ApprovalTicket,
				ApprovedAt: approvalTime(workspace.PlanSettings),
			},
			Waves: migratepkg.WaveStrategy{
				Size:            waveSize,
				DependencyAware: workspace.PlanSettings.DependencyAware,
			},
		},
	}, nil
}

func buildWorkspaceSelectors(workspace models.PilotWorkspace, inventory *models.DiscoveryResult, selectedIDs []string) ([]migratepkg.WorkloadSelector, error) {
	overrides := migratepkg.WorkloadOverrides{
		TargetHost:    workspace.TargetAssumptions.DefaultHost,
		TargetStorage: workspace.TargetAssumptions.DefaultStorage,
	}
	if strings.TrimSpace(workspace.TargetAssumptions.DefaultNetwork) != "" {
		overrides.NetworkMap = map[string]string{
			"default": strings.TrimSpace(workspace.TargetAssumptions.DefaultNetwork),
		}
	}

	if len(selectedIDs) == 0 {
		return []migratepkg.WorkloadSelector{{
			Match:     migratepkg.MatchCriteria{NamePattern: "*"},
			Overrides: overrides,
		}}, nil
	}

	vmIndex := make(map[string]models.VirtualMachine, len(inventory.VMs)*2)
	for _, vm := range inventory.VMs {
		vmIndex[strings.ToLower(strings.TrimSpace(vm.ID))] = vm
		vmIndex[strings.ToLower(strings.TrimSpace(vm.Name))] = vm
	}

	selectors := make([]migratepkg.WorkloadSelector, 0, len(selectedIDs))
	seenNames := make(map[string]struct{}, len(selectedIDs))
	for _, selectedID := range selectedIDs {
		vm, ok := vmIndex[strings.ToLower(strings.TrimSpace(selectedID))]
		if !ok {
			return nil, fmt.Errorf("selected workload %q was not found in workspace inventory", selectedID)
		}
		key := strings.ToLower(strings.TrimSpace(vm.Name))
		if _, ok := seenNames[key]; ok {
			continue
		}
		seenNames[key] = struct{}{}
		selectors = append(selectors, migratepkg.WorkloadSelector{
			Match: migratepkg.MatchCriteria{
				NamePattern: "regex:^" + regexp.QuoteMeta(vm.Name) + "$",
			},
			Overrides: overrides,
		})
	}

	return selectors, nil
}

func deriveWorkspaceReadiness(result *lifecycle.SimulationResult, selectedCount int, includeAll bool, totalInventory int) *models.WorkspaceReadinessResult {
	if result == nil {
		return nil
	}

	workloadCount := selectedCount
	if includeAll {
		workloadCount = totalInventory
	}
	readiness := &models.WorkspaceReadinessResult{
		GeneratedAt:           time.Now().UTC(),
		Status:                models.WorkspaceReadinessStatusReady,
		SelectedWorkloadCount: workloadCount,
	}
	if result.RecommendationReport != nil {
		readiness.RecommendationCount = len(result.RecommendationReport.Recommendations)
		if readiness.RecommendationCount > 0 {
			readiness.WarningIssues = append(readiness.WarningIssues, fmt.Sprintf("%d remediation recommendation(s) remain open", readiness.RecommendationCount))
		}
	}
	if result.PolicyReport != nil {
		readiness.PolicyViolationCount = len(result.PolicyReport.Violations)
		for _, violation := range result.PolicyReport.Violations {
			if violation.Severity == lifecycle.PolicySeverityEnforce {
				readiness.BlockingIssues = append(readiness.BlockingIssues, fmt.Sprintf("%s violates enforce-level policy %s", violation.VM.Name, violation.Policy.Name))
				continue
			}
			readiness.WarningIssues = append(readiness.WarningIssues, fmt.Sprintf("%s requires review for policy %s", violation.VM.Name, violation.Policy.Name))
		}
	}
	if result.MovedVMs == 0 {
		readiness.BlockingIssues = append(readiness.BlockingIssues, "No workloads matched the simulation scope")
	}
	if len(readiness.BlockingIssues) > 0 {
		readiness.Status = models.WorkspaceReadinessStatusBlocked
	} else if len(readiness.WarningIssues) > 0 {
		readiness.Status = models.WorkspaceReadinessStatusAttention
	}
	return readiness
}

func syncWorkspacePlanApproval(workspace *models.PilotWorkspace, createdAt time.Time) {
	if workspace == nil || !workspace.PlanSettings.ApprovalRequired {
		return
	}

	status := models.WorkspaceApprovalStatusPending
	if strings.TrimSpace(workspace.PlanSettings.ApprovedBy) != "" {
		status = models.WorkspaceApprovalStatusApproved
	}

	for index := range workspace.Approvals {
		if strings.EqualFold(workspace.Approvals[index].Stage, "plan") {
			workspace.Approvals[index].Status = status
			workspace.Approvals[index].ApprovedBy = strings.TrimSpace(workspace.PlanSettings.ApprovedBy)
			workspace.Approvals[index].Ticket = strings.TrimSpace(workspace.PlanSettings.ApprovalTicket)
			if workspace.Approvals[index].CreatedAt.IsZero() {
				workspace.Approvals[index].CreatedAt = createdAt
			}
			return
		}
	}

	workspace.Approvals = append(workspace.Approvals, models.WorkspaceApproval{
		ID:         uuid.NewString(),
		Stage:      "plan",
		Status:     status,
		ApprovedBy: strings.TrimSpace(workspace.PlanSettings.ApprovedBy),
		Ticket:     strings.TrimSpace(workspace.PlanSettings.ApprovalTicket),
		CreatedAt:  createdAt,
	})
}

func renderWorkspaceReportMarkdown(document workspaceReportDocument) string {
	workspace := document.Workspace
	var builder strings.Builder
	builder.WriteString("# Pilot Workspace Report\n\n")
	_, _ = fmt.Fprintf(&builder, "- Workspace: %s\n", workspace.Name)
	_, _ = fmt.Fprintf(&builder, "- Workspace ID: %s\n", workspace.ID)
	if strings.TrimSpace(workspace.Description) != "" {
		_, _ = fmt.Fprintf(&builder, "- Description: %s\n", workspace.Description)
	}
	_, _ = fmt.Fprintf(&builder, "- Status: %s\n", workspace.Status)
	_, _ = fmt.Fprintf(&builder, "- Exported at: %s\n", document.ExportedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(&builder, "- Sources: %d\n", len(workspace.SourceConnections))
	_, _ = fmt.Fprintf(&builder, "- Snapshots: %d\n", len(workspace.Snapshots))
	if workspace.Readiness != nil {
		_, _ = fmt.Fprintf(&builder, "- Readiness: %s\n", workspace.Readiness.Status)
	}
	if workspace.SavedPlan != nil {
		_, _ = fmt.Fprintf(&builder, "- Saved plan: %s\n", workspace.SavedPlan.MigrationID)
	}
	builder.WriteString("\n## Source Connections\n\n")
	if len(workspace.SourceConnections) == 0 {
		builder.WriteString("No source connections recorded.\n")
	} else {
		for _, source := range workspace.SourceConnections {
			_, _ = fmt.Fprintf(&builder, "- %s (%s) at `%s`\n", source.Name, source.Platform, source.Address)
		}
	}
	builder.WriteString("\n## Discovery Snapshots\n\n")
	if len(workspace.Snapshots) == 0 {
		builder.WriteString("No snapshots have been saved for this workspace.\n")
	} else {
		for _, snapshot := range workspace.Snapshots {
			_, _ = fmt.Fprintf(&builder, "- %s captured %d workload(s) at %s\n", firstNonEmpty(snapshot.Source, snapshot.SourceConnectionID), snapshot.VMCount, snapshot.DiscoveredAt.Format(time.RFC3339))
		}
	}
	builder.WriteString("\n## Target Assumptions\n\n")
	_, _ = fmt.Fprintf(&builder, "- Platform: %s\n", workspace.TargetAssumptions.Platform)
	_, _ = fmt.Fprintf(&builder, "- Address: %s\n", workspace.TargetAssumptions.Address)
	_, _ = fmt.Fprintf(&builder, "- Default host: %s\n", workspace.TargetAssumptions.DefaultHost)
	_, _ = fmt.Fprintf(&builder, "- Default storage: %s\n", workspace.TargetAssumptions.DefaultStorage)
	_, _ = fmt.Fprintf(&builder, "- Default network: %s\n", workspace.TargetAssumptions.DefaultNetwork)
	if strings.TrimSpace(workspace.TargetAssumptions.Notes) != "" {
		_, _ = fmt.Fprintf(&builder, "- Notes: %s\n", workspace.TargetAssumptions.Notes)
	}
	builder.WriteString("\n## Dependency Graph\n\n")
	if workspace.Graph == nil {
		builder.WriteString("Dependency graph output has not been generated yet.\n")
	} else {
		_, _ = fmt.Fprintf(&builder, "- Generated at: %s\n", workspace.Graph.GeneratedAt.Format(time.RFC3339))
		_, _ = fmt.Fprintf(&builder, "- Nodes: %d\n", workspace.Graph.NodeCount)
		_, _ = fmt.Fprintf(&builder, "- Edges: %d\n", workspace.Graph.EdgeCount)
	}
	builder.WriteString("\n## Simulation\n\n")
	if workspace.Simulation == nil {
		builder.WriteString("Simulation output has not been generated yet.\n")
	} else {
		_, _ = fmt.Fprintf(&builder, "- Generated at: %s\n", workspace.Simulation.GeneratedAt.Format(time.RFC3339))
		_, _ = fmt.Fprintf(&builder, "- Target platform: %s\n", workspace.Simulation.TargetPlatform)
		_, _ = fmt.Fprintf(&builder, "- Workloads moved: %d\n", workspace.Simulation.MovedVMs)
		_, _ = fmt.Fprintf(&builder, "- Selected workload IDs: %s\n", strings.Join(workspace.Simulation.SelectedWorkloadIDs, ", "))
	}
	builder.WriteString("\n## Readiness\n\n")
	if workspace.Readiness == nil {
		builder.WriteString("Readiness has not been generated yet.\n")
	} else {
		_, _ = fmt.Fprintf(&builder, "- Status: %s\n", workspace.Readiness.Status)
		_, _ = fmt.Fprintf(&builder, "- Selected workloads: %d\n", workspace.Readiness.SelectedWorkloadCount)
		_, _ = fmt.Fprintf(&builder, "- Recommendations: %d\n", workspace.Readiness.RecommendationCount)
		_, _ = fmt.Fprintf(&builder, "- Policy violations: %d\n", workspace.Readiness.PolicyViolationCount)
		for _, issue := range workspace.Readiness.BlockingIssues {
			_, _ = fmt.Fprintf(&builder, "- Blocker: %s\n", issue)
		}
		for _, issue := range workspace.Readiness.WarningIssues {
			_, _ = fmt.Fprintf(&builder, "- Warning: %s\n", issue)
		}
	}
	builder.WriteString("\n## Saved Plan\n\n")
	if workspace.SavedPlan == nil {
		builder.WriteString("No saved migration plan is attached to this workspace.\n")
	} else {
		_, _ = fmt.Fprintf(&builder, "- Migration ID: %s\n", workspace.SavedPlan.MigrationID)
		_, _ = fmt.Fprintf(&builder, "- Spec name: %s\n", workspace.SavedPlan.SpecName)
		_, _ = fmt.Fprintf(&builder, "- Workloads: %d\n", workspace.SavedPlan.WorkloadCount)
		_, _ = fmt.Fprintf(&builder, "- Generated at: %s\n", workspace.SavedPlan.GeneratedAt.Format(time.RFC3339))
	}
	builder.WriteString("\n## Approvals\n\n")
	if len(workspace.Approvals) == 0 {
		builder.WriteString("No approvals recorded.\n")
	} else {
		for _, approval := range workspace.Approvals {
			_, _ = fmt.Fprintf(&builder, "- %s: %s (%s) ticket=%s created=%s\n", approval.Stage, approval.Status, firstNonEmpty(approval.ApprovedBy, "unassigned"), firstNonEmpty(approval.Ticket, "n/a"), approval.CreatedAt.Format(time.RFC3339))
		}
	}
	builder.WriteString("\n## Notes\n\n")
	if len(workspace.Notes) == 0 {
		builder.WriteString("No operator notes recorded.\n")
	} else {
		for _, note := range workspace.Notes {
			_, _ = fmt.Fprintf(&builder, "- [%s] %s at %s: %s\n", note.Kind, note.Author, note.CreatedAt.Format(time.RFC3339), note.Body)
		}
	}
	builder.WriteString("\n## Background Jobs\n\n")
	if len(document.Jobs) == 0 {
		builder.WriteString("No background jobs recorded.\n")
	} else {
		for _, job := range document.Jobs {
			_, _ = fmt.Fprintf(&builder, "- %s: %s requested_by=%s requested_at=%s correlation_id=%s\n", job.Type, job.Status, firstNonEmpty(job.RequestedBy, "unknown"), job.RequestedAt.Format(time.RFC3339), firstNonEmpty(job.CorrelationID, "n/a"))
			if strings.TrimSpace(job.Message) != "" {
				_, _ = fmt.Fprintf(&builder, "  message: %s\n", job.Message)
			}
			if strings.TrimSpace(job.Error) != "" {
				_, _ = fmt.Fprintf(&builder, "  error: %s\n", job.Error)
			}
		}
	}
	builder.WriteString("\n## Report History\n\n")
	if len(workspace.Reports) == 0 {
		builder.WriteString("No report exports recorded.\n")
	} else {
		for _, report := range workspace.Reports {
			_, _ = fmt.Fprintf(&builder, "- %s (%s) file=%s correlation_id=%s exported_at=%s\n", report.Name, report.Format, report.FileName, firstNonEmpty(report.CorrelationID, "n/a"), report.ExportedAt.Format(time.RFC3339))
		}
	}
	return builder.String()
}

func parseLimitQuery(r *http.Request, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func workspaceSourceConnections(workspace models.PilotWorkspace, selectedIDs []string) []models.WorkspaceSourceConnection {
	if len(selectedIDs) == 0 {
		return append([]models.WorkspaceSourceConnection(nil), workspace.SourceConnections...)
	}

	allowed := make(map[string]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		allowed[strings.TrimSpace(id)] = struct{}{}
	}
	items := make([]models.WorkspaceSourceConnection, 0, len(selectedIDs))
	for _, source := range workspace.SourceConnections {
		if _, ok := allowed[source.ID]; ok {
			items = append(items, source)
		}
	}
	return items
}

func workspaceSelectedWorkloadIDs(workspace models.PilotWorkspace, selectedIDs []string) []string {
	if len(selectedIDs) > 0 {
		return append([]string(nil), selectedIDs...)
	}
	return append([]string(nil), workspace.SelectedWorkloadIDs...)
}

func workspaceJobQueuedMessage(jobType models.WorkspaceJobType) string {
	switch jobType {
	case models.WorkspaceJobTypeDiscovery:
		return "Discovery job queued."
	case models.WorkspaceJobTypeGraph:
		return "Dependency graph job queued."
	case models.WorkspaceJobTypeSimulation:
		return "Simulation job queued."
	case models.WorkspaceJobTypePlan:
		return "Plan generation job queued."
	default:
		return "Workspace job queued."
	}
}

func workspaceJobRunningMessage(jobType models.WorkspaceJobType) string {
	switch jobType {
	case models.WorkspaceJobTypeDiscovery:
		return "Running discovery across workspace sources."
	case models.WorkspaceJobTypeGraph:
		return "Generating workspace dependency graph."
	case models.WorkspaceJobTypeSimulation:
		return "Running workspace readiness simulation."
	case models.WorkspaceJobTypePlan:
		return "Generating saved migration plan."
	default:
		return "Running workspace job."
	}
}

func advanceWorkspaceStatus(current, next models.PilotWorkspaceStatus) models.PilotWorkspaceStatus {
	statusWeight := map[models.PilotWorkspaceStatus]int{
		models.PilotWorkspaceStatusDraft:      0,
		models.PilotWorkspaceStatusDiscovered: 1,
		models.PilotWorkspaceStatusGraphReady: 2,
		models.PilotWorkspaceStatusSimulated:  3,
		models.PilotWorkspaceStatusPlanned:    4,
		models.PilotWorkspaceStatusReported:   5,
	}
	if current == "" {
		return next
	}
	if statusWeight[next] > statusWeight[current] {
		return next
	}
	return current
}

func upsertWorkspaceSnapshot(workspace *models.PilotWorkspace, snapshot models.WorkspaceSnapshot) {
	if workspace == nil {
		return
	}
	for index := range workspace.Snapshots {
		if workspace.Snapshots[index].SourceConnectionID == snapshot.SourceConnectionID && snapshot.SourceConnectionID != "" {
			workspace.Snapshots[index] = snapshot
			return
		}
	}
	workspace.Snapshots = append(workspace.Snapshots, snapshot)
}

func updateWorkspaceSourceConnectionDiscovery(workspace *models.PilotWorkspace, snapshot models.WorkspaceSnapshot) {
	if workspace == nil {
		return
	}
	for index := range workspace.SourceConnections {
		if workspace.SourceConnections[index].ID != snapshot.SourceConnectionID {
			continue
		}
		workspace.SourceConnections[index].LastSnapshotID = snapshot.SnapshotID
		workspace.SourceConnections[index].LastDiscoveredAt = snapshot.DiscoveredAt
		return
	}
}

func approvalTime(settings models.WorkspacePlanSettings) time.Time {
	if strings.TrimSpace(settings.ApprovedBy) == "" {
		return time.Time{}
	}
	return time.Now().UTC()
}

func workspaceReportExtension(format string) string {
	if format == "json" {
		return "json"
	}
	return "md"
}

func workspaceReportSlug(workspace models.PilotWorkspace) string {
	name := strings.ToLower(strings.TrimSpace(workspace.Name))
	if name == "" {
		name = workspace.ID
	}
	name = strings.Map(func(value rune) rune {
		switch {
		case value >= 'a' && value <= 'z':
			return value
		case value >= '0' && value <= '9':
			return value
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, "-")
	if name == "" {
		return "pilot-workspace"
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isRetryableWorkspaceJobError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "timeout") || strings.Contains(message, "tempor")
}
