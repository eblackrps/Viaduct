package discovery

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

type mockConnector struct {
	connectErr  error
	discoverErr error
	closeErr    error
	result      *models.DiscoveryResult
	delay       time.Duration
	connects    atomic.Int32
	closes      atomic.Int32
}

func (m *mockConnector) Connect(ctx context.Context) error {
	m.connects.Add(1)
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.connectErr
}

func (m *mockConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.result, m.discoverErr
}

func (m *mockConnector) Platform() models.Platform {
	if m.result != nil {
		return m.result.Platform
	}
	return models.PlatformVMware
}

func (m *mockConnector) Close() error {
	m.closes.Add(1)
	return m.closeErr
}

var _ connectors.Connector = (*mockConnector)(nil)

func TestEngine_RunAll_SingleSource(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	engine.AddSource("lab-vcenter", &mockConnector{
		result: &models.DiscoveryResult{
			Source:   "lab-vcenter",
			Platform: models.PlatformVMware,
			VMs: []models.VirtualMachine{
				{Name: "web-01", CPUCount: 4, MemoryMB: 8192},
			},
		},
	})

	result, err := engine.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	if result.TotalVMs != 1 {
		t.Fatalf("TotalVMs = %d, want 1", result.TotalVMs)
	}

	if result.TotalCPU != 4 {
		t.Fatalf("TotalCPU = %d, want 4", result.TotalCPU)
	}
}

func TestEngine_RunAll_MultipleSources(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	engine.AddSource("vcenter", &mockConnector{
		result: &models.DiscoveryResult{
			Source:   "vcenter",
			Platform: models.PlatformVMware,
			VMs: []models.VirtualMachine{
				{Name: "web-01", CPUCount: 2, MemoryMB: 2048},
			},
		},
	})
	engine.AddSource("proxmox", &mockConnector{
		result: &models.DiscoveryResult{
			Source:   "proxmox",
			Platform: models.PlatformProxmox,
			VMs: []models.VirtualMachine{
				{Name: "db-01", CPUCount: 4, MemoryMB: 4096},
				{Name: "cache-01", CPUCount: 1, MemoryMB: 1024},
			},
		},
	})

	result, err := engine.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	if result.TotalVMs != 3 {
		t.Fatalf("TotalVMs = %d, want 3", result.TotalVMs)
	}

	if len(result.ByPlatform) != 2 {
		t.Fatalf("len(ByPlatform) = %d, want 2", len(result.ByPlatform))
	}
}

func TestEngine_RunAll_ConnectorError(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	engine.AddSource("broken", &mockConnector{discoverErr: errors.New("boom")})

	result, err := engine.RunAll(context.Background())
	if err == nil {
		t.Fatal("RunAll() error = nil, want error")
	}

	if result == nil {
		t.Fatal("RunAll() result = nil, want merged result")
	}

	if len(result.Errors) == 0 {
		t.Fatal("Errors = empty, want aggregated error messages")
	}
}

func TestEngine_RunAll_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	engine := NewEngine()
	for idx := 0; idx < 5; idx++ {
		engine.AddSource(time.Now().Add(time.Duration(idx)*time.Second).Format(time.RFC3339Nano), &mockConnector{
			delay: 10 * time.Millisecond,
			result: &models.DiscoveryResult{
				Source:   "source",
				Platform: models.PlatformProxmox,
				VMs: []models.VirtualMachine{
					{Name: "vm", CPUCount: 1, MemoryMB: 512},
				},
			},
		})
	}

	result, err := engine.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	if result.TotalVMs != 5 {
		t.Fatalf("TotalVMs = %d, want 5", result.TotalVMs)
	}
}
