package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	var (
		inputPath  string
		outputPath string
	)

	flag.StringVar(&inputPath, "input", filepath.Join("docs", "reference", "openapi.yaml"), "Path to the canonical OpenAPI YAML document")
	flag.StringVar(&outputPath, "output", filepath.Join("docs", "swagger.json"), "Path to the generated Swagger JSON document")
	flag.Parse()

	if err := generateSwaggerJSON(inputPath, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "openapi-generate: %v\n", err)
		os.Exit(1)
	}
}

func generateSwaggerJSON(inputPath, outputPath string) error {
	// #nosec G304 -- the generator reads the explicit canonical OpenAPI document requested by the release workflow.
	payload, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	var document any
	if err := yaml.Unmarshal(payload, &document); err != nil {
		return fmt.Errorf("decode %s: %w", inputPath, err)
	}

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", inputPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(outputPath), err)
	}
	if err := os.WriteFile(outputPath, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	return nil
}
