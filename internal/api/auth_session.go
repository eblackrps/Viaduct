package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const dashboardSessionCookieName = "viaduct_dashboard_session"

type authSessionRecord struct {
	PublicID   string
	Secret     string
	Mode       string
	APIKey     string
	Persistent bool
	ExpiresAt  time.Time
}

type authSessionManager struct {
	mu            sync.Mutex
	sessionTTL    time.Duration
	persistentTTL time.Duration
	sessions      map[string]authSessionRecord
}

func newAuthSessionManager(sessionTTL, persistentTTL time.Duration) *authSessionManager {
	if sessionTTL <= 0 {
		sessionTTL = 12 * time.Hour
	}
	if persistentTTL <= 0 {
		persistentTTL = 30 * 24 * time.Hour
	}
	return &authSessionManager{
		sessionTTL:    sessionTTL,
		persistentTTL: persistentTTL,
		sessions:      make(map[string]authSessionRecord),
	}
}

func (m *authSessionManager) Create(mode, apiKey string, persistent bool) authSessionRecord {
	if m == nil {
		return authSessionRecord{}
	}

	ttl := m.sessionTTL
	if persistent {
		ttl = m.persistentTTL
	}
	record := authSessionRecord{
		PublicID:   uuid.NewString(),
		Secret:     uuid.NewString(),
		Mode:       strings.TrimSpace(mode),
		APIKey:     strings.TrimSpace(apiKey),
		Persistent: persistent,
		ExpiresAt:  time.Now().UTC().Add(ttl),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(time.Now().UTC())
	m.sessions[record.Secret] = record
	return record
}

func (m *authSessionManager) Lookup(secret string) (authSessionRecord, bool) {
	if m == nil {
		return authSessionRecord{}, false
	}

	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	record, ok := m.sessions[strings.TrimSpace(secret)]
	if !ok {
		return authSessionRecord{}, false
	}
	if !record.ExpiresAt.IsZero() && now.After(record.ExpiresAt) {
		delete(m.sessions, secret)
		return authSessionRecord{}, false
	}
	return record, true
}

func (m *authSessionManager) Delete(secret string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, strings.TrimSpace(secret))
}

func (m *authSessionManager) StartSweeper(ctx context.Context, interval time.Duration) {
	if m == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				m.SweepExpired(now.UTC())
			}
		}
	}()
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
	for secret, record := range m.sessions {
		if !record.ExpiresAt.IsZero() && now.After(record.ExpiresAt) {
			delete(m.sessions, secret)
		}
	}
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

func setAuthSessionCookie(w http.ResponseWriter, r *http.Request, record authSessionRecord) {
	if w == nil {
		return
	}

	cookie := &http.Cookie{
		Name:     dashboardSessionCookieName,
		Value:    record.Secret,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestScheme(r) == "https",
	}
	if record.Persistent {
		cookie.Expires = record.ExpiresAt
		cookie.MaxAge = int(time.Until(record.ExpiresAt).Seconds())
	}
	http.SetCookie(w, cookie)
}

func clearAuthSessionCookie(w http.ResponseWriter, r *http.Request) {
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
		Secure:   requestScheme(r) == "https",
	})
}
