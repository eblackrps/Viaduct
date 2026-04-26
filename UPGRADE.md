# Upgrade

This is the top-level upgrade entrypoint. Use it with [docs/operations/upgrade.md](docs/operations/upgrade.md) and [docs/operations/rollback.md](docs/operations/rollback.md) before changing any persistent or pilot environment.

## Recommended Upgrade Flow

1. Record the current version with `viaduct version`.
2. Back up your config and database state before replacing the running deployment.
3. If you deploy with Docker or Kubernetes, pull the new immutable semver image, verify its cosign signature, and update your Compose or Helm tag while preserving `/var/lib/viaduct` and your config mount.
4. If you rely on native bundles or source builds, build or unpack the new release bundle and replace the binary plus matching web assets together.
5. Compare your config against [configs/config.example.yaml](configs/config.example.yaml) for new optional fields.
6. Start the new runtime and verify health, auth, inventory reads, and the dashboard shell at `/` when bundled web assets are present.
7. Run `make release-gate` in a staging or pre-release environment when practical.

## Upgrade Notes

- Viaduct's PostgreSQL backend handles additive schema initialization and compatibility updates on startup.
- Keep tenant-scoped migration and recovery-point data intact; do not manually rewrite tenant keys.
- Validate `/api/v1/about` after startup so the reported store backend and schema version match the expected deployment state.
- Validate `/api/v1/docs` after startup if you ship automation or integrations from the checked-in API contract.
- If active migrations exist, finish or intentionally pause them before upgrading.
- If you rely on packaged web assets, redeploy the matching dashboard bundle together with the binary.
- If you use the standard local dashboard path, prefer `viaduct start` after upgrade. Keep `viaduct serve-api` for service, container, or intentionally headless deployments.
- The loopback local session path now requires a direct `127.0.0.1` browser request. If you place the runtime behind a same-host reverse proxy or use any non-loopback hostname, authenticate with a tenant or service account key instead of expecting `Start local session` to appear.
- If `VIADUCT_ALLOWED_ORIGINS` was empty in older environments, note that the effective default is now same-origin only. Set it explicitly only for trusted cross-origin dashboards.
- The dashboard runtime auth path now uses server-backed sessions with an `httpOnly` cookie. If you are testing across older browser state, sign out once after upgrade so stale browser session state is cleared cleanly.
- If an upgraded PostgreSQL environment fails during credential migration with a duplicate-credential conflict, resolve any reused tenant or service account API keys so every persisted credential is globally unique, then restart the new binary.
- New clients can adopt `/api/v2/inventory`, `/api/v2/snapshots`, and `/api/v2/migrations` for paginated list responses. Legacy `/api/v1` response shapes remain available for existing clients.

## Rollback

If an upgrade regresses workflows, follow [docs/operations/rollback.md](docs/operations/rollback.md) and roll back to the last known-good image tag or reinstall the last known-good native bundle with matching web assets.
