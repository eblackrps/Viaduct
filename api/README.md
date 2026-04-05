# API Assets

This directory contains API-related assets that are shared across Viaduct components.

## Contents
- `proto/plugin.proto`: gRPC contract for community connector plugins

## Notes
- The plugin host in `internal/connectors/plugin/` treats this proto as the compatibility contract for external connector processes.
- Changes here are compatibility-sensitive and should be coordinated with plugin documentation and regression coverage.
