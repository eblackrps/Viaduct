# Upgrade Guide

This guide covers upgrading the Viaduct binary, dashboard assets, and persisted state.

## Before You Upgrade
- Record the current version with `viaduct version`
- Back up your config file and any local secrets
- If you use PostgreSQL, take a database backup or snapshot before replacing the binary
- If you have an active migration, confirm whether it is safe to finish or explicitly pause before upgrading

## Binary Upgrade

From source:

```bash
git pull
go mod tidy
make build
```

From a packaged release bundle:
1. Unpack the new bundle.
2. Verify `release-manifest.json` and `SHA256SUMS.txt`.
3. Replace the existing binary with the new one.
4. Replace packaged web assets if you serve the dashboard statically.

## Config Compatibility
- Keep your existing `~/.viaduct/config.yaml`.
- Compare it against `configs/config.example.yaml` from the new release for any new optional fields.
- Prefer additive configuration changes over carrying forward deprecated local forks.

## PostgreSQL Store Upgrade
- Viaduct's PostgreSQL backend runs schema creation and compatibility updates automatically on startup.
- The migrations table and recovery point table are tenant-scoped; do not manually collapse keys back to single-column identifiers.
- After upgrade, start Viaduct and verify a successful database connection before exposing the API to operators.

## Post-Upgrade Checks
- `viaduct version`
- `viaduct --help`
- `viaduct plan --spec examples/lab/migration-window.yaml`
- API health: `GET /api/v1/health`
- Dashboard build or static asset load, if applicable

## Recommended Release Validation

```bash
make release-gate
```

This keeps local upgrade validation aligned with CI.
