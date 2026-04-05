package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/connectors/proxmox"
	"github.com/eblackrps/viaduct/internal/discovery"
)

func TestFullDiscoveryPipeline_ProxmoxMock_ReturnsMergedTotals(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"ticket": "mock", "CSRFPreventionToken": "mock"}})
		case "/api2/json/nodes":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"node": "pve-01"}}})
		case "/api2/json/nodes/pve-01/qemu":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"vmid": "101", "name": "web-01", "status": "running", "cpus": 2, "maxmem": 2147483648}}})
		case "/api2/json/nodes/pve-01/qemu/101/config":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"vmid": "101", "name": "web-01", "cores": 2, "memory": 2048, "scsi0": "local-lvm:vm-101-disk-0,size=32G", "net0": "virtio=AA:BB:CC:DD:EE:01,bridge=vmbr0"}})
		case "/api2/json/nodes/pve-01/lxc":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"vmid": "301", "name": "proxy-01", "status": "running", "maxmem": 536870912}}})
		case "/api2/json/nodes/pve-01/lxc/301/config":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"vmid": "301", "hostname": "proxy-01", "cores": 1, "memory": 512, "rootfs": "local-lvm:subvol-301-disk-0,size=8G", "net0": "name=eth0,bridge=vmbr1,hwaddr=AA:BB:CC:DD:EE:11"}})
		case "/api2/json/nodes/pve-01/network":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"iface": "vmbr0", "type": "bridge"}}})
		case "/api2/json/storage":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"storage": "local-lvm", "type": "lvmthin", "node": "pve-01"}}})
		case "/api2/json/nodes/pve-01/storage/local-lvm/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"total": 107374182400, "avail": 53687091200}})
		case "/api2/json/nodes/pve-01/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"cpuinfo": map[string]any{"cpus": 16}, "memory": 137438953472, "status": "online"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	engine := discovery.NewEngine()
	engine.AddSource(server.URL, proxmox.NewProxmoxConnector(connectors.Config{
		Address:  server.URL,
		Username: "root@pam",
		Password: "secret",
		Insecure: true,
	}))

	result, err := engine.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}

	if result.TotalVMs != 2 {
		t.Fatalf("TotalVMs = %d, want 2", result.TotalVMs)
	}

	if result.TotalCPU != 3 {
		t.Fatalf("TotalCPU = %d, want 3", result.TotalCPU)
	}

	if len(result.Sources) != 1 || len(result.Sources[0].VMs) != 2 {
		t.Fatalf("unexpected merged sources: %#v", result.Sources)
	}
}
