package vmware

import (
	"context"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestNewVMwareConnector_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  connectors.Config
	}{
		{
			name: "configured address",
			cfg:  connectors.Config{Address: "vcenter.example.com"},
		},
		{
			name: "empty config",
			cfg:  connectors.Config{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := NewVMwareConnector(tt.cfg)
			if conn == nil {
				t.Fatal("NewVMwareConnector() returned nil")
			}

			if conn.config != tt.cfg {
				t.Fatalf("connector config = %#v, want %#v", conn.config, tt.cfg)
			}
		})
	}
}

func TestVMwareConnector_DiscoverBeforeConnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  connectors.Config
	}{
		{
			name: "default config",
			cfg:  connectors.Config{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := NewVMwareConnector(tt.cfg)
			_, err := conn.Discover(context.Background())
			if err == nil {
				t.Fatal("Discover() error = nil, want error")
			}

			if !strings.Contains(err.Error(), "not connected") {
				t.Fatalf("Discover() error = %q, want substring %q", err.Error(), "not connected")
			}
		})
	}
}

func TestVMwareConnector_Platform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  connectors.Config
		want models.Platform
	}{
		{
			name: "vmware platform",
			cfg:  connectors.Config{},
			want: models.PlatformVMware,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := NewVMwareConnector(tt.cfg)
			if got := conn.Platform(); got != tt.want {
				t.Fatalf("Platform() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVMwareConnector_ConnectAndClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  connectors.Config
	}{
		{
			name: "connect then close",
			cfg:  connectors.Config{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			conn := NewVMwareConnector(tt.cfg)

			if err := conn.Connect(context.Background()); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}

			if !conn.connected {
				t.Fatal("connected = false, want true after Connect")
			}

			if err := conn.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}

			if conn.connected {
				t.Fatal("connected = true, want false after Close")
			}
		})
	}
}
