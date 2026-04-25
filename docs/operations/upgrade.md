# Upgrade Guide

This guide covers upgrading the primary Viaduct container deployment, plus the alternative native-bundle and source-build paths.

## Before You Upgrade
- Record the current version with `viaduct version`
- Back up your config file and any local secrets
- If you use PostgreSQL, take a database backup or snapshot before replacing the binary
- If you have an active migration, confirm whether it is safe to finish or explicitly pause before upgrading

## Docker And Kubernetes Upgrade

1. Pull the new immutable semver image tag.
2. Verify the image signature and SBOM attestation before rollout.
3. Update your Compose file, Helm values, or workload manifest to the new tag.
4. Preserve the mounted config path and writable state volume at `/var/lib/viaduct`.
5. Restart the workload and verify `/healthz`, `/readyz`, `/api/v1/about`, and the dashboard shell.

## Native Bundle Or Source Upgrade

From source:

```bash
git pull
go mod tidy
make build
make web-build
```

From a packaged release bundle:
1. Unpack the new bundle.
2. Verify `release-manifest.json`, `dependency-manifest.json`, and `SHA256SUMS`.
3. Replace the existing binary with the new one.
4. Replace the packaged web assets if you keep a separate installed copy of `share/viaduct/web`.

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
- API readiness: `GET /readyz` or the compatibility alias `GET /api/v1/health`
- Dashboard shell load at `/` when built assets are present

## Recommended Release Validation

```bash
make release-gate
```

This keeps local upgrade validation aligned with CI.
