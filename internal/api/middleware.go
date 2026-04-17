package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

const (
	tenantAPIKeyHeader         = "X-API-Key"
	serviceAccountAPIKeyHeader = "X-Service-Account-Key"
	adminAPIKeyHeader          = "X-Admin-Key"
)

type tenantContextKey struct{}
type principalContextKey struct{}

// AuthenticatedPrincipal captures the tenant-scoped caller identity attached to a request.
type AuthenticatedPrincipal struct {
	// Tenant is the tenant the caller has been authenticated into.
	Tenant models.Tenant
	// Role is the effective tenant-scoped authorization level.
	Role models.TenantRole
	// ServiceAccount is populated when authentication used a scoped machine credential.
	ServiceAccount *models.ServiceAccount
	// AuthMethod identifies how the request authenticated.
	AuthMethod string
}

// TenantAuthMiddleware authenticates tenant API keys and injects tenant context.
func TenantAuthMiddleware(stateStore store.Store, next http.Handler) http.Handler {
	return tenantAuthMiddleware(stateStore, nil, false, next)
}

func (s *Server) tenantAuthMiddleware(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return tenantAuthMiddleware(s.store, s.authSessions, s.allowAnonymousAdmin, next)
}

func tenantAuthMiddleware(stateStore store.Store, sessions *authSessionManager, allowAnonymousAdmin bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stateStore == nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "tenant store is not configured", apiErrorOptions{Retryable: true})
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get(tenantAPIKeyHeader))
		serviceAccountKey := strings.TrimSpace(r.Header.Get(serviceAccountAPIKeyHeader))
		authMethod := ""
		if apiKey == "" && serviceAccountKey == "" && sessions != nil {
			if sessionRecord, ok := sessions.Lookup(readAuthSessionSecret(r)); ok {
				switch sessionRecord.Mode {
				case "tenant":
					apiKey = sessionRecord.APIKey
				case "service-account":
					serviceAccountKey = sessionRecord.APIKey
				}
				authMethod = "runtime-session"
			}
		}
		tenants, err := stateStore.ListTenants(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		principal, ok := authenticateTenantPrincipal(tenants, apiKey, serviceAccountKey)
		if !ok {
			if apiKey == "" && serviceAccountKey == "" {
				if tenant, fallbackOK := defaultTenantFallback(tenants); fallbackOK {
					role := models.TenantRoleViewer
					if allowAnonymousAdmin {
						role = models.TenantRoleAdmin
					}
					packageLogger.Warn(
						"AUDIT default tenant fallback granted",
						"request_id", responseRequestID(nil, r),
						"tenant_id", tenant.ID,
						"role", role,
						"allow_anonymous_admin", allowAnonymousAdmin,
					)
					principal = AuthenticatedPrincipal{
						Tenant:     tenant,
						Role:       role,
						AuthMethod: "default-fallback",
					}
					ctx := withPrincipal(r.Context(), principal)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				writeAPIError(w, r, http.StatusUnauthorized, "missing_credentials", "missing tenant API key", apiErrorOptions{})
				return
			}

			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid tenant credentials", apiErrorOptions{})
			return
		}
		if authMethod != "" {
			principal.AuthMethod = authMethod
		}
		credentialTenantIDs := lookupCredentialTenantIDs(tenants, apiKey, serviceAccountKey)
		if mismatch := tenantAuthMismatch(store.TenantIDFromContext(r.Context()), principal.Tenant.ID, credentialTenantIDs); mismatch {
			packageLogger.Warn(
				"tenant auth mismatch rejected",
				"request_id", responseRequestID(nil, r),
				"context_tenant_id", strings.TrimSpace(store.TenantIDFromContext(r.Context())),
				"principal_tenant_id", principal.Tenant.ID,
				"credential_tenant_ids", keysFromSet(credentialTenantIDs),
				"auth_method", firstNonEmpty(principal.AuthMethod, authMethod),
			)
			writeAPIError(w, r, http.StatusForbidden, "tenant_mismatch", "tenant credentials do not match the active tenant context", apiErrorOptions{
				Details: map[string]any{
					"context_tenant_id":     strings.TrimSpace(store.TenantIDFromContext(r.Context())),
					"principal_tenant_id":   principal.Tenant.ID,
					"credential_tenant_ids": keysFromSet(credentialTenantIDs),
				},
			})
			return
		}

		ctx := withPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireTenant returns the authenticated tenant from a request context.
func RequireTenant(ctx context.Context) (*models.Tenant, error) {
	if principal, err := RequirePrincipal(ctx); err == nil {
		cloned := cloneTenant(principal.Tenant)
		return &cloned, nil
	}

	tenant, ok := ctx.Value(tenantContextKey{}).(models.Tenant)
	if !ok {
		return nil, fmt.Errorf("authenticated tenant is missing from context")
	}

	cloned := tenant
	return &cloned, nil
}

// RequirePrincipal returns the authenticated tenant principal from a request context.
func RequirePrincipal(ctx context.Context) (*AuthenticatedPrincipal, error) {
	principal, ok := ctx.Value(principalContextKey{}).(AuthenticatedPrincipal)
	if !ok {
		return nil, fmt.Errorf("authenticated principal is missing from context")
	}

	cloned := principal
	cloned.Tenant = cloneTenant(principal.Tenant)
	if principal.ServiceAccount != nil {
		serviceAccount := *principal.ServiceAccount
		serviceAccount.Metadata = copyStringMap(serviceAccount.Metadata)
		cloned.ServiceAccount = &serviceAccount
	}
	return &cloned, nil
}

// RequireTenantRole enforces a minimum tenant-scoped role for a request.
func RequireTenantRole(required models.TenantRole, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := RequirePrincipal(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", err.Error(), apiErrorOptions{})
			return
		}
		if !principal.Role.Allows(required) {
			writeAPIError(w, r, http.StatusForbidden, "permission_denied", fmt.Sprintf("tenant role %q cannot access this route", principal.Role), apiErrorOptions{
				Details: map[string]any{
					"required_role": required,
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireTenantPermission enforces a tenant-scoped capability for a request.
func RequireTenantPermission(required models.TenantPermission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := RequirePrincipal(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", err.Error(), apiErrorOptions{})
			return
		}
		if !principalAllowsPermission(*principal, required) {
			writeAPIError(w, r, http.StatusForbidden, "permission_denied", fmt.Sprintf("tenant principal cannot access %q", required), apiErrorOptions{
				Details: map[string]any{
					"required_permission": required,
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAnyTenantPermission enforces that the authenticated principal has at least one of the supplied permissions.
func RequireAnyTenantPermission(next http.Handler, permissions ...models.TenantPermission) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := RequirePrincipal(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", err.Error(), apiErrorOptions{})
			return
		}
		for _, permission := range permissions {
			if principalAllowsPermission(*principal, permission) {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeAPIError(w, r, http.StatusForbidden, "permission_denied", "tenant principal does not include any required workspace permissions", apiErrorOptions{
			Details: map[string]any{
				"required_permissions": permissions,
			},
		})
	})
}

// AdminAuthMiddleware authenticates administrative API requests.
func AdminAuthMiddleware(adminAPIKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(adminAPIKey) == "" {
			writeAPIError(w, r, http.StatusServiceUnavailable, "internal_error", "admin API key is not configured", apiErrorOptions{Retryable: true})
			return
		}
		if r.Header.Get(adminAPIKeyHeader) != adminAPIKey {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid admin API key", apiErrorOptions{})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authenticateTenantPrincipal(tenants []models.Tenant, tenantAPIKey, serviceAccountKey string) (AuthenticatedPrincipal, bool) {
	if principal, ok := findTenantPrincipalByAPIKey(tenants, tenantAPIKey); ok {
		return principal, true
	}
	if principal, ok := findServiceAccountPrincipalByAPIKey(tenants, serviceAccountKey); ok {
		return principal, true
	}
	if serviceAccountKey == "" {
		if principal, ok := findServiceAccountPrincipalByAPIKey(tenants, tenantAPIKey); ok {
			return principal, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func lookupCredentialTenantIDs(tenants []models.Tenant, tenantAPIKey, serviceAccountKey string) map[string]struct{} {
	tenantIDs := make(map[string]struct{})
	addMatches := func(apiKey string) {
		if principal, ok := findTenantPrincipalByAPIKey(tenants, apiKey); ok {
			tenantIDs[principal.Tenant.ID] = struct{}{}
		}
		if principal, ok := findServiceAccountPrincipalByAPIKey(tenants, apiKey); ok {
			tenantIDs[principal.Tenant.ID] = struct{}{}
		}
	}
	addMatches(tenantAPIKey)
	addMatches(serviceAccountKey)
	return tenantIDs
}

func tenantAuthMismatch(contextTenantID, principalTenantID string, credentialTenantIDs map[string]struct{}) bool {
	contextTenantID = strings.TrimSpace(contextTenantID)
	principalTenantID = strings.TrimSpace(principalTenantID)
	if contextTenantID == store.DefaultTenantID && principalTenantID != "" && principalTenantID != store.DefaultTenantID {
		contextTenantID = ""
	}

	if contextTenantID != "" && principalTenantID != "" && contextTenantID != principalTenantID {
		return true
	}
	for tenantID := range credentialTenantIDs {
		if principalTenantID != "" && tenantID != "" && tenantID != principalTenantID {
			return true
		}
		if contextTenantID != "" && tenantID != "" && tenantID != contextTenantID {
			return true
		}
	}
	return false
}

func defaultTenantFallback(tenants []models.Tenant) (models.Tenant, bool) {
	if len(tenants) == 0 {
		return models.Tenant{}, false
	}

	activeCustomTenants := 0
	var defaultTenant models.Tenant
	for _, tenant := range tenants {
		if !tenant.Active {
			continue
		}
		if tenant.ID == store.DefaultTenantID {
			defaultTenant = tenant
			continue
		}
		activeCustomTenants++
	}

	if activeCustomTenants == 0 && defaultTenant.ID == store.DefaultTenantID && defaultTenant.APIKey == "" {
		return defaultTenant, true
	}
	return models.Tenant{}, false
}

func findTenantPrincipalByAPIKey(tenants []models.Tenant, apiKey string) (AuthenticatedPrincipal, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return AuthenticatedPrincipal{}, false
	}

	for _, tenant := range tenants {
		if tenant.Active && tenant.APIKey == apiKey {
			return AuthenticatedPrincipal{
				Tenant:     tenant,
				Role:       models.TenantRoleAdmin,
				AuthMethod: "tenant-api-key",
			}, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func findServiceAccountPrincipalByAPIKey(tenants []models.Tenant, apiKey string) (AuthenticatedPrincipal, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return AuthenticatedPrincipal{}, false
	}

	now := time.Now().UTC()
	for _, tenant := range tenants {
		if !tenant.Active {
			continue
		}
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if account.APIKey != apiKey || !serviceAccountUsable(account, now) {
				continue
			}
			role := account.Role
			if role == "" {
				role = models.TenantRoleViewer
			}
			return AuthenticatedPrincipal{
				Tenant:         tenant,
				Role:           role,
				ServiceAccount: &account,
				AuthMethod:     "service-account",
			}, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func serviceAccountUsable(account models.ServiceAccount, now time.Time) bool {
	if !account.Active {
		return false
	}
	if !account.ExpiresAt.IsZero() && now.After(account.ExpiresAt) {
		return false
	}
	return true
}

func withPrincipal(ctx context.Context, principal AuthenticatedPrincipal) context.Context {
	tenant := cloneTenant(principal.Tenant)
	if scope := requestScopeFromContext(ctx); scope != nil {
		scope.tenantID = tenant.ID
		scope.authMethod = principal.AuthMethod
	}
	ctx = store.ContextWithTenantID(ctx, tenant.ID)
	ctx = context.WithValue(ctx, tenantContextKey{}, tenant)
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func cloneTenant(tenant models.Tenant) models.Tenant {
	tenant.Settings = copyStringMap(tenant.Settings)
	tenant.ServiceAccounts = cloneServiceAccounts(tenant.ServiceAccounts)
	return tenant
}

func cloneServiceAccounts(accounts []models.ServiceAccount) []models.ServiceAccount {
	if len(accounts) == 0 {
		return nil
	}

	cloned := make([]models.ServiceAccount, 0, len(accounts))
	for _, account := range accounts {
		item := account
		item.Metadata = copyStringMap(account.Metadata)
		if len(account.Permissions) > 0 {
			item.Permissions = append([]models.TenantPermission(nil), account.Permissions...)
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func principalAllowsPermission(principal AuthenticatedPrincipal, permission models.TenantPermission) bool {
	if !permission.Valid() {
		return false
	}
	if principal.ServiceAccount != nil {
		return principal.ServiceAccount.Allows(permission)
	}
	for _, granted := range principal.Role.DefaultPermissions() {
		if granted == permission {
			return true
		}
	}
	return false
}

func copyStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
