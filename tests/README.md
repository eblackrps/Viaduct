# Tests

This directory contains cross-package test assets that do not fit cleanly inside a single package.

## Layout

- `integration/`: end-to-end and integration coverage for discovery, planning, migration, lifecycle, tenant isolation, plugins, packaging, and workspace smoke flows
- `certification/`: fixture-backed connector certification coverage for stable normalization behavior
- `soak/`: tagged longer-running migration scaling tests used by `make soak-test`

## Expectations

- Default package tests should remain race-safe.
- Integration tests should favor realistic fixtures and operator-visible flows over brittle internal snapshots.
- Public-facing changes should keep `make release-gate` green.
