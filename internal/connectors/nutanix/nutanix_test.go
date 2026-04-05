package nutanix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

func TestMapNutanixVM_Expected(t *testing.T) {
	t.Parallel()

	vm := mapNutanixVM(readNutanixFixture(t, "vm.json"))
	if vm.Name != "ahv-web-01" || vm.CPUCount != 4 || vm.Cluster != "Cluster-A" {
		t.Fatalf("unexpected VM: %#v", vm)
	}
}

func TestMapNutanixPowerState_Values_Expected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		state string
		want  models.PowerState
	}{
		{name: "on", state: "ON", want: models.PowerOn},
		{name: "off", state: "OFF", want: models.PowerOff},
		{name: "suspended", state: "SUSPENDED", want: models.PowerSuspend},
		{name: "unknown", state: "UNKNOWN", want: models.PowerUnknown},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := mapNutanixPowerState(tc.state); got != tc.want {
				t.Fatalf("mapNutanixPowerState() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapNutanixDisks_Expected(t *testing.T) {
	t.Parallel()

	vmFixture := readNutanixFixture(t, "vm.json")
	disks := mapNutanixDisks(mapValue(mapValue(vmFixture, "spec"), "resources"))
	if len(disks) != 1 || disks[0].StorageBackend != "container-a" {
		t.Fatalf("unexpected disks: %#v", disks)
	}
}

func TestMapNutanixNICs_Expected(t *testing.T) {
	t.Parallel()

	vmFixture := readNutanixFixture(t, "vm.json")
	nics := mapNutanixNICs(mapValue(mapValue(vmFixture, "spec"), "resources"))
	if len(nics) != 1 || nics[0].Network != "Prod-Subnet" || len(nics[0].IPAddresses) != 1 {
		t.Fatalf("unexpected NICs: %#v", nics)
	}
}

func TestNutanixConnector_Platform_Expected(t *testing.T) {
	t.Parallel()

	connector := NewNutanixConnector(connectors.Config{})
	if got := connector.Platform(); got != models.PlatformNutanix {
		t.Fatalf("Platform() = %q, want %q", got, models.PlatformNutanix)
	}
}

func TestNutanixConnector_Pagination_ListAllPages(t *testing.T) {
	t.Parallel()

	pageRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/nutanix/v3/vms/list":
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			offset := int(request["offset"].(float64))
			pageRequests++

			response := map[string]interface{}{
				"entities": []map[string]interface{}{readNutanixFixture(t, "vm.json")},
				"metadata": map[string]int{
					"offset":        offset,
					"length":        1,
					"total_matches": 2,
				},
			}
			if offset > 0 {
				response["entities"] = []map[string]interface{}{readNutanixFixture(t, "vm-page-2.json")}
			}
			writeNutanixJSON(t, w, response)
		default:
			writeNutanixJSON(t, w, map[string]interface{}{"entities": []map[string]interface{}{}, "metadata": map[string]int{"offset": 0, "length": 0, "total_matches": 0}})
		}
	}))
	defer server.Close()

	client := NewPrismClient(server.URL, "admin", "password", true)
	entities, err := client.ListAll(context.Background(), "/api/nutanix/v3/vms/list", map[string]interface{}{"kind": "vm", "length": 1})
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(entities) != 2 || pageRequests != 2 {
		t.Fatalf("unexpected pagination results: len=%d requests=%d", len(entities), pageRequests)
	}
}

func readNutanixFixture(t *testing.T, name string) map[string]interface{} {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", name, err)
	}

	var item map[string]interface{}
	if err := json.Unmarshal(payload, &item); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", name, err)
	}
	return item
}

func writeNutanixJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
