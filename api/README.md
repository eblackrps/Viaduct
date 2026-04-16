# API Assets

This directory contains API-related assets that are shared across Viaduct components.

## Contents
- `proto/plugin.proto`: gRPC contract for community connector plugins

## Public Contract Surfaces

- Checked-in OpenAPI contract: [`../docs/reference/openapi.yaml`](../docs/reference/openapi.yaml)
- Runtime Swagger UI: `/api/v1/docs`
- Runtime Swagger JSON: `/api/v1/docs/swagger.json`
- Build metadata endpoint: `/api/v1/about`
- Health and connector-circuit state: `/api/v1/health`

## Notes
- The plugin host in `internal/connectors/plugin/` treats this proto as the compatibility contract for external connector processes.
- Changes here are compatibility-sensitive and should be coordinated with plugin documentation, published OpenAPI docs, and regression coverage.
