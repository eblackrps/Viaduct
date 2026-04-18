package store

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

const apiKeyHashPrefix = "sha256:"

// CredentialConflictError reports that a tenant or service-account credential collides with another persisted credential.
type CredentialConflictError struct {
	TenantID         string
	ServiceAccountID string
}

// Error returns a human-readable description of the conflicting credential owner.
func (e *CredentialConflictError) Error() string {
	if e == nil {
		return "credential already exists"
	}
	if strings.TrimSpace(e.ServiceAccountID) != "" {
		return fmt.Sprintf("credential already exists on service account %s in tenant %s", e.ServiceAccountID, e.TenantID)
	}
	if strings.TrimSpace(e.TenantID) != "" {
		return fmt.Sprintf("credential already exists on tenant %s", e.TenantID)
	}
	return "credential already exists"
}

// IsCredentialConflict reports whether err describes a persisted credential collision.
func IsCredentialConflict(err error) bool {
	var target *CredentialConflictError
	return errors.As(err, &target)
}

// HashAPIKey returns the canonical non-recoverable digest used for persisted tenant and service-account credentials.
func HashAPIKey(apiKey string) string {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(trimmed))
	return apiKeyHashPrefix + hex.EncodeToString(sum[:])
}

// APIKeyMatches reports whether apiKey matches either a persisted credential digest or a legacy raw value.
func APIKeyMatches(apiKey, storedHash, legacyRaw string) bool {
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return false
	}

	storedHash = strings.TrimSpace(storedHash)
	if storedHash != "" {
		candidate := HashAPIKey(trimmed)
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(storedHash)) == 1
	}

	legacyRaw = strings.TrimSpace(legacyRaw)
	if legacyRaw == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(trimmed), []byte(legacyRaw)) == 1
}

// HasAPIKeyConfigured reports whether a credential record contains either a persisted digest or a legacy raw value.
func HasAPIKeyConfigured(storedHash, legacyRaw string) bool {
	return strings.TrimSpace(storedHash) != "" || strings.TrimSpace(legacyRaw) != ""
}

func prepareTenantCredentials(tenant models.Tenant) (models.Tenant, error) {
	if len(tenant.ServiceAccounts) > 0 {
		tenant.ServiceAccounts = append([]models.ServiceAccount(nil), tenant.ServiceAccounts...)
	}
	tenant.APIKeyHash = normalizedAPIKeyHash(tenant.APIKey, tenant.APIKeyHash)
	tenant.APIKey = ""

	for index := range tenant.ServiceAccounts {
		hash := normalizedAPIKeyHash(tenant.ServiceAccounts[index].APIKey, tenant.ServiceAccounts[index].APIKeyHash)
		tenant.ServiceAccounts[index].APIKeyHash = hash
		tenant.ServiceAccounts[index].APIKey = ""
	}

	return tenant, nil
}

func normalizedAPIKeyHash(apiKey, storedHash string) string {
	if candidate := HashAPIKey(apiKey); candidate != "" {
		return candidate
	}
	return strings.TrimSpace(storedHash)
}

type credentialOwner struct {
	tenantID         string
	serviceAccountID string
}

func (o credentialOwner) conflictError() error {
	return &CredentialConflictError{
		TenantID:         o.tenantID,
		ServiceAccountID: o.serviceAccountID,
	}
}

func tenantCredentialOwners(tenant models.Tenant) map[string]credentialOwner {
	owners := make(map[string]credentialOwner)
	if hash := strings.TrimSpace(tenant.APIKeyHash); hash != "" {
		owners[hash] = credentialOwner{tenantID: tenant.ID}
	}
	for _, account := range tenant.ServiceAccounts {
		if hash := strings.TrimSpace(account.APIKeyHash); hash != "" {
			owners[hash] = credentialOwner{
				tenantID:         tenant.ID,
				serviceAccountID: account.ID,
			}
		}
	}
	return owners
}

func validateCredentialUniqueness(tenants []models.Tenant, tenant models.Tenant) error {
	owners := tenantCredentialOwners(tenant)
	if err := validateCredentialUniquenessWithinTenant(tenant); err != nil {
		return err
	}

	for _, existingTenant := range tenants {
		if existingTenant.ID == tenant.ID {
			continue
		}
		for hash, owner := range tenantCredentialOwners(existingTenant) {
			if _, ok := owners[hash]; ok {
				return owner.conflictError()
			}
		}
	}
	return nil
}

func validateCredentialUniquenessWithinTenant(tenant models.Tenant) error {
	seen := make(map[string]credentialOwner)
	if hash := strings.TrimSpace(tenant.APIKeyHash); hash != "" {
		owner := credentialOwner{tenantID: tenant.ID}
		seen[hash] = owner
	}
	for _, account := range tenant.ServiceAccounts {
		hash := strings.TrimSpace(account.APIKeyHash)
		if hash == "" {
			continue
		}
		if existing, ok := seen[hash]; ok {
			return existing.conflictError()
		}
		seen[hash] = credentialOwner{
			tenantID:         tenant.ID,
			serviceAccountID: account.ID,
		}
	}
	return nil
}
