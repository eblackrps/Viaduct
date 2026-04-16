package api

import (
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

func (s *Server) createAuthSession(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var request authSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", fmt.Errorf("decode auth session request: %w", err).Error(), apiErrorOptions{})
		return
	}

	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "api key is required", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "api_key", Message: "api key is required"}},
		})
		return
	}

	if request.Mode != "tenant" && request.Mode != "service-account" {
		writeAPIError(w, r, http.StatusBadRequest, "invalid_request", "mode must be tenant or service-account", apiErrorOptions{
			FieldErrors: []apiFieldError{{Path: "mode", Message: "mode must be tenant or service-account"}},
		})
		return
	}

	tenants, err := s.store.ListTenants(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	var principal AuthenticatedPrincipal
	var ok bool
	switch request.Mode {
	case "tenant":
		principal, ok = findTenantPrincipalByAPIKey(tenants, apiKey)
	case "service-account":
		principal, ok = findServiceAccountPrincipalByAPIKey(tenants, apiKey)
	}
	if !ok {
		writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid tenant credentials", apiErrorOptions{})
		return
	}

	record := s.authSessions.Create(request.Mode, apiKey, request.Remember)
	setAuthSessionCookie(w, r, record)
	writeJSON(w, http.StatusCreated, authSessionResponse{
		SessionID: record.PublicID,
		Mode:      request.Mode,
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
			"auth_mode":  request.Mode,
			"persistent": fmt.Sprintf("%t", request.Remember),
		},
	})
}

func (s *Server) deleteAuthSession(w http.ResponseWriter, r *http.Request) {
	if secret := readAuthSessionSecret(r); secret != "" {
		s.authSessions.Delete(secret)
	}
	clearAuthSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}
