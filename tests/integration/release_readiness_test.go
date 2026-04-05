package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/connectors/kvm"
	migratepkg "github.com/eblackrps/viaduct/internal/migrate"
	"github.com/eblackrps/viaduct/internal/store"
)

func TestQuickstart_KVMLabDiscoveryAndPlan_Expected(t *testing.T) {
	t.Parallel()

	connector := kvm.NewKVMConnector(connectors.Config{
		Address: filepath.Join("..", "..", "examples", "lab", "kvm"),
	})
	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() {
		_ = connector.Close()
	}()

	result, err := connector.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(result.VMs) == 0 {
		t.Fatal("len(VMs) = 0, want discovered lab VMs")
	}

	stateStore := store.NewMemoryStore()
	defer stateStore.Close()
	if _, err := stateStore.SaveDiscovery(context.Background(), store.DefaultTenantID, result); err != nil {
		t.Fatalf("SaveDiscovery() error = %v", err)
	}

	spec, err := migratepkg.ParseSpec(filepath.Join("..", "..", "examples", "lab", "migration-window.yaml"))
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}
	if !spec.Options.Approval.Required {
		t.Fatal("Approval.Required = false, want true")
	}
	if !spec.Options.Waves.DependencyAware {
		t.Fatal("DependencyAware = false, want true")
	}
}
