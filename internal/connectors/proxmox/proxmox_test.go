package proxmox

import (
	"context"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestNewProxmoxConnector_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	conn := NewProxmoxConnector(connectors.Config{Address: "pve.example.com"})
	if conn == nil {
		t.Fatal("NewProxmoxConnector() returned nil")
	}
}

func TestProxmoxConnector_DiscoverBeforeConnect(t *testing.T) {
	t.Parallel()

	conn := NewProxmoxConnector(connectors.Config{})
	_, err := conn.Discover(context.Background())
	if err == nil {
		t.Fatal("Discover() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("Discover() error = %q, want substring %q", err.Error(), "not connected")
	}
}

func TestProxmoxConnector_Platform(t *testing.T) {
	t.Parallel()

	conn := NewProxmoxConnector(connectors.Config{})
	if got := conn.Platform(); got != models.PlatformProxmox {
		t.Fatalf("Platform() = %q, want %q", got, models.PlatformProxmox)
	}
}

func TestProxmoxConnector_ConnectAndClose(t *testing.T) {
	t.Parallel()

	conn := NewProxmoxConnector(connectors.Config{})

	if err := conn.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
