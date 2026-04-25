# Rollback Guide

This guide covers release rollback and migration rollback. They are related but distinct operations.

## 1. Release Rollback

Use this when a newly deployed Viaduct binary or dashboard bundle causes operator-impacting regressions.

### Steps
1. Stop the new Viaduct process.
2. Reinstall the previous known-good binary and matching web assets.
3. Restore the previous config only if the new version required config changes.
4. Start Viaduct and verify:
   - `viaduct version`
   - `GET /healthz` and `GET /readyz`
   - tenant auth and inventory reads

### PostgreSQL Note
Viaduct's schema updates are additive and compatibility-oriented, but you should still take a database backup before upgrades. If an upgrade introduced an unwanted schema state, restore from your database backup rather than editing tables manually.

## 2. Migration Rollback

Use this when a workload migration run must be reverted to its pre-cutover or pre-import state.

### CLI

```bash
viaduct rollback --migration <migration-id>
```

### API

```bash
curl -X POST \
  -H "X-API-Key: <tenant-key>" \
  http://localhost:8080/api/v1/migrations/<migration-id>/rollback
```

### What Viaduct Rolls Back
- target VM removal when the target connector supports it
- cleanup of converted disk artifacts
- restoration of source VM power state when the source connector supports it
- persisted migration state updates and rollback diagnostics

### What To Check After Rollback
- the migration record is marked `rolled_back` or remains `failed` with actionable rollback errors
- source workloads are reachable and in the expected power state
- target artifacts are removed or explicitly listed in the rollback errors

If rollback reports errors, treat the migration as incomplete and investigate the returned diagnostics before retrying.
