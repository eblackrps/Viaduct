package integration

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPISpec_Phase5RoutesAndSchemasDocumented_Expected(t *testing.T) {
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

	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths = %#v, want map", document["paths"])
	}
	for _, route := range []string{
		"/api/v1/audit",
		"/api/v1/migrations/{migrationID}/execute",
		"/api/v1/migrations/{migrationID}/resume",
		"/api/v1/migrations/{migrationID}/rollback",
		"/api/v1/workspaces/{workspaceID}/jobs",
		"/api/v1/workspaces/{workspaceID}/jobs/{jobID}",
		"/api/v1/workspaces/{workspaceID}/reports/export",
	} {
		if _, ok := paths[route]; !ok {
			t.Fatalf("OpenAPI paths missing %s", route)
		}
	}

	components, ok := document["components"].(map[string]any)
	if !ok {
		t.Fatalf("components = %#v, want map", document["components"])
	}

	responses, ok := components["responses"].(map[string]any)
	if !ok {
		t.Fatalf("responses = %#v, want map", components["responses"])
	}
	if _, ok := responses["ErrorResponse"]; !ok {
		t.Fatal("responses missing ErrorResponse")
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("schemas = %#v, want map", components["schemas"])
	}
	for _, schema := range []string{
		"ApiErrorEnvelope",
		"ApiError",
		"ApiFieldError",
		"Pagination",
		"InventoryListResponse",
		"SnapshotListResponse",
		"MigrationListResponse",
		"MigrationCommandResponse",
		"MigrationExecutionRequest",
		"PilotWorkspace",
		"WorkspaceJob",
	} {
		if _, ok := schemas[schema]; !ok {
			t.Fatalf("schemas missing %s", schema)
		}
	}
}
