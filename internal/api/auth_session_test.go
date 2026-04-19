package api

import (
	"context"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestAuthSessionManager_SweepExpired_RemovesExpiredSessions_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(time.Hour, time.Hour)
	expired := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-expired"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-expired"), false)
	active := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-active"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-active"), false)

	manager.mu.Lock()
	expiredRecord := manager.sessions[expired.Secret]
	expiredRecord.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	manager.sessions[expired.Secret] = expiredRecord
	activeRecord := manager.sessions[active.Secret]
	activeRecord.ExpiresAt = time.Now().UTC().Add(time.Hour)
	manager.sessions[active.Secret] = activeRecord
	manager.mu.Unlock()

	removed := manager.SweepExpired(time.Now().UTC())
	if removed != 1 {
		t.Fatalf("SweepExpired() removed %d sessions, want 1", removed)
	}
	if _, ok := manager.Lookup(expired.Secret); ok {
		t.Fatal("Lookup(expired) = true, want false")
	}
	if _, ok := manager.Lookup(active.Secret); !ok {
		t.Fatal("Lookup(active) = false, want true")
	}
}

func TestAuthSessionManager_StartSweeper_RemovesExpiredSessions_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(time.Hour, time.Hour)
	record := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-swept"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-swept"), false)

	manager.mu.Lock()
	expiredRecord := manager.sessions[record.Secret]
	expiredRecord.ExpiresAt = time.Now().UTC().Add(-time.Second)
	manager.sessions[record.Secret] = expiredRecord
	manager.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.StartSweeper(ctx, 5*time.Millisecond)

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := manager.Lookup(record.Secret); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("session sweeper did not remove expired session before timeout")
}

func TestAuthSessionManager_DefaultPersistentTTL_CappedToSevenDays_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(0, 0)
	if manager.persistentTTL != 7*24*time.Hour {
		t.Fatalf("persistentTTL = %s, want %s", manager.persistentTTL, 7*24*time.Hour)
	}
}
