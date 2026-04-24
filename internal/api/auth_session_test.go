package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
	"go.uber.org/goleak"
)

func TestAuthSessionManager_SweepExpired_RemovesExpiredSessions_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(time.Hour, time.Hour)
	expired, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-expired"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-expired"), false)
	if err != nil {
		t.Fatalf("CreateCredential(expired) error = %v", err)
	}
	active, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-active"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-active"), false)
	if err != nil {
		t.Fatalf("CreateCredential(active) error = %v", err)
	}

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
	record, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-swept"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-swept"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	manager.mu.Lock()
	expiredRecord := manager.sessions[record.Secret]
	expiredRecord.ExpiresAt = time.Now().UTC().Add(-time.Second)
	manager.sessions[record.Secret] = expiredRecord
	manager.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopSweeper := manager.StartSweeper(ctx, 5*time.Millisecond)
	defer stopSweeper()

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

func TestAuthSessionManager_CreateRecord_RejectsMissingExpiration_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(time.Hour, time.Hour)
	if _, err := manager.createRecord(authSessionRecord{
		Mode:       "tenant",
		TenantID:   "tenant-a",
		Role:       models.TenantRoleAdmin,
		AuthMethod: "tenant-api-key",
	}); err == nil {
		t.Fatal("createRecord() error = nil, want missing expiration rejection")
	}
}

func TestAuthSessionManager_Revoke_ConcurrentLookupStaysRaceSafe_Expected(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	stateStore := store.NewMemoryStore()
	manager := newAuthSessionManager(time.Hour, time.Hour)
	record, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-revoke"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-revoke"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					manager.Lookup(record.Secret)
					manager.LookupByPublicID(record.PublicID)
				}
			}
		}()
	}

	if err := manager.Revoke(context.Background(), stateStore, record, record.Secret, nil); err != nil {
		close(done)
		wg.Wait()
		t.Fatalf("Revoke() error = %v", err)
	}

	close(done)
	wg.Wait()

	if _, ok := manager.Lookup(record.Secret); ok {
		t.Fatal("Lookup() = true, want false after revoke")
	}
	if _, ok, err := manager.LookupActive(context.Background(), stateStore, record.Secret); err != nil {
		t.Fatalf("LookupActive() error = %v", err)
	} else if ok {
		t.Fatal("LookupActive() = true, want replayed secret rejected after revoke")
	}
	revoked, err := stateStore.IsAuthSessionRevoked(context.Background(), record.PublicID)
	if err != nil {
		t.Fatalf("IsAuthSessionRevoked() error = %v", err)
	}
	if !revoked {
		t.Fatal("IsAuthSessionRevoked() = false, want true")
	}
}

func TestAuthSessionManager_Revoke_StoreFailureLeavesSessionActive_Expected(t *testing.T) {
	t.Parallel()

	manager := newAuthSessionManager(time.Hour, time.Hour)
	record, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-retry"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-retry"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	revokeErr := manager.Revoke(context.Background(), failingAuthSessionRevocationStore{
		err: fmt.Errorf("simulated transaction failure"),
	}, record, record.Secret, nil)
	if revokeErr == nil {
		t.Fatal("Revoke() error = nil, want store failure")
	}

	if _, ok := manager.Lookup(record.Secret); !ok {
		t.Fatal("Lookup() = false, want session to remain active after failed revoke")
	}
}

func TestAuthSessionManager_DurableIO_DoesNotHoldSessionMutex_Expected(t *testing.T) {
	manager := newAuthSessionManager(time.Hour, time.Hour)
	record, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-blocking"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-blocking"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	revokeStore := newBlockingAuthSessionRevocationStore()
	revokeErr := make(chan error, 1)
	go func() {
		revokeErr <- manager.Revoke(context.Background(), revokeStore, record, record.Secret, nil)
	}()
	<-revokeStore.revokeStarted

	runSessionOpWithTimeout(t, "Lookup during blocked revoke", func() error {
		if _, ok := manager.Lookup(record.Secret); !ok {
			return fmt.Errorf("Lookup() = false, want active session while durable revoke is pending")
		}
		return nil
	})
	runSessionOpWithTimeout(t, "SweepExpired during blocked revoke", func() error {
		manager.SweepExpired(time.Now().UTC())
		return nil
	})

	close(revokeStore.allowRevoke)
	if err := <-revokeErr; err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	lookupStore := newBlockingAuthSessionRevocationStore()
	record, err = manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-lookup"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-lookup"), false)
	if err != nil {
		t.Fatalf("CreateCredential(lookup) error = %v", err)
	}

	lookupErr := make(chan error, 1)
	go func() {
		_, _, err := manager.LookupActive(context.Background(), lookupStore, record.Secret)
		lookupErr <- err
	}()
	<-lookupStore.lookupStarted

	runSessionOpWithTimeout(t, "Lookup during blocked revocation read", func() error {
		if _, ok := manager.Lookup(record.Secret); !ok {
			return fmt.Errorf("Lookup() = false, want active session while durable revocation read is pending")
		}
		return nil
	})
	runSessionOpWithTimeout(t, "LookupByPublicID during blocked revocation read", func() error {
		if _, _, ok := manager.LookupByPublicID(record.PublicID); !ok {
			return fmt.Errorf("LookupByPublicID() = false, want active session while durable revocation read is pending")
		}
		return nil
	})

	close(lookupStore.allowLookup)
	if err := <-lookupErr; err != nil {
		t.Fatalf("LookupActive() error = %v", err)
	}
}

func TestRevocationAtomicity_RaceOnDBCrash(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	manager := newAuthSessionManager(time.Hour, time.Hour)
	record, err := manager.CreateCredential("tenant", AuthenticatedPrincipal{
		Tenant: models.Tenant{ID: "tenant-race"},
		Role:   models.TenantRoleAdmin,
	}, hashCredential(context.Background(), "hash-race"), false)
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					manager.Lookup(record.Secret)
					manager.LookupByPublicID(record.PublicID)
				}
			}
		}()
	}

	revokeErr := manager.Revoke(context.Background(), failingAuthSessionRevocationStore{
		err: fmt.Errorf("simulated transaction failure"),
	}, record, record.Secret, nil)
	close(done)
	wg.Wait()

	if revokeErr == nil {
		t.Fatal("Revoke() error = nil, want store failure")
	}
	if _, ok := manager.Lookup(record.Secret); !ok {
		t.Fatal("Lookup() = false, want replayed secret to remain active after failed revoke")
	}
}

func TestSweeperGoroutineShutsDown(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	manager := newAuthSessionManager(time.Hour, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	stopSweeper := manager.StartSweeper(ctx, 5*time.Millisecond)
	cancel()
	stopSweeper()
}

type failingAuthSessionRevocationStore struct {
	err error
}

func (s failingAuthSessionRevocationStore) RevokeAuthSession(_ context.Context, _ string, _ time.Time) error {
	return s.err
}

func (failingAuthSessionRevocationStore) IsAuthSessionRevoked(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type blockingAuthSessionRevocationStore struct {
	revokeStarted chan struct{}
	allowRevoke   chan struct{}
	lookupStarted chan struct{}
	allowLookup   chan struct{}
}

func newBlockingAuthSessionRevocationStore() *blockingAuthSessionRevocationStore {
	return &blockingAuthSessionRevocationStore{
		revokeStarted: make(chan struct{}),
		allowRevoke:   make(chan struct{}),
		lookupStarted: make(chan struct{}),
		allowLookup:   make(chan struct{}),
	}
}

func (s *blockingAuthSessionRevocationStore) RevokeAuthSession(ctx context.Context, _ string, _ time.Time) error {
	close(s.revokeStarted)
	select {
	case <-s.allowRevoke:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingAuthSessionRevocationStore) IsAuthSessionRevoked(ctx context.Context, _ string) (bool, error) {
	close(s.lookupStarted)
	select {
	case <-s.allowLookup:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func runSessionOpWithTimeout(t *testing.T, name string, fn func() error) {
	t.Helper()

	errCh := make(chan error, 1)
	go func() {
		errCh <- fn()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("%s timed out while durable store I/O was in flight", name)
	}
}
