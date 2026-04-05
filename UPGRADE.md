# Upgrade

This document is the top-level upgrade entrypoint. For the detailed operational version, see [docs/operations/upgrade.md](docs/operations/upgrade.md). Pair it with [docs/operations/rollback.md](docs/operations/rollback.md) before changing a production-like environment.

## Recommended Upgrade Flow
1. Record the current version with `viaduct version`.
2. Back up your config and database state before replacing binaries.
3. Build or unpack the new release bundle.
4. Compare your config against [configs/config.example.yaml](configs/config.example.yaml) for new optional fields.
5. Start the new binary and verify health, auth, inventory reads, and any active dashboard/API flows.
6. Run `make release-gate` in a staging or pre-release environment when possible.

## Upgrade Notes
- Viaduct's PostgreSQL backend handles additive schema initialization and compatibility updates on startup.
- Keep tenant-scoped migration and recovery-point data intact; do not manually rewrite tenant keys.
- If you have active migrations, finish or intentionally pause them before upgrading.

## Rollback
If an upgrade regresses operator workflows, follow [docs/operations/rollback.md](docs/operations/rollback.md) and reinstall the last known-good binary and matching web assets.
