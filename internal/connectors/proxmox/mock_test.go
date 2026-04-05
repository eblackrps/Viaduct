package proxmox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mockProxmoxServer(t *testing.T) *httptest.Server {
	t.Helper()

	nodes := mustLoadFixture[apiEnvelope](t, "proxmox_nodes.json")
	qemuList := mustLoadFixture[apiEnvelope](t, "proxmox_qemu_list.json")
	qemuConfigs := mustLoadFixture[apiEnvelope](t, "proxmox_vm_config.json")
	lxcList := mustLoadFixture[apiEnvelope](t, "proxmox_lxc_list.json")
	lxcConfigs := mustLoadFixture[apiEnvelope](t, "proxmox_lxc_config.json")
	networks := mustLoadFixture[apiEnvelope](t, "proxmox_network.json")
	storage := mustLoadFixture[apiEnvelope](t, "proxmox_storage.json")
	nodeStatus := mustLoadFixture[apiEnvelope](t, "proxmox_node_status.json")
	storageStatus := mustLoadFixture[apiEnvelope](t, "proxmox_storage_status.json")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api2/json/access/ticket":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"ticket":              "mock-ticket",
					"CSRFPreventionToken": "mock-csrf",
				},
			})
			return
		case "/api2/json/nodes":
			_ = json.NewEncoder(w).Encode(nodes)
			return
		case "/api2/json/storage":
			_ = json.NewEncoder(w).Encode(storage)
			return
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/api2/json/")
		parts := strings.Split(trimmed, "/")
		if len(parts) < 2 || parts[0] != "nodes" {
			http.NotFound(w, r)
			return
		}

		node := parts[1]
		switch {
		case len(parts) == 3 && parts[2] == "qemu":
			writeFilteredList(t, w, qemuList.Data, "node", node)
		case len(parts) == 5 && parts[2] == "qemu" && parts[4] == "config":
			writeConfigByID(t, w, qemuConfigs.Data, parts[3])
		case len(parts) == 3 && parts[2] == "lxc":
			writeFilteredList(t, w, lxcList.Data, "node", node)
		case len(parts) == 5 && parts[2] == "lxc" && parts[4] == "config":
			writeConfigByID(t, w, lxcConfigs.Data, parts[3])
		case len(parts) == 3 && parts[2] == "network":
			writeNamedPayload(t, w, networks.Data, node)
		case len(parts) == 3 && parts[2] == "status":
			writeNamedPayload(t, w, nodeStatus.Data, node)
		case len(parts) == 5 && parts[2] == "storage" && parts[4] == "status":
			writeNamedPayload(t, w, storageStatus.Data, parts[3])
		default:
			http.NotFound(w, r)
		}
	}))
}

func mustLoadFixture[T any](t *testing.T, name string) T {
	t.Helper()

	path := filepath.Join("testdata", name)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var decoded T
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}

	return decoded
}

func writeFilteredList(t *testing.T, w http.ResponseWriter, payload json.RawMessage, key, value string) {
	t.Helper()

	var items []map[string]any
	if err := json.Unmarshal(payload, &items); err != nil {
		t.Fatalf("Unmarshal filtered list payload error = %v", err)
	}

	filtered := make([]map[string]any, 0)
	for _, item := range items {
		if stringField(item, key, "") == value {
			filtered = append(filtered, item)
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": filtered})
}

func writeConfigByID(t *testing.T, w http.ResponseWriter, payload json.RawMessage, id string) {
	t.Helper()

	var items map[string]map[string]any
	if err := json.Unmarshal(payload, &items); err != nil {
		t.Fatalf("Unmarshal config payload error = %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": items[id]})
}

func writeNamedPayload(t *testing.T, w http.ResponseWriter, payload json.RawMessage, name string) {
	t.Helper()

	var items map[string]any
	if err := json.Unmarshal(payload, &items); err != nil {
		t.Fatalf("Unmarshal named payload error = %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"data": items[name]})
}
