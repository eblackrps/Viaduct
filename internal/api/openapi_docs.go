package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"gopkg.in/yaml.v3"
)

const (
	openAPIReferencePath = "docs/reference/openapi.yaml"
	swaggerJSONPath      = "docs/swagger.json"
)

func swaggerUIHandler() http.Handler {
	return httpSwagger.Handler(
		httpSwagger.URL("/api/v1/docs/swagger.json"),
		httpSwagger.DocExpansion("list"),
		httpSwagger.PersistAuthorization(true),
	)
}

func (s *Server) handleOpenAPIDocsRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}
	http.Redirect(w, r, "/api/v1/docs/index.html", http.StatusPermanentRedirect)
}

func (s *Server) handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "invalid_request", "method not allowed", apiErrorOptions{})
		return
	}

	payload, err := loadSwaggerJSON()
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "internal_error", err.Error(), apiErrorOptions{Retryable: true})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(payload); err != nil {
		packageLogger.Error(
			"failed to write swagger json response",
			"request_id", responseRequestID(nil, r),
			"error", err.Error(),
		)
	}
}

func loadSwaggerJSON() ([]byte, error) {
	jsonPath := resolveOperatorPath(swaggerJSONPath)
	if payload, err := os.ReadFile(jsonPath); err == nil {
		return payload, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("load swagger json %s: %w", jsonPath, err)
	}

	yamlPath := resolveOperatorPath(openAPIReferencePath)
	payload, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI reference %s: %w", yamlPath, err)
	}

	var document any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode OpenAPI reference %s: %w", filepath.Clean(yamlPath), err)
	}

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode OpenAPI json %s: %w", filepath.Clean(yamlPath), err)
	}
	return append(encoded, '\n'), nil
}
