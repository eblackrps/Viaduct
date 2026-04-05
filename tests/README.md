# Tests

This directory contains cross-package test assets that do not fit cleanly inside a single package.

## Layout
- `integration/`: end-to-end and integration coverage for discovery, migration, lifecycle, tenant isolation, plugins, packaging, and release-readiness workflows

## Expectations
- Default package tests should remain race-safe.
- Integration tests should favor realistic fixtures and operator-facing flows over brittle internal snapshots.
- Release-affecting changes should keep `make release-gate` green.
