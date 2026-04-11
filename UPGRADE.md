# Upgrade

This is the top-level upgrade entrypoint. Use it with [docs/operations/upgrade.md](docs/operations/upgrade.md) and [docs/operations/rollback.md](docs/operations/rollback.md) before changing any persistent or pilot environment.

## Recommended Upgrade Flow

1. Record the current version with `viaduct version`.
2. Back up your config and database state before replacing binaries.
3. Build or unpack the new release bundle.
4. Compare your config against [configs/config.example.yaml](configs/config.example.yaml) for new optional fields.
5. Start the new binary and verify health, auth, inventory reads, and the dashboard shell at `/` when bundled web assets are present.
6. Run `make release-gate` in a staging or pre-release environment when practical.

## Upgrade Notes

- Viaduct's PostgreSQL backend handles additive schema initialization and compatibility updates on startup.
- Keep tenant-scoped migration and recovery-point data intact; do not manually rewrite tenant keys.
- Validate `/api/v1/about` after startup so the reported store backend and schema version match the expected deployment state.
- If active migrations exist, finish or intentionally pause them before upgrading.
- If you rely on packaged web assets, redeploy the matching dashboard bundle together with the binary.
- If you use the standard local operator path, prefer `viaduct start` after upgrade. Keep `viaduct serve-api` for service, container, or intentionally headless deployments.

## Rollback

If an upgrade regresses operator workflows, follow [docs/operations/rollback.md](docs/operations/rollback.md) and reinstall the last known-good binary with matching web assets.
