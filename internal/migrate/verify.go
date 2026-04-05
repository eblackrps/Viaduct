package migrate

import (
	"context"
	"fmt"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
)

// BootVerifier waits for a migrated workload to boot successfully.
type BootVerifier interface {
	// WaitForBoot blocks until the VM boots or the timeout expires.
	WaitForBoot(ctx context.Context, vmID string, timeout time.Duration) error
}

// VMwareBootVerifier verifies VMware-backed boot readiness.
type VMwareBootVerifier struct {
	connector connectors.Connector
}

// NewVMwareBootVerifier creates a VMware boot verifier.
func NewVMwareBootVerifier(connector connectors.Connector) *VMwareBootVerifier {
	return &VMwareBootVerifier{connector: connector}
}

// WaitForBoot waits for a VMware workload to pass generic verification.
func (v *VMwareBootVerifier) WaitForBoot(ctx context.Context, vmID string, timeout time.Duration) error {
	return newGenericBootVerifier(v.connector).WaitForBoot(ctx, vmID, timeout)
}

// ProxmoxBootVerifier verifies Proxmox-backed boot readiness.
type ProxmoxBootVerifier struct {
	connector connectors.Connector
}

// NewProxmoxBootVerifier creates a Proxmox boot verifier.
func NewProxmoxBootVerifier(connector connectors.Connector) *ProxmoxBootVerifier {
	return &ProxmoxBootVerifier{connector: connector}
}

// WaitForBoot waits for a Proxmox workload to pass generic verification.
func (v *ProxmoxBootVerifier) WaitForBoot(ctx context.Context, vmID string, timeout time.Duration) error {
	return newGenericBootVerifier(v.connector).WaitForBoot(ctx, vmID, timeout)
}

// GenericBootVerifier provides a connector-agnostic boot verification fallback.
type GenericBootVerifier struct {
	connector connectors.Connector
}

// NewGenericBootVerifier creates a generic boot verifier.
func NewGenericBootVerifier(connector connectors.Connector) *GenericBootVerifier {
	return newGenericBootVerifier(connector)
}

// WaitForBoot waits for connector verification or timeout.
func (v *GenericBootVerifier) WaitForBoot(ctx context.Context, vmID string, timeout time.Duration) error {
	if verifier, ok := v.connector.(vmVerifier); ok {
		verifyCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			verifyCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		defer cancel()

		if err := verifier.VerifyVM(verifyCtx, vmID); err != nil {
			return fmt.Errorf("wait for boot %s: %w", vmID, err)
		}
		return nil
	}

	if timeout <= 0 {
		return nil
	}

	timer := time.NewTimer(minDuration(timeout, 10*time.Millisecond))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait for boot: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func newGenericBootVerifier(connector connectors.Connector) *GenericBootVerifier {
	return &GenericBootVerifier{connector: connector}
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
