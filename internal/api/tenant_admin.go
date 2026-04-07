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
	"github.com/google/uuid"
)

type serviceAccountCreateRequest struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	APIKey      string                    `json:"api_key,omitempty"`
	Role        models.TenantRole         `json:"role,omitempty"`
	Permissions []models.TenantPermission `json:"permissions,omitempty"`
	Active      *bool                     `json:"active,omitempty"`
	ExpiresAt   time.Time                 `json:"expires_at,omitempty"`
	Metadata    map[string]string         `json:"metadata,omitempty"`
}

type serviceAccountRotateRequest struct {
	APIKey string `json:"api_key,omitempty"`
}

type currentTenantResponse struct {
	TenantID            string                    `json:"tenant_id"`
	Name                string                    `json:"name"`
	Active              bool                      `json:"active"`
	Settings            map[string]string         `json:"settings,omitempty"`
	Quotas              models.TenantQuota        `json:"quotas,omitempty"`
	Role                models.TenantRole         `json:"role"`
	Permissions         []models.TenantPermission `json:"permissions,omitempty"`
	AuthMethod          string                    `json:"auth_method"`
	ServiceAccountID    string                    `json:"service_account_id,omitempty"`
	ServiceAccountName  string                    `json:"service_account_name,omitempty"`
	ServiceAccountCount int                       `json:"service_account_count"`
}

type serviceAccountResponse struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description,omitempty"`
	APIKey        string                    `json:"api_key,omitempty"`
	Role          models.TenantRole         `json:"role"`
	Permissions   []models.TenantPermission `json:"permissions,omitempty"`
	Active        bool                      `json:"active"`
	CreatedAt     time.Time                 `json:"created_at"`
	LastRotatedAt time.Time                 `json:"last_rotated_at,omitempty"`
	ExpiresAt     time.Time                 `json:"expires_at,omitempty"`
	Metadata      map[string]string         `json:"metadata,omitempty"`
}

type adminTenantResponse struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	APIKey          string                   `json:"api_key,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	Active          bool                     `json:"active"`
	Settings        map[string]string        `json:"settings,omitempty"`
	Quotas          models.TenantQuota       `json:"quotas,omitempty"`
	ServiceAccounts []serviceAccountResponse `json:"service_accounts,omitempty"`
}

func (s *Server) handleCurrentTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, err := RequirePrincipal(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	response := currentTenantResponse{
		TenantID:            principal.Tenant.ID,
		Name:                principal.Tenant.Name,
		Active:              principal.Tenant.Active,
		Settings:            copyStringMap(principal.Tenant.Settings),
		Quotas:              principal.Tenant.Quotas,
		Role:                principal.Role,
		AuthMethod:          principal.AuthMethod,
		ServiceAccountCount: len(principal.Tenant.ServiceAccounts),
	}
	if principal.ServiceAccount != nil {
		response.ServiceAccountID = principal.ServiceAccount.ID
		response.ServiceAccountName = principal.ServiceAccount.Name
		response.Permissions = append([]models.TenantPermission(nil), principal.ServiceAccount.EffectivePermissions()...)
	} else {
		response.Permissions = append([]models.TenantPermission(nil), principal.Role.DefaultPermissions()...)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleServiceAccounts(w http.ResponseWriter, r *http.Request) {
	principal, err := RequirePrincipal(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		accounts := make([]serviceAccountResponse, 0, len(principal.Tenant.ServiceAccounts))
		for _, account := range principal.Tenant.ServiceAccounts {
			accounts = append(accounts, toServiceAccountResponse(account, false))
		}
		writeJSON(w, http.StatusOK, accounts)
	case http.MethodPost:
		defer r.Body.Close()

		var request serviceAccountCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Errorf("decode service account: %w", err).Error(), http.StatusBadRequest)
			return
		}

		tenant, err := s.store.GetTenant(r.Context(), principal.Tenant.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		account, err := newServiceAccountFromRequest(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := ensureUniqueServiceAccount(*tenant, account); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		tenant.ServiceAccounts = append(tenant.ServiceAccounts, account)
		if err := s.store.UpdateTenant(r.Context(), *tenant); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "tenant",
			Action:   "create-service-account",
			Resource: account.ID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "service account created",
			Details: map[string]string{
				"service_account_name": account.Name,
				"role":                 string(account.Role),
			},
		})

		writeJSON(w, http.StatusCreated, toServiceAccountResponse(account, true))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleServiceAccountByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/service-accounts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "rotate" {
		http.Error(w, "service account rotate route not found", http.StatusNotFound)
		return
	}

	principal, err := RequirePrincipal(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()
	var request serviceAccountRotateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, fmt.Errorf("decode service account rotation request: %w", err).Error(), http.StatusBadRequest)
		return
	}

	tenant, err := s.store.GetTenant(r.Context(), principal.Tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accountID := parts[0]
	for index := range tenant.ServiceAccounts {
		if tenant.ServiceAccounts[index].ID != accountID {
			continue
		}
		if strings.TrimSpace(request.APIKey) == "" {
			tenant.ServiceAccounts[index].APIKey = uuid.NewString()
		} else {
			tenant.ServiceAccounts[index].APIKey = strings.TrimSpace(request.APIKey)
		}
		tenant.ServiceAccounts[index].LastRotatedAt = time.Now().UTC()

		if err := s.store.UpdateTenant(r.Context(), *tenant); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.recordAuditEvent(r, models.AuditEvent{
			Category: "tenant",
			Action:   "rotate-service-account-key",
			Resource: accountID,
			Outcome:  models.AuditOutcomeSuccess,
			Message:  "service account key rotated",
		})

		writeJSON(w, http.StatusOK, toServiceAccountResponse(tenant.ServiceAccounts[index], true))
		return
	}

	http.Error(w, "service account not found", http.StatusNotFound)
}

func newServiceAccountFromRequest(request serviceAccountCreateRequest) (models.ServiceAccount, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return models.ServiceAccount{}, fmt.Errorf("service account name is required")
	}

	role := request.Role
	if role == "" {
		role = models.TenantRoleViewer
	}
	if !validTenantRole(role) {
		return models.ServiceAccount{}, fmt.Errorf("service account role %q is invalid", role)
	}
	permissions := make([]models.TenantPermission, 0, len(request.Permissions))
	for _, permission := range request.Permissions {
		if !permission.Valid() {
			return models.ServiceAccount{}, fmt.Errorf("service account permission %q is invalid", permission)
		}
		permissions = append(permissions, permission)
	}

	active := true
	if request.Active != nil {
		active = *request.Active
	}

	apiKey := strings.TrimSpace(request.APIKey)
	if apiKey == "" {
		apiKey = uuid.NewString()
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		id = uuid.NewString()
	}

	return models.ServiceAccount{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(request.Description),
		APIKey:      apiKey,
		Role:        role,
		Permissions: permissions,
		Active:      active,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   request.ExpiresAt,
		Metadata:    copyStringMap(request.Metadata),
	}, nil
}

func ensureUniqueServiceAccount(tenant models.Tenant, account models.ServiceAccount) error {
	for _, existing := range tenant.ServiceAccounts {
		if existing.ID == account.ID {
			return fmt.Errorf("service account %s already exists", account.ID)
		}
		if strings.EqualFold(existing.Name, account.Name) {
			return fmt.Errorf("service account name %q already exists", account.Name)
		}
	}
	return nil
}

func toServiceAccountResponse(account models.ServiceAccount, revealKey bool) serviceAccountResponse {
	response := serviceAccountResponse{
		ID:            account.ID,
		Name:          account.Name,
		Description:   account.Description,
		Role:          account.Role,
		Permissions:   append([]models.TenantPermission(nil), account.EffectivePermissions()...),
		Active:        account.Active,
		CreatedAt:     account.CreatedAt,
		LastRotatedAt: account.LastRotatedAt,
		ExpiresAt:     account.ExpiresAt,
		Metadata:      copyStringMap(account.Metadata),
	}
	if revealKey {
		response.APIKey = account.APIKey
	}
	return response
}

func toAdminTenantResponse(tenant models.Tenant, revealTenantKey bool) adminTenantResponse {
	response := adminTenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		CreatedAt: tenant.CreatedAt,
		Active:    tenant.Active,
		Settings:  copyStringMap(tenant.Settings),
		Quotas:    tenant.Quotas,
	}
	if revealTenantKey {
		response.APIKey = tenant.APIKey
	}
	if len(tenant.ServiceAccounts) > 0 {
		response.ServiceAccounts = make([]serviceAccountResponse, 0, len(tenant.ServiceAccounts))
		for _, account := range tenant.ServiceAccounts {
			response.ServiceAccounts = append(response.ServiceAccounts, toServiceAccountResponse(account, false))
		}
	}
	return response
}

func validTenantRole(role models.TenantRole) bool {
	switch role {
	case models.TenantRoleViewer, models.TenantRoleOperator, models.TenantRoleAdmin:
		return true
	default:
		return false
	}
}
