package proxmox

import (
	"context"
	"testing"
)

func TestDiscoverNetworks_MockServer_ReturnsNetworks(t *testing.T) {
	t.Parallel()

	server := mockProxmoxServer(t)
	defer server.Close()

	client := NewProxmoxClient(server.URL, true)
	client.AuthenticateToken("root@pam!viaduct", "secret")

	networks, err := discoverNetworks(context.Background(), client)
	if err != nil {
		t.Fatalf("discoverNetworks() error = %v", err)
	}

	if len(networks) == 0 {
		t.Fatal("discoverNetworks() returned no networks")
	}
}

func TestDiscoverStorage_MockServer_ReturnsDatastores(t *testing.T) {
	t.Parallel()

	server := mockProxmoxServer(t)
	defer server.Close()

	client := NewProxmoxClient(server.URL, true)
	client.AuthenticateToken("root@pam!viaduct", "secret")

	datastores, err := discoverStorage(context.Background(), client)
	if err != nil {
		t.Fatalf("discoverStorage() error = %v", err)
	}

	if len(datastores) == 0 {
		t.Fatal("discoverStorage() returned no datastores")
	}
}

func TestDiscoverNodes_MockServer_ReturnsHosts(t *testing.T) {
	t.Parallel()

	server := mockProxmoxServer(t)
	defer server.Close()

	client := NewProxmoxClient(server.URL, true)
	client.AuthenticateToken("root@pam!viaduct", "secret")

	hosts, err := discoverNodes(context.Background(), client)
	if err != nil {
		t.Fatalf("discoverNodes() error = %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("len(hosts) = %d, want 2", len(hosts))
	}
}
