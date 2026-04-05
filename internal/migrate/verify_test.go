package migrate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

type blockingVerifyConnector struct{}

func (b *blockingVerifyConnector) Connect(context.Context) error { return nil }
func (b *blockingVerifyConnector) Discover(context.Context) (*models.DiscoveryResult, error) {
	return &models.DiscoveryResult{Platform: models.PlatformProxmox}, nil
}
func (b *blockingVerifyConnector) Platform() models.Platform { return models.PlatformProxmox }
func (b *blockingVerifyConnector) Close() error              { return nil }
func (b *blockingVerifyConnector) VerifyVM(ctx context.Context, vmID string) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestGenericBootVerifier_WaitForBoot_TimeoutEnforced_Expected(t *testing.T) {
	t.Parallel()

	verifier := NewGenericBootVerifier(&blockingVerifyConnector{})
	startedAt := time.Now()
	err := verifier.WaitForBoot(context.Background(), "vm-timeout", 25*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForBoot() error = nil, want timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitForBoot() error = %v, want deadline exceeded", err)
	}
	if time.Since(startedAt) > 500*time.Millisecond {
		t.Fatalf("WaitForBoot() exceeded expected timeout window: %s", time.Since(startedAt))
	}
}
