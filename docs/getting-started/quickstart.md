# Quickstart

This quickstart uses the local KVM fixture lab so you can evaluate Viaduct end to end without a live hypervisor. The fastest route from clone to a working dashboard is the checked-in Docker Compose stack. The repo-local current release/install reference for `v3.3.0` lives in [../releases/current.md](../releases/current.md).

The default dashboard path is assessment-first: start Viaduct, open the browser, create an assessment, discover, inspect, simulate, save a plan, and export a report.

If you are deploying rather than building from source, start with [../operations/docker.md](../operations/docker.md).

## Prerequisites
- Docker with Docker Compose
- `make` only if you want the convenience targets

## 1. Start Viaduct

```bash
docker compose up -d --build
```

Or:

```bash
make local-up
```

The local Compose stack starts PostgreSQL, persists state in Docker volumes, publishes Viaduct on `127.0.0.1:8080`, and uses `deploy/local/config.yaml`. That config contains no tenant, service-account, or admin keys.

## 2. Open The Dashboard

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). The local runtime serves the dashboard at `/` and the API under `/api/v1/`.
Live operator API docs are also available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

For the default local Docker path, the dashboard starts a local browser session automatically. There is no key to paste.

If you want the Vite development server while editing frontend code:

```bash
cd web
npm ci
npm run dev
```

## 3. Run The Assessment Workflow

1. Create the first assessment from the prefilled lab defaults.
2. Run discovery to save assessment snapshots.
3. Inspect the workload table and dependency graph.
4. Run readiness simulation.
5. Save the migration plan.
6. Export the assessment report.

The seeded API request body for the same intake is available in `examples/lab/pilot-workspace-create.json`.

## 4. Runtime Checks

```bash
docker compose ps
docker compose logs -f viaduct
```

Stop the local stack with `docker compose down`. Add `-v` only when you want to delete the local PostgreSQL data volume.

If you want the real browser smoke from the repo root before a demo or release review:

```bash
make web-e2e-setup
make pilot-smoke
```

## 5. Optional Native Source Path

If you want to run the binary directly instead of Docker:

```bash
make build
make web-build
./bin/viaduct start
```

On Windows without `make`:

```powershell
go mod tidy
go build -o bin/viaduct.exe ./cmd/viaduct
npm --prefix web ci
npm --prefix web run build
.\bin\viaduct.exe start
```

On a fresh source checkout, `viaduct start` creates `~/.viaduct/config.yaml` if it is missing and points it at `examples/lab/kvm`.

## 6. Optional CLI Corroboration

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

This validates the same local fixture set through the CLI.

## Next Steps
- Installation details: [installation.md](installation.md)
- Assessment workflow guide: [../operations/pilot-workspace-flow.md](../operations/pilot-workspace-flow.md)
- Configuration reference: [../reference/configuration.md](../reference/configuration.md)
- Migration operations guide: [../operations/migration-operations.md](../operations/migration-operations.md)
- Auth, role, and auditability model: [../operations/auth-role-audit-model.md](../operations/auth-role-audit-model.md)
- Ship-readiness plan: [../operations/ship-readiness-plan.md](../operations/ship-readiness-plan.md)
- Lab assets: [../../examples/lab/README.md](../../examples/lab/README.md)
