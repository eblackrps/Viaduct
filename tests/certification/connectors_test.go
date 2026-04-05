package certification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/connectors/kvm"
	"github.com/eblackrps/viaduct/internal/connectors/proxmox"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestConnectorCertification_KVMFixtureDiscovery_Normalized(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "examples", "lab", "kvm")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("Stat(%s) error = %v", root, err)
	}

	connector := kvm.NewKVMConnector(connectors.Config{Address: root})
	if err := connector.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer connector.Close()

	result, err := connector.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertNormalizedDiscoveryResult(t, models.PlatformKVM, result)
}

func TestConnectorCertification_ProxmoxFixtureDiscovery_Normalized(t *testing.T) {
	t.Parallel()

	server := certificationProxmoxServer(t)
	defer server.Close()

	connector := proxmox.NewProxmoxConnector(connectors.Config{
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

	assertNormalizedDiscoveryResult(t, models.PlatformProxmox, result)
}

func assertNormalizedDiscoveryResult(t *testing.T, platform models.Platform, result *models.DiscoveryResult) {
	t.Helper()

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Platform != platform {
		t.Fatalf("result.Platform = %q, want %q", result.Platform, platform)
	}
	if result.Source == "" {
		t.Fatal("result.Source is empty")
	}
	if result.DiscoveredAt.IsZero() {
		t.Fatal("result.DiscoveredAt is zero")
	}
	if len(result.VMs) == 0 {
		t.Fatal("result.VMs is empty")
	}
	for _, vm := range result.VMs {
		if vm.Name == "" {
			t.Fatalf("vm.Name is empty: %#v", vm)
		}
		if vm.Platform != platform {
			t.Fatalf("vm.Platform = %q, want %q", vm.Platform, platform)
		}
		if vm.DiscoveredAt.IsZero() {
			t.Fatalf("vm.DiscoveredAt is zero: %#v", vm)
		}
	}
}

func certificationProxmoxServer(t *testing.T) *httptest.Server {
	t.Helper()

	read := func(name string) []byte {
		payload, err := os.ReadFile(filepath.Join("..", "..", "internal", "connectors", "proxmox", "testdata", name))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		return payload
	}

	respond := func(w http.ResponseWriter, body []byte) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}

	writeFilteredList := func(w http.ResponseWriter, payload []byte, key, value string) {
		t.Helper()

		var envelope struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("Unmarshal filtered payload error = %v", err)
		}

		filtered := make([]map[string]any, 0, len(envelope.Data))
		for _, item := range envelope.Data {
			if itemValue, ok := item[key].(string); ok && itemValue == value {
				filtered = append(filtered, item)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": filtered})
	}

	writeConfigByID := func(w http.ResponseWriter, payload []byte, id string) {
		t.Helper()

		var envelope struct {
			Data map[string]map[string]any `json:"data"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("Unmarshal config payload error = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": envelope.Data[id]})
	}

	writeNamedPayload := func(w http.ResponseWriter, payload []byte, name string) {
		t.Helper()

		var envelope struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("Unmarshal named payload error = %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": envelope.Data[name]})
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/access/ticket"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"ticket":"ticket","CSRFPreventionToken":"csrf"}}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes"):
			respond(w, read("proxmox_nodes.json"))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/qemu"):
			writeFilteredList(w, read("proxmox_qemu_list.json"), "node", "pve-01")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/qemu/101/config"):
			writeConfigByID(w, read("proxmox_vm_config.json"), "101")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/qemu/102/config"):
			writeConfigByID(w, read("proxmox_vm_config.json"), "102")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-02/qemu"):
			writeFilteredList(w, read("proxmox_qemu_list.json"), "node", "pve-02")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-02/qemu/201/config"):
			writeConfigByID(w, read("proxmox_vm_config.json"), "201")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/lxc"):
			writeFilteredList(w, read("proxmox_lxc_list.json"), "node", "pve-01")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/lxc/301/config"):
			writeConfigByID(w, read("proxmox_lxc_config.json"), "301")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-02/lxc"):
			writeFilteredList(w, read("proxmox_lxc_list.json"), "node", "pve-02")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-02/lxc/302/config"):
			writeConfigByID(w, read("proxmox_lxc_config.json"), "302")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-01/network"):
			writeNamedPayload(w, read("proxmox_network.json"), "pve-01")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/nodes/pve-02/network"):
			writeNamedPayload(w, read("proxmox_network.json"), "pve-02")
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/storage"):
			respond(w, read("proxmox_storage.json"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/storage/") && strings.HasSuffix(r.URL.Path, "/status"):
			if strings.Contains(r.URL.Path, "/local-lvm/") {
				writeNamedPayload(w, read("proxmox_storage_status.json"), "local-lvm")
				return
			}
			if strings.Contains(r.URL.Path, "/shared-nfs/") {
				writeNamedPayload(w, read("proxmox_storage_status.json"), "shared-nfs")
				return
			}
			http.NotFound(w, r)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/status"):
			if strings.Contains(r.URL.Path, "/nodes/pve-01/") {
				writeNamedPayload(w, read("proxmox_node_status.json"), "pve-01")
				return
			}
			if strings.Contains(r.URL.Path, "/nodes/pve-02/") {
				writeNamedPayload(w, read("proxmox_node_status.json"), "pve-02")
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}
