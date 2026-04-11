# Quickstart

This quickstart uses the local KVM fixture lab so you can evaluate Viaduct end to end without a live hypervisor.

The default dashboard path is now workspace-first and authenticated at runtime with a service-account key or tenant key.

## Prerequisites
- Go 1.24+
- Node.js 20.19+
- `make` if you want the convenience targets

## 1. Build Viaduct

```bash
make build
make web-build
./bin/viaduct version
```

On Windows without `make`:

```powershell
go mod tidy
go build -o bin/viaduct.exe ./cmd/viaduct
npm --prefix web ci
npm --prefix web run build
.\bin\viaduct.exe version
```

## 2. Create A Local Config

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
```

For the local lab, the config only needs the built-in KVM fixture path. For a persistent pilot environment, configure `state_store_dsn` and use PostgreSQL.

## 3. Start The API And Seed Auth

```bash
export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

The same process now serves the built dashboard at [http://localhost:8080](http://localhost:8080) when assets are present. For any other dashboard origin, set `VIADUCT_ALLOWED_ORIGINS` before starting the API.

In another terminal, create the lab tenant and operator service account:

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants

curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://localhost:8080/api/v1/service-accounts
```

## 4. Open The Dashboard

Open [http://localhost:8080](http://localhost:8080). The dashboard opens on the pilot workspace route and calls the same origin API.

Authenticate through the runtime bootstrap screen:
- preferred service-account key: `lab-operator-key`
- bootstrap-only tenant key: `lab-tenant-key`

The runtime key is kept in session storage by default. Use the remember option only when you intentionally want the browser to keep a local copy across restarts.

For packaged or persistent environments, prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` over `VITE_VIADUCT_API_KEY` only when you intentionally pre-seed a development build. The runtime bootstrap is the default operator path.

If you want the Vite development server while editing frontend code:

```bash
cd web
npm ci
npm run dev
```

## 5. Run The Workspace-First Operator Flow

1. Create the first pilot workspace from the prefilled lab defaults.
2. Run discovery to save workspace snapshots.
3. Inspect the workload table and dependency graph.
4. Run readiness simulation.
5. Save the migration plan.
6. Export the pilot report.

The seeded API request body for the same intake is available in `examples/lab/pilot-workspace-create.json`.

## 6. Optional CLI Corroboration

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
