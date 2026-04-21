package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

type authSessionRequest struct {
	Mode     string `json:"mode"`
	APIKey   string `json:"api_key"`
	Remember bool   `json:"remember"`
}

type authSessionResponse struct {
	SessionID string    `json:"session_id"`
	Mode      string    `json:"mode"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type authSessionRevokeRequest struct {
	SessionID string `json:"session_id"`
}

func (s *Server) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "auth session server is not configured", apiErrorOptions{Retryable: true})
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.createAuthSession(w, r)
	case http.MethodDelete:
		s.deleteAuthSession(w, r)
	default:
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
	}
}

func (s *Server) handleAuthSessionRevoke(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "auth session server is not configured", apiErrorOptions{Retryable: true})
		return
	}
	if r.Method != http.MethodPost {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}
	defer r.Body.Close()

	var request authSessionRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode auth session revoke request: %w", err).Error(), apiErrorOptions{})
		return
	}

	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "session_id is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "session_id", Message: "session_id is required"}},
		})
		return
	}

	record, secret, ok := s.authSessions.LookupByPublicID(sessionID)
	if !ok {
		writeAPIError(w, r, http.StatusNotFound, "session_not_found", "dashboard auth session was not found", apiErrorOptions{
			Details: map[string]any{"session_id": sessionID},
		})
		return
	}
	auditRequest := r.WithContext(withAwaitAudit(r.Context()))
	auditEvent := s.normalizeAuditEvent(auditRequest.Context(), models.AuditEvent{
		Actor:    "admin",
		Category: "tenant",
		Action:   "revoke-auth-session",
		Resource: record.PublicID,
		Outcome:  models.AuditOutcomeSuccess,
		Message:  "dashboard auth session revoked by administrator",
		Details: map[string]string{
			"session_id": record.PublicID,
			"auth_mode":  record.Mode,
			"reason":     "admin_revocation",
		},
	})
	if err := s.authSessions.Revoke(r.Context(), s.store, record, secret, func(ctx context.Context) error {
		return s.persistAuditEvent(ctx, auditEvent)
	}); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "session_id": record.PublicID})
}

func (s *Server) createAuthSession(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var request authSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode auth session request: %w", err).Error(), apiErrorOptions{})
		return
	}

	mode := strings.TrimSpace(request.Mode)
	if mode == "" {
		mode = "service-account"
	}

	var (
		principal AuthenticatedPrincipal
		record    authSessionRecord
		err       error
	)
	switch mode {
	case "local":
		principal, err = s.localRuntimeOperatorPrincipal(r.Context(), r)
		if err != nil {
			writeAPIError(w, r, http.StatusForbidden, "invalid_request", err.Error(), apiErrorOptions{})
			return
		}
		record, err = s.authSessions.CreateLocal(principal.Tenant.ID, principal.Role, principal.AuthMethod, request.Remember)
	case "tenant", "service-account":
		apiKey := strings.TrimSpace(request.APIKey)
		if apiKey == "" {
			writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "api key is required", apiErrorOptions{
				FieldErrors: []apiFieldError{{Path: "api_key", Message: "api key is required"}},
			})
			return
		}

		tenants, listErr := s.store.ListTenants(r.Context())
		if listErr != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", listErr.Error(), apiErrorOptions{Retryable: true})
			return
		}

		var ok bool
		switch mode {
		case "tenant":
			principal, ok = findTenantPrincipalByAPIKey(r.Context(), tenants, apiKey)
		case "service-account":
			principal, ok = findServiceAccountPrincipalByAPIKey(r.Context(), tenants, apiKey)
		}
		if !ok {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid tenant credentials", apiErrorOptions{})
			return
		}

		record, err = s.authSessions.CreateCredential(mode, principal, hashCredential(r.Context(), apiKey), request.Remember)
	default:
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "mode must be local, tenant, or service-account", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "mode", Message: "mode must be local, tenant, or service-account"}},
		})
		return
	}
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	setAuthSessionCookie(w, record, s.requestScheme(r) == "https")
	writeJSON(w, http.StatusCreated, authSessionResponse{
		SessionID: record.PublicID,
		Mode:      record.Mode,
		ExpiresAt: record.ExpiresAt,
	})

	s.recordAuditEvent(r, models.AuditEvent{
		Category: "tenant",
		Action:   "create-auth-session",
		Resource: principal.Tenant.ID,
		Outcome:  models.AuditOutcomeSuccess,
		Message:  "dashboard auth session created",
		Details: map[string]string{
			"tenant_id":  principal.Tenant.ID,
			"auth_mode":  record.Mode,
			"persistent": fmt.Sprintf("%t", request.Remember),
		},
	})
}

func (s *Server) deleteAuthSession(w http.ResponseWriter, r *http.Request) {
	if secret := readAuthSessionSecret(r); secret != "" {
		if record, ok, err := s.authSessions.LookupActive(r.Context(), s.store, secret); err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		} else if ok {
			auditRequest := r.WithContext(withAwaitAudit(r.Context()))
			auditEvent := s.normalizeAuditEvent(auditRequest.Context(), models.AuditEvent{
				Category: "tenant",
				Action:   "revoke-auth-session",
				Resource: record.PublicID,
				Outcome:  models.AuditOutcomeSuccess,
				Message:  "dashboard auth session revoked by current session holder",
				Details: map[string]string{
					"session_id": record.PublicID,
					"auth_mode":  record.Mode,
					"reason":     "self_revocation",
				},
			})
			if err := s.authSessions.Revoke(r.Context(), s.store, record, secret, func(ctx context.Context) error {
				return s.persistAuditEvent(ctx, auditEvent)
			}); err != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
				return
			}
		} else {
			s.authSessions.Delete(secret)
		}
	}
	clearAuthSessionCookie(w, s.requestScheme(r) == "https")
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) localRuntimeOperatorPrincipal(ctx context.Context, r *http.Request) (AuthenticatedPrincipal, error) {
	if s == nil || !s.localRuntimeMode {
		return AuthenticatedPrincipal{}, fmt.Errorf("local runtime operator bootstrap is not enabled on this server")
	}
	if !localRuntimeRequestAllowed(r, s.bindHost) {
		return AuthenticatedPrincipal{}, fmt.Errorf("local runtime operator bootstrap is available only for direct loopback requests to a loopback-bound runtime")
	}
	if hasConfiguredAPIKeys(ctx, s.store, s.adminAPIKey) {
		return AuthenticatedPrincipal{}, fmt.Errorf("local runtime operator bootstrap is disabled when API keys are configured")
	}

	tenants, err := s.store.ListTenants(ctx)
	if err != nil {
		return AuthenticatedPrincipal{}, fmt.Errorf("list tenants for local runtime bootstrap: %w", err)
	}
	tenant, ok := defaultTenantFallback(tenants)
	if !ok {
		return AuthenticatedPrincipal{}, fmt.Errorf("local runtime operator bootstrap requires the default fallback tenant")
	}
	return AuthenticatedPrincipal{
		Tenant:     tenant,
		Role:       models.TenantRoleOperator,
		AuthMethod: "local-runtime-session",
	}, nil
}
