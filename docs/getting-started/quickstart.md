# Quickstart

This quickstart uses the local KVM fixture lab so you can evaluate Viaduct end to end without a live hypervisor.

The default dashboard path is now WebUI-first and workspace-first: `viaduct start`, open the browser, create a workspace, discover, inspect, simulate, save a plan, and export a report.

## Prerequisites
- Go 1.24+
- Node.js 20.19+
- `make` if you want the convenience targets

## 1. Build Viaduct

```bash
make build
make web-build
./bin/viaduct version
./bin/viaduct start
```

On Windows without `make`:

```powershell
go mod tidy
go build -o bin/viaduct.exe ./cmd/viaduct
npm --prefix web ci
npm --prefix web run build
.\bin\viaduct.exe version
.\bin\viaduct.exe start
```

On a fresh source checkout, `viaduct start` creates `~/.viaduct/config.yaml` if it is missing and points it at `examples/lab/kvm`. For a persistent pilot environment, configure `state_store_dsn` and use PostgreSQL instead of the in-memory store.

## 2. Open The Dashboard

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). The local runtime serves the dashboard at `/` and the API under `/api/v1/`.
Live operator API docs are also available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

For the default local lab path, the dashboard can use the built-in single-user fallback and does not require a pasted browser key.

If you intentionally configure a tenant key or service-account key, the runtime bootstrap screen remains available:
- preferred service-account key: `lab-operator-key`
- bootstrap-only tenant key: `lab-tenant-key`

The runtime auth flow creates a server-backed session. The browser keeps only an opaque session marker, and the API credential itself remains in an `httpOnly` cookie. Use the remember option only when you intentionally want that marker kept across restarts on a trusted workstation.

For packaged or persistent environments, prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` over `VITE_VIADUCT_API_KEY` only when you intentionally pre-seed a development build. The runtime bootstrap is the default operator path.

If you want the Vite development server while editing frontend code:

```bash
cd web
npm ci
npm run dev
```

## 3. Run The Workspace-First Operator Flow

1. Create the first pilot workspace from the prefilled lab defaults.
2. Run discovery to save workspace snapshots.
3. Inspect the workload table and dependency graph.
4. Run readiness simulation.
5. Save the migration plan.
6. Export the pilot report.

The seeded API request body for the same intake is available in `examples/lab/pilot-workspace-create.json`.

## 4. Runtime Checks

```bash
./bin/viaduct status --runtime
./bin/viaduct doctor
```

Stop the local runtime when you are done:

```bash
./bin/viaduct stop
```

## 5. Optional CLI Corroboration

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

This validates the same local fixture set through the CLI.

## Next Steps
- Installation details: [installation.md](installation.md)
- Pilot workspace guide: [../operations/pilot-workspace-flow.md](../operations/pilot-workspace-flow.md)
- Configuration reference: [../reference/configuration.md](../reference/configuration.md)
- Migration operations guide: [../operations/migration-operations.md](../operations/migration-operations.md)
- Auth, role, and auditability model: [../operations/auth-role-audit-model.md](../operations/auth-role-audit-model.md)
- Lab assets: [../../examples/lab/README.md](../../examples/lab/README.md)
