package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

// CredentialLookupResult describes the tenant-scoped owner of a persisted API-key hash.
type CredentialLookupResult struct {
	// Tenant owns the matched credential.
	Tenant models.Tenant
	// ServiceAccount is populated when the matched hash belongs to a tenant service account.
	ServiceAccount *models.ServiceAccount
}

// CredentialLookupStore exposes hashed credential lookups backed by the store's credential index.
type CredentialLookupStore interface {
	// LookupCredentialByHash resolves a persisted tenant or service-account credential hash to its active owner.
	LookupCredentialByHash(ctx context.Context, credentialHash string) (*CredentialLookupResult, error)
}

// IsNotFound reports whether err describes a missing persisted record.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// LookupCredentialByHash resolves a persisted credential hash against the in-memory tenant registry.
func (s *MemoryStore) LookupCredentialByHash(ctx context.Context, credentialHash string) (*CredentialLookupResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("memory store: lookup credential by hash: %w", ctx.Err())
	default:
	}

	credentialHash = strings.TrimSpace(credentialHash)
	if credentialHash == "" {
		return nil, nil
	}

	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, tenant := range s.tenants {
		if !tenant.Active {
			continue
		}
		if credentialHashMatches(tenant.APIKeyHash, credentialHash) {
			return &CredentialLookupResult{Tenant: cloneTenant(tenant)}, nil
		}
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if !serviceAccountCredentialActive(account, now) || !credentialHashMatches(account.APIKeyHash, credentialHash) {
				continue
			}
			tenantClone := cloneTenant(tenant)
			for cloneIndex := range tenantClone.ServiceAccounts {
				if tenantClone.ServiceAccounts[cloneIndex].ID == account.ID {
					return &CredentialLookupResult{
						Tenant:         tenantClone,
						ServiceAccount: &tenantClone.ServiceAccounts[cloneIndex],
					}, nil
				}
			}
			return nil, nil
		}
	}

	return nil, nil
}

// LookupCredentialByHash resolves a persisted credential hash through PostgreSQL's credential registry.
func (s *PostgresStore) LookupCredentialByHash(ctx context.Context, credentialHash string) (*CredentialLookupResult, error) {
	ctx, cancel := s.readContext(ctx)
	defer cancel()

	credentialHash = strings.TrimSpace(credentialHash)
	if credentialHash == "" {
		return nil, nil
	}

	var (
		tenantID         string
		ownerType        string
		serviceAccountID string
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT tenant_id, owner_type, service_account_id
		   FROM credential_hashes
		  WHERE credential_hash = $1
		  LIMIT 1`,
		credentialHash,
	).Scan(&tenantID, &ownerType, &serviceAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres store: lookup credential by hash: %w", err)
	}

	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres store: lookup credential tenant %s: %w", tenantID, err)
	}
	if tenant == nil || !tenant.Active {
		return nil, nil
	}

	switch strings.TrimSpace(ownerType) {
	case "tenant":
		if !credentialHashMatches(tenant.APIKeyHash, credentialHash) {
			return nil, nil
		}
		return &CredentialLookupResult{Tenant: cloneTenant(*tenant)}, nil
	case "service_account":
		now := time.Now().UTC()
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if account.ID != strings.TrimSpace(serviceAccountID) || !serviceAccountCredentialActive(account, now) {
				continue
			}
			if !credentialHashMatches(account.APIKeyHash, credentialHash) {
				return nil, nil
			}
			tenantClone := cloneTenant(*tenant)
			for cloneIndex := range tenantClone.ServiceAccounts {
				if tenantClone.ServiceAccounts[cloneIndex].ID == account.ID {
					return &CredentialLookupResult{
						Tenant:         tenantClone,
						ServiceAccount: &tenantClone.ServiceAccounts[cloneIndex],
					}, nil
				}
			}
			return nil, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("postgres store: lookup credential by hash: unknown owner type %q", ownerType)
	}
}

func credentialHashMatches(storedHash, expectedHash string) bool {
	storedHash = strings.TrimSpace(storedHash)
	expectedHash = strings.TrimSpace(expectedHash)
	if storedHash == "" || expectedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(expectedHash)) == 1
}

func serviceAccountCredentialActive(account models.ServiceAccount, now time.Time) bool {
	if !account.Active {
		return false
	}
	if !account.ExpiresAt.IsZero() && now.After(account.ExpiresAt) {
		return false
	}
	return true
}
