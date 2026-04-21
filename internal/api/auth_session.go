package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/google/uuid"
)

const dashboardSessionCookieName = "viaduct_dashboard_session"

type authSessionRecord struct {
	PublicID         string
	Secret           string
	Mode             string
	CredentialHash   [32]byte
	TenantID         string
	ServiceAccountID string
	Role             models.TenantRole
	AuthMethod       string
	Persistent       bool
	ExpiresAt        time.Time
}

type authSessionManager struct {
	mu            sync.RWMutex
	sessionTTL    time.Duration
	persistentTTL time.Duration
	sessions      map[string]authSessionRecord
	revoked       map[string]time.Time
}

func newAuthSessionManager(sessionTTL, persistentTTL time.Duration) *authSessionManager {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	if persistentTTL <= 0 {
		persistentTTL = 7 * 24 * time.Hour
	}
	return &authSessionManager{
		sessionTTL:    sessionTTL,
		persistentTTL: persistentTTL,
		sessions:      make(map[string]authSessionRecord),
		revoked:       make(map[string]time.Time),
	}
}

type authSessionRevocationStore interface {
	RevokeAuthSession(ctx context.Context, sessionID string, expiresAt time.Time) error
	IsAuthSessionRevoked(ctx context.Context, sessionID string) (bool, error)
}

func (m *authSessionManager) CreateCredential(mode string, principal AuthenticatedPrincipal, credentialHash [32]byte, persistent bool) (authSessionRecord, error) {
	serviceAccountID := ""
	if principal.ServiceAccount != nil {
		serviceAccountID = principal.ServiceAccount.ID
	}
	expiresAt, err := m.expirationFor(persistent)
	if err != nil {
		return authSessionRecord{}, err
	}
	return m.createRecord(authSessionRecord{
		Mode:             strings.TrimSpace(mode),
		CredentialHash:   credentialHash,
		TenantID:         strings.TrimSpace(principal.Tenant.ID),
		ServiceAccountID: strings.TrimSpace(serviceAccountID),
		Role:             principal.Role,
		AuthMethod:       strings.TrimSpace(principal.AuthMethod),
		Persistent:       persistent,
		ExpiresAt:        expiresAt,
	})
}

func (m *authSessionManager) CreateLocal(tenantID string, role models.TenantRole, authMethod string, persistent bool) (authSessionRecord, error) {
	expiresAt, err := m.expirationFor(persistent)
	if err != nil {
		return authSessionRecord{}, err
	}
	return m.createRecord(authSessionRecord{
		Mode:       "local",
		TenantID:   strings.TrimSpace(tenantID),
		Role:       role,
		AuthMethod: strings.TrimSpace(authMethod),
		Persistent: persistent,
		ExpiresAt:  expiresAt,
	})
}

func (m *authSessionManager) createRecord(seed authSessionRecord) (authSessionRecord, error) {
	if m == nil {
		return authSessionRecord{}, fmt.Errorf("auth session manager is not configured")
	}

	now := time.Now().UTC()
	record := authSessionRecord{
		PublicID:         uuid.NewString(),
		Secret:           uuid.NewString(),
		Mode:             strings.TrimSpace(seed.Mode),
		CredentialHash:   seed.CredentialHash,
		TenantID:         strings.TrimSpace(seed.TenantID),
		ServiceAccountID: strings.TrimSpace(seed.ServiceAccountID),
		Role:             seed.Role,
		AuthMethod:       strings.TrimSpace(seed.AuthMethod),
		Persistent:       seed.Persistent,
		ExpiresAt:        seed.ExpiresAt.UTC(),
	}
	if record.ExpiresAt.IsZero() || !record.ExpiresAt.After(now) {
		return authSessionRecord{}, fmt.Errorf("auth session expiration must be set")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(now)
	m.sessions[record.Secret] = record
	return record, nil
}

func (m *authSessionManager) Lookup(secret string) (authSessionRecord, bool) {
	if m == nil {
		return authSessionRecord{}, false
	}

	secret = strings.TrimSpace(secret)
	if secret == "" {
		return authSessionRecord{}, false
	}
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.sessions[secret]
	if !ok {
		return authSessionRecord{}, false
	}
	if m.revokedLocked(record.PublicID, now) || !record.ExpiresAt.After(now) {
		delete(m.sessions, secret)
		return authSessionRecord{}, false
	}
	return record, true
}

func (m *authSessionManager) LookupActive(ctx context.Context, revocations authSessionRevocationStore, secret string) (authSessionRecord, bool, error) {
	record, ok := m.Lookup(secret)
	if !ok || revocations == nil {
		return record, ok, nil
	}

	now := time.Now().UTC()
	m.mu.RLock()
	if m.revokedActiveLocked(record.PublicID, now) {
		m.mu.RUnlock()
		m.MarkRevoked(record, secret)
		return authSessionRecord{}, false, nil
	}
	revoked, err := revocations.IsAuthSessionRevoked(ctx, record.PublicID)
	m.mu.RUnlock()
	if err != nil {
		return authSessionRecord{}, false, err
	}
	if !revoked {
		return record, true, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.markRevokedLocked(record, secret)
	return authSessionRecord{}, false, nil
}

func (m *authSessionManager) Delete(secret string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, strings.TrimSpace(secret))
}

func (m *authSessionManager) LookupByPublicID(publicID string) (authSessionRecord, string, bool) {
	if m == nil {
		return authSessionRecord{}, "", false
	}

	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return authSessionRecord{}, "", false
	}

	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for secret, record := range m.sessions {
		if record.PublicID != publicID {
			continue
		}
		if m.revokedLocked(record.PublicID, now) || !record.ExpiresAt.After(now) {
			delete(m.sessions, secret)
			return authSessionRecord{}, "", false
		}
		return record, secret, true
	}
	return authSessionRecord{}, "", false
}

func (m *authSessionManager) StartSweeper(ctx context.Context, interval time.Duration) func() {
	if m == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sweepCtx, cancel := context.WithCancel(ctx)
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ticker.Stop()
		for {
			select {
			case <-sweepCtx.Done():
				return
			case now := <-ticker.C:
				m.SweepExpired(now.UTC())
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
}

func (m *authSessionManager) expirationFor(persistent bool) (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("auth session manager is not configured")
	}

	ttl := m.sessionTTL
	if persistent {
		ttl = m.persistentTTL
	}
	expiresAt := time.Now().UTC().Add(ttl)
	if expiresAt.IsZero() {
		return time.Time{}, fmt.Errorf("auth session expiration must be set")
	}
	return expiresAt, nil
}

func (m *authSessionManager) SweepExpired(now time.Time) int {
	if m == nil {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	before := len(m.sessions)
	m.pruneLocked(now)
	return before - len(m.sessions)
}

func (m *authSessionManager) pruneLocked(now time.Time) {
	for publicID, expiresAt := range m.revoked {
		if !expiresAt.After(now) {
			delete(m.revoked, publicID)
		}
	}
	for secret, record := range m.sessions {
		if m.revokedLocked(record.PublicID, now) || !record.ExpiresAt.After(now) {
			delete(m.sessions, secret)
		}
	}
}

func (m *authSessionManager) MarkRevoked(record authSessionRecord, secret string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.markRevokedLocked(record, secret)
}

func (m *authSessionManager) Revoke(ctx context.Context, revocations authSessionRevocationStore, record authSessionRecord, secret string, recordAudit func(context.Context) error) error {
	if m == nil {
		return fmt.Errorf("auth session manager is not configured")
	}
	if revocations == nil {
		return fmt.Errorf("auth session revocation store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := revocations.RevokeAuthSession(ctx, record.PublicID, record.ExpiresAt); err != nil {
		return err
	}
	if recordAudit != nil {
		if err := recordAudit(ctx); err != nil {
			return err
		}
	}

	m.markRevokedLocked(record, secret)
	return nil
}

func (m *authSessionManager) markRevokedLocked(record authSessionRecord, secret string) {
	m.pruneLocked(time.Now().UTC())
	if record.PublicID != "" && record.ExpiresAt.After(time.Now().UTC()) {
		m.revoked[record.PublicID] = record.ExpiresAt.UTC()
	}
	delete(m.sessions, strings.TrimSpace(secret))
}

func (m *authSessionManager) revokedLocked(publicID string, now time.Time) bool {
	expiresAt, ok := m.revoked[strings.TrimSpace(publicID)]
	if !ok {
		return false
	}
	if !expiresAt.After(now) {
		delete(m.revoked, publicID)
		return false
	}
	return true
}

func (m *authSessionManager) revokedActiveLocked(publicID string, now time.Time) bool {
	expiresAt, ok := m.revoked[strings.TrimSpace(publicID)]
	return ok && expiresAt.After(now)
}

func readAuthSessionSecret(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(dashboardSessionCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func setAuthSessionCookie(w http.ResponseWriter, record authSessionRecord, secure bool) {
	if w == nil {
		return
	}

	cookie := &http.Cookie{
		Name:     dashboardSessionCookieName,
		Value:    record.Secret,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	}
	if record.Persistent {
		cookie.Expires = record.ExpiresAt
		cookie.MaxAge = int(time.Until(record.ExpiresAt).Seconds())
	}
	http.SetCookie(w, cookie)
}

func clearAuthSessionCookie(w http.ResponseWriter, secure bool) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     dashboardSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}
