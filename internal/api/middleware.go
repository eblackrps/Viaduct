package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

const (
	tenantCredentialHeader         = "X-API-Key"             // #nosec G101 -- this is an HTTP header name, not an embedded credential.
	serviceAccountCredentialHeader = "X-Service-Account-Key" // #nosec G101 -- this is an HTTP header name, not an embedded credential.
	adminCredentialHeader          = "X-Admin-Key"
	credentialHashPrefix           = "sha256:"
)

type tenantContextKey struct{}
type principalContextKey struct{}

var zeroCredentialHash [sha256.Size]byte
var invalidCredentialHash = [sha256.Size]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}
var credentialHashHexDecode = newCredentialHashHexDecodeTable()

// API authentication always hashes presented and persisted credentials into fixed-size
// SHA-256 digests before comparing them so every constant-time comparison operates on
// equal-length 32-byte inputs, including legacy plaintext records that are normalized
// into digests at comparison time.

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
	return tenantAuthMiddleware(stateStore, nil, nil, next)
}

func (s *Server) tenantAuthMiddleware(next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return tenantAuthMiddleware(s.store, s.authSessions, s, next)
}

func tenantAuthMiddleware(stateStore store.Store, sessions *authSessionManager, server *Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stateStore == nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "tenant store is not configured", apiErrorOptions{Retryable: true})
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get(tenantCredentialHeader))
		serviceAccountKey := strings.TrimSpace(r.Header.Get(serviceAccountCredentialHeader))
		authMethod := ""
		tenants, err := stateStore.ListTenants(r.Context())
		if err != nil {
			writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
			return
		}

		var sessionPrincipal *AuthenticatedPrincipal
		staleCredentialSession := false
		if apiKey == "" && serviceAccountKey == "" && sessions != nil {
			sessionSecret := readAuthSessionSecret(r)
			if sessionRecord, ok, lookupErr := sessions.LookupActive(r.Context(), stateStore, sessionSecret); lookupErr != nil {
				writeAPIError(w, r, http.StatusInternalServerError, "internal_error", lookupErr.Error(), apiErrorOptions{Retryable: true})
				return
			} else if ok {
				switch sessionRecord.Mode {
				case "tenant":
					if principal, tenantFound := principalFromTenantSession(r.Context(), tenants, sessionRecord); tenantFound {
						sessionPrincipal = &principal
						authMethod = "runtime-session"
					} else if server != nil && !isZeroCredentialHash(sessionRecord.CredentialHash) {
						staleCredentialSession = true
						currentHash, _ := currentCredentialHashForSession(r.Context(), tenants, sessionRecord)
						server.revokeCredentialBoundSession(r, sessionRecord, sessionSecret, "credential_rotated", currentHash)
					}
				case "service-account":
					if principal, accountFound := principalFromServiceAccountSession(r.Context(), tenants, sessionRecord); accountFound {
						sessionPrincipal = &principal
						authMethod = "runtime-session"
					} else if server != nil && !isZeroCredentialHash(sessionRecord.CredentialHash) {
						staleCredentialSession = true
						currentHash, _ := currentCredentialHashForSession(r.Context(), tenants, sessionRecord)
						server.revokeCredentialBoundSession(r, sessionRecord, sessionSecret, "credential_rotated", currentHash)
					}
				case "local":
					authMethod = firstNonEmpty(sessionRecord.AuthMethod, "local-runtime-session")
					if tenant, tenantFound := findTenantByID(tenants, sessionRecord.TenantID); tenantFound {
						principal := AuthenticatedPrincipal{
							Tenant: tenant,
							Role:   sessionRecord.Role,
						}
						sessionPrincipal = &principal
					}
				}
			}
		}

		var (
			principal AuthenticatedPrincipal
			ok        bool
		)
		if sessionPrincipal != nil {
			if tenant, tenantFound := findTenantByID(tenants, sessionPrincipal.Tenant.ID); tenantFound {
				principal = *sessionPrincipal
				principal.Tenant = tenant
				if authMethod != "" {
					principal.AuthMethod = authMethod
				}
				ok = true
			}
		}
		if !ok {
			principal, ok = authenticateTenantPrincipal(r.Context(), tenants, apiKey, serviceAccountKey)
		}
		if !ok {
			if staleCredentialSession {
				writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "dashboard auth session no longer matches the current tenant credential", apiErrorOptions{})
				return
			}
			if apiKey == "" && serviceAccountKey == "" {
				writeAPIError(w, r, http.StatusUnauthorized, "missing_credentials", "missing tenant API key", apiErrorOptions{})
				return
			}

			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid tenant credentials", apiErrorOptions{})
			return
		}
		if authMethod != "" {
			principal.AuthMethod = authMethod
		}
		credentialTenantIDs := lookupCredentialTenantIDs(r.Context(), tenants, apiKey, serviceAccountKey)
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

func (s *Server) revokeCredentialBoundSession(r *http.Request, record authSessionRecord, secret, reason string, currentHash [32]byte) {
	if s == nil || s.authSessions == nil || s.store == nil {
		return
	}
	if err := s.authSessions.Revoke(r.Context(), s.store, record, secret); err != nil {
		packageLogger.Warn(
			"failed to revoke stale dashboard auth session",
			"request_id", responseRequestID(nil, r),
			"session_id", record.PublicID,
			"reason", reason,
			"error", err.Error(),
		)
	}
	s.recordAuditEvent(r.WithContext(withAwaitAudit(r.Context())), models.AuditEvent{
		Category: "tenant",
		Action:   "revoke-auth-session",
		Resource: record.PublicID,
		Outcome:  models.AuditOutcomeSuccess,
		Message:  "dashboard auth session revoked after credential mismatch",
		Details: map[string]string{
			"session_id":      record.PublicID,
			"auth_mode":       record.Mode,
			"reason":          reason,
			"old_hash_prefix": credentialHashPrefix8(record.CredentialHash),
			"new_hash_prefix": credentialHashPrefix8(currentHash),
		},
	})
}

func bindHostIsLoopbackOnly(host string) bool {
	host = strings.TrimSpace(host)
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::":
		return false
	case "localhost":
		return true
	}
	parsed := net.ParseIP(strings.Trim(host, "[]"))
	return parsed != nil && parsed.IsLoopback()
}

func localRuntimeRequestAllowed(r *http.Request, bindHost string) bool {
	reason := localRuntimeRequestRejectionReason(r, bindHost)
	if reason == "" {
		return true
	}
	logLoopbackRejection(r, reason)
	return false
}

func principalFromTenantSession(ctx context.Context, tenants []models.Tenant, record authSessionRecord) (AuthenticatedPrincipal, bool) {
	tenant, ok := findTenantByID(tenants, record.TenantID)
	if !ok {
		return AuthenticatedPrincipal{}, false
	}
	if !storedCredentialHashMatches(ctx, tenant.APIKeyHash, tenant.APIKey, record.CredentialHash) {
		return AuthenticatedPrincipal{}, false
	}
	return AuthenticatedPrincipal{
		Tenant:     tenant,
		Role:       models.TenantRoleAdmin,
		AuthMethod: "runtime-session",
	}, true
}

func principalFromServiceAccountSession(ctx context.Context, tenants []models.Tenant, record authSessionRecord) (AuthenticatedPrincipal, bool) {
	now := time.Now().UTC()
	for _, tenant := range tenants {
		if !tenant.Active || tenant.ID != record.TenantID {
			continue
		}
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if account.ID != record.ServiceAccountID || !serviceAccountUsable(account, now) {
				continue
			}
			if !storedCredentialHashMatches(ctx, account.APIKeyHash, account.APIKey, record.CredentialHash) {
				return AuthenticatedPrincipal{}, false
			}
			role := account.Role
			if role == "" {
				role = models.TenantRoleViewer
			}
			return AuthenticatedPrincipal{
				Tenant:         tenant,
				Role:           role,
				ServiceAccount: &account,
				AuthMethod:     "runtime-session",
			}, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func storedCredentialHashMatches(ctx context.Context, storedHash, legacyRaw string, expectedHash [32]byte) bool {
	current, ok := storedCredentialHash(ctx, storedHash, legacyRaw)
	return constantTimeCredentialHashMatch(current, ok, expectedHash, true)
}

func constantTimeEqual(leftStoredHash, rightPresented string) bool {
	if strings.TrimSpace(leftStoredHash) == "" || strings.TrimSpace(rightPresented) == "" {
		return false
	}
	leftHash, leftOK := storedCredentialHash(context.Background(), leftStoredHash, "")
	rightHash := hashCredential(context.Background(), rightPresented)
	return constantTimeCredentialHashMatch(leftHash, leftOK, rightHash, !isZeroCredentialHash(rightHash))
}

// AdminAuthMiddleware authenticates administrative API requests.
func AdminAuthMiddleware(adminAPIKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(adminAPIKey) == "" {
			writeAPIError(w, r, http.StatusServiceUnavailable, "internal_error", "admin API key is not configured", apiErrorOptions{Retryable: true})
			return
		}
		if !constantTimeEqual(adminAPIKey, r.Header.Get(adminCredentialHeader)) {
			writeAPIError(w, r, http.StatusUnauthorized, "invalid_credentials", "invalid admin API key", apiErrorOptions{})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authenticateTenantPrincipal(ctx context.Context, tenants []models.Tenant, tenantAPIKey, serviceAccountKey string) (AuthenticatedPrincipal, bool) {
	if principal, ok := findTenantPrincipalByAPIKey(ctx, tenants, tenantAPIKey); ok {
		return principal, true
	}
	if principal, ok := findServiceAccountPrincipalByAPIKey(ctx, tenants, serviceAccountKey); ok {
		return principal, true
	}
	if serviceAccountKey == "" {
		if principal, ok := findServiceAccountPrincipalByAPIKey(ctx, tenants, tenantAPIKey); ok {
			return principal, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func lookupCredentialTenantIDs(ctx context.Context, tenants []models.Tenant, tenantAPIKey, serviceAccountKey string) map[string]struct{} {
	tenantIDs := make(map[string]struct{})
	addMatches := func(apiKey string) {
		if principal, ok := findTenantPrincipalByAPIKey(ctx, tenants, apiKey); ok {
			tenantIDs[principal.Tenant.ID] = struct{}{}
		}
		if principal, ok := findServiceAccountPrincipalByAPIKey(ctx, tenants, apiKey); ok {
			tenantIDs[principal.Tenant.ID] = struct{}{}
		}
	}
	addMatches(tenantAPIKey)
	addMatches(serviceAccountKey)
	return tenantIDs
}

func hashCredential(_ context.Context, key string) [32]byte {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return [32]byte{}
	}
	return sha256.Sum256([]byte(trimmed))
}

func storedCredentialHash(ctx context.Context, storedHash, legacyRaw string) ([32]byte, bool) {
	normalizedHash := strings.TrimSpace(storedHash)
	if normalizedHash != "" {
		return parseCredentialHash(normalizedHash)
	}
	return hashCredential(ctx, legacyRaw), strings.TrimSpace(legacyRaw) != ""
}

func parseCredentialHash(value string) ([32]byte, bool) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, credentialHashPrefix)
	var digest [32]byte
	valid := boolToConstantTime(trimmed != "") & boolToConstantTime(len(trimmed) == sha256.Size*2)

	var normalized [sha256.Size * 2]byte
	for index := 0; index < len(normalized); index++ {
		if index < len(trimmed) {
			normalized[index] = trimmed[index]
		}
	}

	for index := 0; index < sha256.Size; index++ {
		high := credentialHashHexDecode[normalized[index*2]]
		low := credentialHashHexDecode[normalized[index*2+1]]
		valid &= boolToConstantTime(high < 16)
		valid &= boolToConstantTime(low < 16)
		digest[index] = (high << 4) | low
	}

	invalidMask := byte((1 - subtle.ConstantTimeEq(valid, 1)) * 0xff)
	for index := range digest {
		digest[index] = (digest[index] &^ invalidMask) | (invalidCredentialHash[index] & invalidMask)
	}
	return digest, subtle.ConstantTimeEq(valid, 1) == 1
}

func isZeroCredentialHash(value [32]byte) bool {
	return subtle.ConstantTimeCompare(value[:], zeroCredentialHash[:]) == 1
}

func boolToConstantTime(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func newCredentialHashHexDecodeTable() [256]byte {
	var table [256]byte
	for index := range table {
		table[index] = 0xff
	}
	for index := byte(0); index < 10; index++ {
		table['0'+index] = index
	}
	for index := byte(0); index < 6; index++ {
		table['a'+index] = 10 + index
		table['A'+index] = 10 + index
	}
	return table
}

func normalizeCredentialHashForCompare(value [32]byte, valid bool) [32]byte {
	var normalized [32]byte
	var nonZero byte
	for _, element := range value {
		nonZero |= element
	}
	nonZeroResult := boolToConstantTime(nonZero != 0)
	validResult := boolToConstantTime(valid) & nonZeroResult
	invalidMask := byte((1 - subtle.ConstantTimeEq(validResult, 1)) * 0xff)
	for index := range normalized {
		normalized[index] = (value[index] &^ invalidMask) | (invalidCredentialHash[index] & invalidMask)
	}
	return normalized
}

func constantTimeCredentialHashMatch(current [32]byte, currentValid bool, expected [32]byte, expectedValid bool) bool {
	currentBuffer := normalizeCredentialHashForCompare(current, currentValid)
	expectedBuffer := normalizeCredentialHashForCompare(expected, expectedValid)
	return constantTimeNormalizedCredentialHashMatch(currentBuffer, expectedBuffer)
}

func constantTimeNormalizedCredentialHashMatch(currentBuffer, expectedBuffer [32]byte) bool {
	return subtle.ConstantTimeCompare(currentBuffer[:], expectedBuffer[:]) == 1
}

func credentialHashPrefix8(value [32]byte) string {
	if isZeroCredentialHash(value) {
		return ""
	}
	return fmt.Sprintf("%x", value[:4])
}

func currentCredentialHashForSession(ctx context.Context, tenants []models.Tenant, record authSessionRecord) ([32]byte, bool) {
	switch record.Mode {
	case "tenant":
		_, currentHash, ok := tenantSessionCredentialHash(ctx, tenants, record)
		return currentHash, ok
	case "service-account":
		return serviceAccountSessionCredentialHash(ctx, tenants, record)
	default:
		return [32]byte{}, false
	}
}

func tenantSessionCredentialHash(ctx context.Context, tenants []models.Tenant, record authSessionRecord) (models.Tenant, [32]byte, bool) {
	tenant, ok := findTenantByID(tenants, record.TenantID)
	if !ok {
		return models.Tenant{}, [32]byte{}, false
	}
	currentHash, hashOK := storedCredentialHash(ctx, tenant.APIKeyHash, tenant.APIKey)
	if !hashOK || isZeroCredentialHash(currentHash) {
		return models.Tenant{}, [32]byte{}, false
	}
	return tenant, currentHash, true
}

func serviceAccountSessionCredentialHash(ctx context.Context, tenants []models.Tenant, record authSessionRecord) ([32]byte, bool) {
	now := time.Now().UTC()
	for _, tenant := range tenants {
		if !tenant.Active || tenant.ID != record.TenantID {
			continue
		}
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if account.ID != record.ServiceAccountID || !serviceAccountUsable(account, now) {
				continue
			}
			currentHash, hashOK := storedCredentialHash(ctx, account.APIKeyHash, account.APIKey)
			if !hashOK || isZeroCredentialHash(currentHash) {
				return [32]byte{}, false
			}
			return currentHash, true
		}
	}
	return [32]byte{}, false
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

	if activeCustomTenants == 0 && defaultTenant.ID == store.DefaultTenantID && !store.HasAPIKeyConfigured(defaultTenant.APIKeyHash, defaultTenant.APIKey) {
		return defaultTenant, true
	}
	return models.Tenant{}, false
}

func findTenantPrincipalByAPIKey(ctx context.Context, tenants []models.Tenant, apiKey string) (AuthenticatedPrincipal, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return AuthenticatedPrincipal{}, false
	}

	digest := hashCredential(ctx, apiKey)
	for _, tenant := range tenants {
		if tenant.Active && storedCredentialHashMatches(ctx, tenant.APIKeyHash, tenant.APIKey, digest) {
			return AuthenticatedPrincipal{
				Tenant:     tenant,
				Role:       models.TenantRoleAdmin,
				AuthMethod: "tenant-api-key",
			}, true
		}
	}
	return AuthenticatedPrincipal{}, false
}

func findServiceAccountPrincipalByAPIKey(ctx context.Context, tenants []models.Tenant, apiKey string) (AuthenticatedPrincipal, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return AuthenticatedPrincipal{}, false
	}

	now := time.Now().UTC()
	digest := hashCredential(ctx, apiKey)
	for _, tenant := range tenants {
		if !tenant.Active {
			continue
		}
		for index := range tenant.ServiceAccounts {
			account := tenant.ServiceAccounts[index]
			if !storedCredentialHashMatches(ctx, account.APIKeyHash, account.APIKey, digest) || !serviceAccountUsable(account, now) {
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

func findTenantByID(tenants []models.Tenant, tenantID string) (models.Tenant, bool) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return models.Tenant{}, false
	}
	for _, tenant := range tenants {
		if tenant.Active && tenant.ID == tenantID {
			return tenant, true
		}
	}
	return models.Tenant{}, false
}

func requestFromLoopback(r *http.Request) bool {
	if r == nil {
		return false
	}
	ip, ok := remoteAddrIP(r.RemoteAddr)
	return ok && ip.IsLoopback()
}

func requestHostIsLoopback(r *http.Request) bool {
	if r == nil {
		return false
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return false
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isMutatingMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requestUsesForwardingHeaders(r *http.Request) bool {
	if r == nil {
		return false
	}
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP", "Via"} {
		if strings.TrimSpace(r.Header.Get(header)) != "" {
			return true
		}
	}
	return false
}

func localRuntimeRequestRejectionReason(r *http.Request, bindHost string) string {
	if !bindHostIsLoopbackOnly(bindHost) {
		return "bind_host_not_loopback_only"
	}
	if !requestFromLoopback(r) {
		return "peer_not_loopback"
	}
	if !requestHostIsLoopback(r) {
		return "host_not_loopback"
	}
	if header := rejectedForwardingHeader(r); header != "" {
		logForwardedHeaderRejected(r, header)
		return "forwarded_header_rejected"
	}
	if requestUsesForwardingHeaders(r) {
		return "forwarded_headers_present"
	}

	if r == nil {
		return "request_missing"
	}

	_, originPresent := r.Header["Origin"]
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if isMutatingMethod(r.Method) {
		if originPresent && origin == "" {
			return "empty_origin"
		}
		if origin == "" {
			if referer == "" {
				return "missing_origin_or_referer"
			}
			if !sameOriginRequest(r, referer) {
				return "referer_mismatch"
			}
			return ""
		}
		if !sameOriginRequest(r, origin) {
			return "origin_mismatch"
		}
		return ""
	}
	if origin != "" && !sameOriginRequest(r, origin) {
		return "origin_mismatch"
	}
	if referer != "" && !sameOriginRequest(r, referer) {
		return "referer_mismatch"
	}
	return ""
}

func loopbackRejectionPeer(r *http.Request) string {
	if r == nil {
		return ""
	}
	if ip, ok := remoteAddrIP(r.RemoteAddr); ok {
		return ip.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func logLoopbackRejection(r *http.Request, reason string) {
	if r == nil {
		return
	}
	packageLogger.Warn(
		"AUDIT",
		"event", "loopback_rejection",
		"method", strings.ToUpper(strings.TrimSpace(r.Method)),
		"path", strings.TrimSpace(r.URL.Path),
		"reason", reason,
		"peer", loopbackRejectionPeer(r),
		"forwarded_present", strings.TrimSpace(r.Header.Get("Forwarded")) != "",
		"x_forwarded_for_present", strings.TrimSpace(r.Header.Get("X-Forwarded-For")) != "",
		"x_forwarded_host_present", strings.TrimSpace(r.Header.Get("X-Forwarded-Host")) != "",
		"x_forwarded_proto_present", strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")) != "",
		"x_real_ip_present", strings.TrimSpace(r.Header.Get("X-Real-IP")) != "",
		"via_present", strings.TrimSpace(r.Header.Get("Via")) != "",
	)
}

func logForwardedHeaderRejected(r *http.Request, header string) {
	if r == nil || strings.TrimSpace(header) == "" {
		return
	}
	packageLogger.Warn(
		"AUDIT",
		"event", "forwarded_header_rejected",
		"header", header,
		"peer", loopbackRejectionPeer(r),
	)
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
