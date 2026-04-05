package proxmox

import (
	"context"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
)

func TestProxmoxConnector_FullDiscovery_ReturnsInventory(t *testing.T) {
	t.Parallel()

	server := mockProxmoxServer(t)
	defer server.Close()

	connector := NewProxmoxConnector(connectors.Config{
		Address:  server.URL,
		Username: "root@pam",
		Password: "secret",
		Insecure: true,
	})

	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer connector.Close()

	result, err := connector.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(result.VMs) != 5 {
		t.Fatalf("len(VMs) = %d, want 5", len(result.VMs))
	}

	var webFound bool
	var proxyFound bool
	for _, vm := range result.VMs {
		switch vm.Name {
		case "web-01":
			webFound = true
			if len(vm.Disks) == 0 || len(vm.NICs) == 0 {
				t.Fatalf("web-01 disks/nics not parsed: %#v", vm)
			}
		case "proxy-01":
			proxyFound = true
		}
	}

	if !webFound || !proxyFound {
		t.Fatalf("expected VM names missing from discovery result: %#v", result.VMs)
	}
}
