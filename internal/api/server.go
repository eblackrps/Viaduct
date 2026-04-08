package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/connectorcatalog"
	"github.com/eblackrps/viaduct/internal/connectors"
	pluginhost "github.com/eblackrps/viaduct/internal/connectors/plugin"
	"github.com/eblackrps/viaduct/internal/deps"
	"github.com/eblackrps/viaduct/internal/discovery"
	"github.com/eblackrps/viaduct/internal/lifecycle"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"github.com/google/uuid"
)

type tenantCreateRequest struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	APIKey    string             `json:"api_key"`
	CreatedAt time.Time          `json:"created_at"`
	Active    *bool              `json:"active,omitempty"`
	Settings  map[string]string  `json:"settings,omitempty"`
	Quotas    models.TenantQuota `json:"quotas,omitempty"`
}

type migrationExecutionRequest struct {
	ApprovedBy string `json:"approved_by"`
	Ticket     string `json:"ticket"`
}

type migrationCommandResponse struct {
	MigrationID    string                    `json:"migration_id"`
	Action         string                    `json:"action"`
	OperationState string                    `json:"operation_state"`
	LifecycleState string                    `json:"lifecycle_state,omitempty"`
	Phase          migratepkg.MigrationPhase `json:"phase,omitempty"`
	AcceptedAt     time.Time                 `json:"accepted_at"`
	RequestID      string                    `json:"request_id"`
}

type tenantSummary struct {
	TenantID            string             `json:"tenant_id"`
	WorkloadCount       int                `json:"workload_count"`
	SnapshotCount       int                `json:"snapshot_count"`
	ActiveMigrations    int                `json:"active_migrations"`
	CompletedMigrations int                `json:"completed_migrations"`
	FailedMigrations    int                `json:"failed_migrations"`
	PendingApprovals    int                `json:"pending_approvals"`
	RecommendationCount int                `json:"recommendation_count"`
	PlatformCounts      map[string]int     `json:"platform_counts"`
	LastDiscoveryAt     time.Time          `json:"last_discovery_at,omitempty"`
	Quotas              models.TenantQuota `json:"quotas,omitempty"`
	SnapshotQuotaFree   int                `json:"snapshot_quota_free,omitempty"`
	MigrationQuotaFree  int                `json:"migration_quota_free,omitempty"`
}

type buildInfo struct {
	version string
	commit  string
	date    string
}

type aboutResponse struct {
	Name                 string                    `json:"name"`
	APIVersion           string                    `json:"api_version"`
	Version              string                    `json:"version"`
	Commit               string                    `json:"commit"`
	BuiltAt              string                    `json:"built_at"`
	GoVersion            string                    `json:"go_version"`
	PluginProtocol       string                    `json:"plugin_protocol"`
	SupportedPlatforms   []models.Platform         `json:"supported_platforms"`
	SupportedPermissions []models.TenantPermission `json:"supported_permissions"`
	StoreBackend         string                    `json:"store_backend"`
	StoreSchemaVersion   int                       `json:"store_schema_version,omitempty"`
	PersistentStore      bool                      `json:"persistent_store"`
}

// Server serves Viaduct REST API endpoints for discovery, migration, and lifecycle workflows.
type Server struct {
	engine               *discovery.Engine
	store                store.Store
	port                 int
	adminAPIKey          string
	catalog              *connectorcatalog.Catalog
	metrics              *apiMetrics
	rateLimiter          *tenantRateLimiter
	build                buildInfo
	backups              *models.BackupDiscoveryResult
	costEngine           *lifecycle.CostEngine
	policyEngine         *lifecycle.PolicyEngine
	recommendationEngine *lifecycle.RecommendationEngine
	driftDetector        *lifecycle.DriftDetector
	resolveConfig        func(platform models.Platform, address, credentialRef string) connectors.Config

	mu    sync.RWMutex
	specs map[string]*migratepkg.MigrationSpec
}

// NewServer creates a REST API server on the supplied port.
func NewServer(engine *discovery.Engine, stateStore store.Store, port int, catalog *connectorcatalog.Catalog) *Server {
	if stateStore == nil {
		stateStore = store.NewMemoryStore()
	}
	if port == 0 {
		port = 8080
	}
	if engine == nil {
		engine = discovery.NewEngine()
	}

	costEngine := lifecycle.NewCostEngine()
	if profiles, err := lifecycle.LoadCostProfilesDir(filepath.Join("configs", "cost-profiles")); err == nil {
		for _, profile := range profiles {
			costEngine.AddProfile(*profile)
		}
	}

	policyEngine := lifecycle.NewPolicyEngine(costEngine)
	_ = policyEngine.LoadPolicies(filepath.Join("configs", "policies"))

	return &Server{
		engine:               engine,
		store:                stateStore,
		port:                 port,
		adminAPIKey:          os.Getenv("VIADUCT_ADMIN_KEY"),
		catalog:              catalog,
		metrics:              newAPIMetrics(),
		rateLimiter:          newTenantRateLimiter(300, time.Minute),
		build:                buildInfo{version: "dev", commit: "none", date: "unknown"},
		costEngine:           costEngine,
		policyEngine:         policyEngine,
		recommendationEngine: lifecycle.NewRecommendationEngine(costEngine, policyEngine),
		driftDetector:        lifecycle.NewDriftDetector(stateStore, policyEngine, lifecycle.DriftConfig{}),
		resolveConfig: func(platform models.Platform, address, credentialRef string) connectors.Config {
			return connectors.Config{Address: address}
		},
		specs: make(map[string]*migratepkg.MigrationSpec),
	}
}

// SetBuildInfo configures operator-visible release metadata exposed by the API server.
func (s *Server) SetBuildInfo(version, commit, date string) {
	if s == nil {
		return
	}
	if strings.TrimSpace(version) != "" {
		s.build.version = strings.TrimSpace(version)
	}
	if strings.TrimSpace(commit) != "" {
		s.build.commit = strings.TrimSpace(commit)
	}
	if strings.TrimSpace(date) != "" {
		s.build.date = strings.TrimSpace(date)
	}
}

// SetConnectorConfigResolver configures how migration specs resolve connector credentials and transport settings.
func (s *Server) SetConnectorConfigResolver(resolver func(platform models.Platform, address, credentialRef string) connectors.Config) {
	if s == nil || resolver == nil {
		return
	}
	s.resolveConfig = resolver
}

// Start runs the HTTP server until the context is canceled.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/about", s.handleAbout)
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.Handle("/api/v1/admin/tenants", AdminAuthMiddleware(s.adminAPIKey, http.HandlerFunc(s.handleAdminTenants)))
	mux.Handle("/api/v1/admin/tenants/", AdminAuthMiddleware(s.adminAPIKey, http.HandlerFunc(s.handleAdminTenantByID)))

	tenantMux := http.NewServeMux()
	tenantRoute := func(requiredRole models.TenantRole, requiredPermission models.TenantPermission, handler http.HandlerFunc) http.Handler {
		return RequireTenantRole(requiredRole, RequireTenantPermission(requiredPermission, handler))
	}
	tenantMux.Handle("/api/v1/inventory", tenantRoute(models.TenantRoleViewer, models.TenantPermissionInventoryRead, s.handleInventory))
	tenantMux.Handle("/api/v1/audit", tenantRoute(models.TenantRoleViewer, models.TenantPermissionReportsRead, s.handleAudit))
	tenantMux.Handle("/api/v1/snapshots", tenantRoute(models.TenantRoleViewer, models.TenantPermissionInventoryRead, s.handleSnapshots))
	tenantMux.Handle("/api/v1/snapshots/", tenantRoute(models.TenantRoleViewer, models.TenantPermissionInventoryRead, s.handleSnapshotByID))
	tenantMux.Handle("/api/v1/graph", tenantRoute(models.TenantRoleViewer, models.TenantPermissionInventoryRead, s.handleGraph))
	tenantMux.Handle("/api/v1/preflight", tenantRoute(models.TenantRoleOperator, models.TenantPermissionMigrationManage, s.handlePreflight))
	tenantMux.Handle("/api/v1/migrations", tenantRoute(models.TenantRoleOperator, models.TenantPermissionMigrationManage, s.handleMigrations))
	tenantMux.Handle("/api/v1/migrations/", tenantRoute(models.TenantRoleOperator, models.TenantPermissionMigrationManage, s.handleMigrationByID))
	tenantMux.Handle("/api/v1/costs", tenantRoute(models.TenantRoleViewer, models.TenantPermissionLifecycleRead, s.handleCosts))
	tenantMux.Handle("/api/v1/policies", tenantRoute(models.TenantRoleViewer, models.TenantPermissionLifecycleRead, s.handlePolicies))
	tenantMux.Handle("/api/v1/drift", tenantRoute(models.TenantRoleViewer, models.TenantPermissionLifecycleRead, s.handleDrift))
	tenantMux.Handle("/api/v1/remediation", tenantRoute(models.TenantRoleViewer, models.TenantPermissionLifecycleRead, s.handleRemediation))
	tenantMux.Handle("/api/v1/simulation", tenantRoute(models.TenantRoleOperator, models.TenantPermissionMigrationManage, s.handleSimulation))
	tenantMux.Handle("/api/v1/summary", tenantRoute(models.TenantRoleViewer, models.TenantPermissionLifecycleRead, s.handleSummary))
	tenantMux.Handle("/api/v1/reports/", tenantRoute(models.TenantRoleViewer, models.TenantPermissionReportsRead, s.handleReports))
	tenantMux.Handle("/api/v1/tenants/current", tenantRoute(models.TenantRoleViewer, models.TenantPermissionTenantRead, s.handleCurrentTenant))
	tenantMux.Handle("/api/v1/service-accounts", tenantRoute(models.TenantRoleAdmin, models.TenantPermissionTenantManage, s.handleServiceAccounts))
	tenantMux.Handle("/api/v1/service-accounts/", tenantRoute(models.TenantRoleAdmin, models.TenantPermissionTenantManage, s.handleServiceAccountByID))

	tenantHandler := TenantAuthMiddleware(s.store, TenantRateLimitMiddleware(s.rateLimiter, tenantMux))
	mux.Handle("/api/v1/inventory", tenantHandler)
	mux.Handle("/api/v1/audit", tenantHandler)
	mux.Handle("/api/v1/snapshots", tenantHandler)
	mux.Handle("/api/v1/snapshots/", tenantHandler)
	mux.Handle("/api/v1/graph", tenantHandler)
	mux.Handle("/api/v1/preflight", tenantHandler)
	mux.Handle("/api/v1/migrations", tenantHandler)
	mux.Handle("/api/v1/migrations/", tenantHandler)
	mux.Handle("/api/v1/costs", tenantHandler)
	mux.Handle("/api/v1/policies", tenantHandler)
	mux.Handle("/api/v1/drift", tenantHandler)
	mux.Handle("/api/v1/remediation", tenantHandler)
	mux.Handle("/api/v1/simulation", tenantHandler)
	mux.Handle("/api/v1/summary", tenantHandler)
	mux.Handle("/api/v1/reports/", tenantHandler)
	mux.Handle("/api/v1/tenants/current", tenantHandler)
	mux.Handle("/api/v1/service-accounts", tenantHandler)
	mux.Handle("/api/v1/service-accounts/", tenantHandler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.withObservability(s.withCORS(mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("api server: listen and serve: %w", err)
	}

	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	diagnostics := store.Diagnostics{
		Backend:    "unknown",
		Persistent: false,
	}
	if provider, ok := s.store.(store.DiagnosticsProvider); ok {
		if reported, err := provider.Diagnostics(r.Context()); err == nil {
			diagnostics = reported
		}
	}

	writeJSON(w, http.StatusOK, aboutResponse{
		Name:                 "Viaduct",
		APIVersion:           "v1",
		Version:              s.build.version,
		Commit:               s.build.commit,
		BuiltAt:              s.build.date,
		GoVersion:            runtime.Version(),
		PluginProtocol:       pluginhost.ProtocolVersion,
		SupportedPlatforms:   s.supportedPlatforms(),
		SupportedPermissions: models.TenantRoleAdmin.DefaultPermissions(),
		StoreBackend:         diagnostics.Backend,
		StoreSchemaVersion:   diagnostics.SchemaVersion,
		PersistentStore:      diagnostics.Persistent,
	})
}

func (s *Server) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenants, err := s.store.ListTenants(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		responses := make([]adminTenantResponse, 0, len(tenants))
		for _, tenant := range tenants {
			responses = append(responses, toAdminTenantResponse(tenant, false))
		}
		writeJSON(w, http.StatusOK, responses)
	case http.MethodPost:
		defer r.Body.Close()

		var request tenantCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode tenant: %w", err).Error(), apiErrorOptions{})
			return
		}
		tenant := models.Tenant{
			ID:        request.ID,
			Name:      request.Name,
			APIKey:    request.APIKey,
			CreatedAt: request.CreatedAt,
			Settings:  request.Settings,
			Quotas:    request.Quotas,
		}
		if tenant.ID == "" {
			tenant.ID = uuid.NewString()
		}
		if tenant.APIKey == "" {
			tenant.APIKey = uuid.NewString()
		}
		if tenant.Name == "" {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "tenant name is required", apiErrorOptions{
				FieldErrors: []apiFieldError{{Path: "name", Message: "tenant name is required"}},
			})
			return
		}
		if tenant.CreatedAt.IsZero() {
			tenant.CreatedAt = time.Now().UTC()
		}
		if request.Active != nil {
			tenant.Active = *request.Active
		} else {
			tenant.Active = true
		}
		if err := s.store.CreateTenant(r.Context(), tenant); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			TenantID: tenant.ID,
			Actor:    "admin",
			Category: "admin",
			Action:   "create-tenant",
			Resource: tenant.ID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "tenant created",
		})
		writeJSON(w, http.StatusCreated, toAdminTenantResponse(tenant, true))
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleAdminTenantByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	tenantID := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/tenants/")
	if tenantID == "" || strings.Contains(tenantID, "/") {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "tenant ID is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "tenant_id", Message: "tenant ID is required"}},
		})
		return
	}
	if err := s.store.DeleteTenant(r.Context(), tenantID); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
		return
	}
	s.recordAuditEvent(r, models.AuditEvent{
		TenantID: tenantID,
		Actor:    "admin",
		Category: "admin",
		Action:   "delete-tenant",
		Resource: tenantID,
		Outcome:  models.AuditOutcomeSuccess,
		Message:  "tenant deleted",
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	platform := models.Platform(r.URL.Query().Get("platform"))
	result, err := s.latestInventory(r.Context(), platform)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	items, err := s.store.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 100)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleSnapshotByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	snapshotID := strings.TrimPrefix(r.URL.Path, "/api/v1/snapshots/")
	result, err := s.store.GetSnapshot(r.Context(), store.TenantIDFromContext(r.Context()), snapshotID)
	if err != nil {
		writeAPIError(w, r, http.StatusNotFound, "invalid_request", err.Error(), apiErrorOptions{
			Details: map[string]any{
				"snapshot_id": snapshotID,
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, deps.BuildGraph(inventory, s.backups))
}

func (s *Server) handlePreflight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	spec, err := decodeSpec(r)
	if err != nil {
		var validationErr specValidationError
		if errors.As(err, &validationErr) {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_spec", validationErr.Error(), apiErrorOptions{
				FieldErrors: validationErr.fieldErrors,
			})
			return
		}
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
		return
	}

	sourceConnector, targetConnector, err := s.connectorsForSpec(spec)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
		return
	}

	report, err := migratepkg.NewPreflightChecker(sourceConnector, targetConnector, spec).RunAll(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleMigrations(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListMigrations(r.Context(), tenantID, 100)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		spec, err := decodeSpec(r)
		if err != nil {
			var validationErr specValidationError
			if errors.As(err, &validationErr) {
				writeAPIError(w, r, http.StatusBadRequest, "invalid_spec", validationErr.Error(), apiErrorOptions{
					FieldErrors: validationErr.fieldErrors,
				})
				return
			}
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		sourceConnector, targetConnector, err := s.connectorsForSpec(spec)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		migrationID := uuid.NewString()
		orchestrator := migratepkg.NewOrchestrator(sourceConnector, targetConnector, s.store, nil)
		orchestrator.SetIDGenerator(func() string { return migrationID })

		specCopy := *spec
		specCopy.Options.DryRun = true
		ctx := store.ContextWithTenantID(r.Context(), tenantID)
		state, err := orchestrator.Execute(ctx, &specCopy)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		s.mu.Lock()
		s.specs[s.specKey(tenantID, migrationID)] = spec
		s.mu.Unlock()
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "migration",
			Action:   "plan",
			Resource: migrationID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "migration dry-run planned",
			Details: map[string]string{
				"spec_name": spec.Name,
			},
		})

		writeJSON(w, http.StatusAccepted, state)
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleMigrationByID(w http.ResponseWriter, r *http.Request) {
	tenantID := store.TenantIDFromContext(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/migrations/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "migration ID is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "migration_id", Message: "migration ID is required"}},
		})
		return
	}

	migrationID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
			return
		}
		record, err := s.store.GetMigration(r.Context(), tenantID, migrationID)
		if err != nil {
			writeAPIError(w, r, http.StatusNotFound, "migration_not_found", err.Error(), apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		var state migratepkg.MigrationState
		if err := json.Unmarshal(record.RawJSON, &state); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		writeJSON(w, http.StatusOK, state)
		return
	}

	switch parts[1] {
	case "execute":
		if r.Method != http.MethodPost {
			writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
			return
		}

		spec, ok := s.lookupSpec(tenantID, migrationID)
		if !ok {
			writeAPIError(w, r, http.StatusNotFound, "migration_not_found", "migration spec not found", apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		sourceConnector, targetConnector, err := s.connectorsForSpec(spec)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}
		executionRequest, err := decodeExecutionRequest(r)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		executionSpec := *spec
		executionSpec.Options.DryRun = false
		executionSpec.Options.Approval = applyExecutionApproval(executionSpec.Options.Approval, executionRequest)
		if err := validateExecutionRequest(executionSpec, time.Now().UTC()); err != nil {
			s.recordAuditEvent(r, models.AuditEvent{
				Category: "migration",
				Action:   "execute",
				Resource: migrationID,
				Outcome:  models.AuditOutcomeFailure,
				Message:  err.Error(),
			})
			writeAPIError(w, r, http.StatusConflict, executionErrorCode(err), err.Error(), apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		go func(spec *migratepkg.MigrationSpec, id, tenantID string) {
			orchestrator := migratepkg.NewOrchestrator(sourceConnector, targetConnector, s.store, nil)
			orchestrator.SetIDGenerator(func() string { return id })
			ctx := store.ContextWithTenantID(context.Background(), tenantID)
			_, _ = orchestrator.Execute(ctx, spec)
		}(&executionSpec, migrationID, tenantID)
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "migration",
			Action:   "execute",
			Resource: migrationID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "migration execution started",
			Details: map[string]string{
				"approved_by": executionSpec.Options.Approval.ApprovedBy,
			},
		})

		writeJSON(w, http.StatusAccepted, s.newMigrationCommandResponse(r, tenantID, migrationID, "execute", "executing"))
	case "resume":
		if r.Method != http.MethodPost {
			writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
			return
		}

		spec, ok := s.lookupSpec(tenantID, migrationID)
		if !ok {
			writeAPIError(w, r, http.StatusNotFound, "migration_not_found", "migration spec not found", apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		sourceConnector, targetConnector, err := s.connectorsForSpec(spec)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}
		resumeRequest, err := decodeExecutionRequest(r)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		resumeSpec := *spec
		resumeSpec.Options.DryRun = false
		resumeSpec.Options.Approval = applyExecutionApproval(resumeSpec.Options.Approval, resumeRequest)
		if err := validateExecutionRequest(resumeSpec, time.Now().UTC()); err != nil {
			s.recordAuditEvent(r, models.AuditEvent{
				Category: "migration",
				Action:   "resume",
				Resource: migrationID,
				Outcome:  models.AuditOutcomeFailure,
				Message:  err.Error(),
			})
			writeAPIError(w, r, http.StatusConflict, executionErrorCode(err), err.Error(), apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		go func(spec *migratepkg.MigrationSpec, id, tenantID string) {
			orchestrator := migratepkg.NewOrchestrator(sourceConnector, targetConnector, s.store, nil)
			ctx := store.ContextWithTenantID(context.Background(), tenantID)
			_, _ = orchestrator.Resume(ctx, id, spec)
		}(&resumeSpec, migrationID, tenantID)
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "migration",
			Action:   "resume",
			Resource: migrationID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "migration resume started",
		})

		writeJSON(w, http.StatusAccepted, s.newMigrationCommandResponse(r, tenantID, migrationID, "resume", "executing"))
	case "rollback":
		if r.Method != http.MethodPost {
			writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
			return
		}

		spec, ok := s.lookupSpec(tenantID, migrationID)
		if !ok {
			writeAPIError(w, r, http.StatusNotFound, "migration_not_found", "migration spec not found", apiErrorOptions{
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}

		sourceConnector, targetConnector, err := s.connectorsForSpec(spec)
		if err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}

		result, err := migratepkg.NewRollbackManager(s.store, sourceConnector, targetConnector).Rollback(store.ContextWithTenantID(r.Context(), tenantID), migrationID)
		if err != nil {
			s.recordAuditEvent(r, models.AuditEvent{
				Category: "migration",
				Action:   "rollback",
				Resource: migrationID,
				Outcome:  models.AuditOutcomeFailure,
				Message:  err.Error(),
			})
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{
				Retryable: true,
				Details: map[string]any{
					"migration_id": migrationID,
				},
			})
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "migration",
			Action:   "rollback",
			Resource: migrationID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "migration rollback completed",
		})

		writeJSON(w, http.StatusOK, result)
	default:
		writeAPIError(w, r, http.StatusNotFound, "invalid_request", "not found", apiErrorOptions{})
	}
}

func (s *Server) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	platform := r.URL.Query().Get("platform")
	if platform == "" || platform == "all" {
		comparisons := make([]*lifecycle.PlatformComparison, 0, len(inventory.VMs))
		for _, vm := range inventory.VMs {
			comparison, err := s.costEngine.CompareVM(vm)
			if err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
				return
			}
			comparisons = append(comparisons, comparison)
		}
		writeJSON(w, http.StatusOK, comparisons)
		return
	}

	fleet, err := s.costEngine.CalculateFleetCost(models.Platform(platform), inventory.VMs)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	writeJSON(w, http.StatusOK, fleet)
}

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	report, err := s.policyEngine.Evaluate(inventory)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	baselineID := strings.TrimSpace(r.URL.Query().Get("baseline"))
	if baselineID == "" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "baseline query parameter is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "baseline", Message: "baseline query parameter is required"}},
		})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	report, err := s.driftDetector.Compare(r.Context(), baselineID, inventory)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleRemediation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	var driftReport *lifecycle.DriftReport
	if baselineID := strings.TrimSpace(r.URL.Query().Get("baseline")); baselineID != "" {
		driftReport, err = s.driftDetector.Compare(r.Context(), baselineID, inventory)
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}
	}

	report, err := s.recommendationEngine.Generate(inventory, driftReport, nil)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	defer r.Body.Close()
	var request lifecycle.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode simulation request: %w", err).Error(), apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	result, err := s.recommendationEngine.Simulate(inventory, request)
	if err != nil {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), apiErrorOptions{})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	inventory, err := s.latestInventory(r.Context(), "")
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	snapshots, err := s.store.ListSnapshots(r.Context(), store.TenantIDFromContext(r.Context()), "", 100)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	migrations, err := s.store.ListMigrations(r.Context(), store.TenantIDFromContext(r.Context()), 100)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}
	recommendations, err := s.recommendationEngine.Generate(inventory, nil, nil)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	summary := tenantSummary{
		TenantID:            store.TenantIDFromContext(r.Context()),
		WorkloadCount:       len(inventory.VMs),
		SnapshotCount:       len(snapshots),
		RecommendationCount: len(recommendations.Recommendations),
		PlatformCounts:      make(map[string]int),
	}
	if tenant, err := s.store.GetTenant(r.Context(), summary.TenantID); err == nil && tenant != nil {
		summary.Quotas = tenant.Quotas
		if tenant.Quotas.MaxSnapshots > 0 {
			summary.SnapshotQuotaFree = tenant.Quotas.MaxSnapshots - len(snapshots)
			if summary.SnapshotQuotaFree < 0 {
				summary.SnapshotQuotaFree = 0
			}
		}
	}
	if len(snapshots) > 0 {
		summary.LastDiscoveryAt = snapshots[0].DiscoveredAt
	}
	for _, vm := range inventory.VMs {
		summary.PlatformCounts[string(vm.Platform)]++
	}
	for _, migration := range migrations {
		switch migration.Phase {
		case string(migratepkg.PhaseComplete):
			summary.CompletedMigrations++
		case string(migratepkg.PhaseFailed):
			summary.FailedMigrations++
		case string(migratepkg.PhasePlan):
			record, err := s.store.GetMigration(r.Context(), store.TenantIDFromContext(r.Context()), migration.ID)
			if err == nil && pendingApprovalFromRecord(record) {
				summary.PendingApprovals++
			}
			summary.ActiveMigrations++
		case string(migratepkg.PhaseRolledBack):
		default:
			summary.ActiveMigrations++
		}
	}
	if summary.Quotas.MaxMigrations > 0 {
		summary.MigrationQuotaFree = summary.Quotas.MaxMigrations - len(migrations)
		if summary.MigrationQuotaFree < 0 {
			summary.MigrationQuotaFree = 0
		}
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) lookupSpec(tenantID, migrationID string) (*migratepkg.MigrationSpec, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	spec, ok := s.specs[s.specKey(tenantID, migrationID)]
	return spec, ok
}

func (s *Server) specKey(tenantID, migrationID string) string {
	return tenantID + ":" + migrationID
}

func (s *Server) newMigrationCommandResponse(r *http.Request, tenantID, migrationID, action, lifecycleState string) migrationCommandResponse {
	return migrationCommandResponse{
		MigrationID:    migrationID,
		Action:         action,
		OperationState: "accepted",
		LifecycleState: lifecycleState,
		Phase:          s.commandPhase(r.Context(), tenantID, migrationID, migratepkg.PhasePlan),
		AcceptedAt:     time.Now().UTC(),
		RequestID:      responseRequestID(nil, r),
	}
}

func (s *Server) commandPhase(ctx context.Context, tenantID, migrationID string, fallback migratepkg.MigrationPhase) migratepkg.MigrationPhase {
	if s == nil || s.store == nil {
		return fallback
	}
	record, err := s.store.GetMigration(ctx, tenantID, migrationID)
	if err != nil || strings.TrimSpace(record.Phase) == "" {
		return fallback
	}
	return migratepkg.MigrationPhase(record.Phase)
}

func (s *Server) latestInventory(ctx context.Context, platform models.Platform) (*models.DiscoveryResult, error) {
	tenantID := store.TenantIDFromContext(ctx)
	items, err := s.store.ListSnapshots(ctx, tenantID, platform, 0)
	if err != nil {
		return nil, fmt.Errorf("load latest inventory: %w", err)
	}

	if len(items) == 0 {
		return &models.DiscoveryResult{
			Platform:     platform,
			DiscoveredAt: time.Now().UTC(),
		}, nil
	}

	latestItems := latestSnapshotsBySource(items)
	merged := &models.DiscoveryResult{
		Platform:      platform,
		VMs:           make([]models.VirtualMachine, 0),
		Networks:      make([]models.NetworkInfo, 0),
		Datastores:    make([]models.DatastoreInfo, 0),
		Hosts:         make([]models.HostInfo, 0),
		Clusters:      make([]models.ClusterInfo, 0),
		ResourcePools: make([]models.ResourcePoolInfo, 0),
		DiscoveredAt:  items[0].DiscoveredAt,
	}

	for _, item := range latestItems {
		snapshot, err := s.store.GetSnapshot(ctx, tenantID, item.ID)
		if err != nil {
			return nil, fmt.Errorf("load latest inventory snapshot %s: %w", item.ID, err)
		}
		merged.VMs = append(merged.VMs, snapshot.VMs...)
		merged.Networks = append(merged.Networks, snapshot.Networks...)
		merged.Datastores = append(merged.Datastores, snapshot.Datastores...)
		merged.Hosts = append(merged.Hosts, snapshot.Hosts...)
		merged.Clusters = append(merged.Clusters, snapshot.Clusters...)
		merged.ResourcePools = append(merged.ResourcePools, snapshot.ResourcePools...)
	}

	return merged, nil
}

func latestSnapshotsBySource(items []store.SnapshotMeta) []store.SnapshotMeta {
	if len(items) == 0 {
		return nil
	}

	selected := make([]store.SnapshotMeta, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Source)) + "|" + string(item.Platform)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, item)
	}
	return selected
}

func (s *Server) connectorsForSpec(spec *migratepkg.MigrationSpec) (connectors.Connector, connectors.Connector, error) {
	catalog := s.catalog
	if catalog == nil {
		var err error
		catalog, err = connectorcatalog.New(nil)
		if err != nil {
			return nil, nil, fmt.Errorf("open connector catalog: %w", err)
		}
	}

	resolveConfig := s.resolveConfig
	if resolveConfig == nil {
		resolveConfig = func(platform models.Platform, address, credentialRef string) connectors.Config {
			return connectors.Config{Address: address}
		}
	}

	sourceConnector, err := catalog.Build(spec.Source.Platform, resolveConfig(spec.Source.Platform, spec.Source.Address, spec.Source.CredentialRef))
	if err != nil {
		return nil, nil, err
	}

	targetConnector, err := catalog.Build(spec.Target.Platform, resolveConfig(spec.Target.Platform, spec.Target.Address, spec.Target.CredentialRef))
	if err != nil {
		return nil, nil, err
	}

	return sourceConnector, targetConnector, nil
}

func decodeSpec(r *http.Request) (*migratepkg.MigrationSpec, error) {
	defer r.Body.Close()

	var spec migratepkg.MigrationSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode migration spec: %w", err)
	}

	if errs := migratepkg.ValidateSpec(&spec); len(errs) > 0 {
		messages := make([]string, 0, len(errs))
		for _, item := range errs {
			messages = append(messages, item.Error())
		}
		return nil, specValidationError{
			message:     fmt.Sprintf("invalid migration spec: %s", strings.Join(messages, "; ")),
			fieldErrors: fieldErrorsFromValidationErrors(errs),
		}
	}

	if spec.Options.Parallel <= 0 {
		spec.Options.Parallel = 1
	}

	return &spec, nil
}

func decodeExecutionRequest(r *http.Request) (migrationExecutionRequest, error) {
	defer r.Body.Close()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return migrationExecutionRequest{}, fmt.Errorf("decode execution request: %w", err)
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return migrationExecutionRequest{}, nil
	}

	var request migrationExecutionRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return migrationExecutionRequest{}, fmt.Errorf("decode execution request: %w", err)
	}
	return request, nil
}

func applyExecutionApproval(approval migratepkg.ApprovalGate, request migrationExecutionRequest) migratepkg.ApprovalGate {
	if strings.TrimSpace(request.ApprovedBy) == "" {
		return approval
	}
	approval.ApprovedBy = strings.TrimSpace(request.ApprovedBy)
	if approval.ApprovedAt.IsZero() {
		approval.ApprovedAt = time.Now().UTC()
	}
	if strings.TrimSpace(request.Ticket) != "" {
		approval.Ticket = strings.TrimSpace(request.Ticket)
	}
	return approval
}

func validateExecutionRequest(spec migratepkg.MigrationSpec, now time.Time) error {
	if spec.Options.Approval.Required && !spec.Options.Approval.Approved() {
		return fmt.Errorf("migration requires approval before execution")
	}
	if !spec.Options.Window.NotBefore.IsZero() && now.Before(spec.Options.Window.NotBefore) {
		return fmt.Errorf("migration window opens at %s", spec.Options.Window.NotBefore.Format(time.RFC3339))
	}
	if !spec.Options.Window.NotAfter.IsZero() && now.After(spec.Options.Window.NotAfter) {
		return fmt.Errorf("migration window closed at %s", spec.Options.Window.NotAfter.Format(time.RFC3339))
	}
	return nil
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Service-Account-Key, X-Admin-Key, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) supportedPlatforms() []models.Platform {
	if s == nil || s.catalog == nil {
		return []models.Platform{
			models.PlatformHyperV,
			models.PlatformKVM,
			models.PlatformNutanix,
			models.PlatformProxmox,
			models.PlatformVMware,
		}
	}
	return s.catalog.Platforms()
}

func pendingApprovalFromRecord(record *store.MigrationRecord) bool {
	if record == nil || len(record.RawJSON) == 0 {
		return false
	}

	var state migratepkg.MigrationState
	if err := json.Unmarshal(record.RawJSON, &state); err != nil {
		return false
	}
	return state.PendingApproval
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
