package integration

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPISpec_StableRoutesDocumented_Expected(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "docs", "reference", "openapi.yaml")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatalf("openapi = %#v, want 3.1.0", document["openapi"])
	}

	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths = %#v, want map", document["paths"])
	}
	for _, route := range []string{
		"/api/v1/health",
		"/api/v1/about",
		"/api/v1/inventory",
		"/api/v1/summary",
		"/api/v1/workspaces",
		"/api/v1/workspaces/{workspaceID}",
		"/api/v1/tenants/current",
		"/api/v1/service-accounts",
		"/api/v1/migrations",
	} {
		if _, ok := paths[route]; !ok {
			t.Fatalf("OpenAPI paths missing %s", route)
		}
	}

	components, ok := document["components"].(map[string]any)
	if !ok {
		t.Fatalf("components = %#v, want map", document["components"])
	}
	securitySchemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatalf("securitySchemes = %#v, want map", components["securitySchemes"])
	}
	for _, scheme := range []string{"TenantAPIKey", "ServiceAccountKey", "AdminAPIKey"} {
		if _, ok := securitySchemes[scheme]; !ok {
			t.Fatalf("securitySchemes missing %s", scheme)
		}
	}
}
