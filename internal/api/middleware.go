package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

const (
	tenantAPIKeyHeader = "X-API-Key"
	adminAPIKeyHeader  = "X-Admin-Key"
)

type tenantContextKey struct{}

// TenantAuthMiddleware authenticates tenant API keys and injects tenant context.
func TenantAuthMiddleware(stateStore store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stateStore == nil {
			http.Error(w, "tenant store is not configured", http.StatusInternalServerError)
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get(tenantAPIKeyHeader))
		tenants, err := stateStore.ListTenants(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if apiKey == "" {
			if tenant, ok := defaultTenantFallback(tenants); ok {
				ctx := context.WithValue(store.ContextWithTenantID(r.Context(), tenant.ID), tenantContextKey{}, tenant)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, "missing tenant API key", http.StatusUnauthorized)
			return
		}

		for _, tenant := range tenants {
			if tenant.Active && tenant.APIKey == apiKey {
				ctx := context.WithValue(store.ContextWithTenantID(r.Context(), tenant.ID), tenantContextKey{}, tenant)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		http.Error(w, "invalid tenant API key", http.StatusUnauthorized)
	})
}

// RequireTenant returns the authenticated tenant from a request context.
func RequireTenant(ctx context.Context) (*models.Tenant, error) {
	tenant, ok := ctx.Value(tenantContextKey{}).(models.Tenant)
	if !ok {
		return nil, fmt.Errorf("authenticated tenant is missing from context")
	}

	cloned := tenant
	return &cloned, nil
}

// AdminAuthMiddleware authenticates administrative API requests.
func AdminAuthMiddleware(adminAPIKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(adminAPIKey) == "" {
			http.Error(w, "admin API key is not configured", http.StatusServiceUnavailable)
			return
		}
		if r.Header.Get(adminAPIKeyHeader) != adminAPIKey {
			http.Error(w, "invalid admin API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
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
