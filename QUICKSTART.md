# Quickstart

This is the fastest way to evaluate Viaduct from source without a live hypervisor.

The default first-run path is now the workspace-first operator flow:

1. bootstrap the local lab tenant and service account
2. sign into the dashboard with a runtime key
3. create a pilot workspace
4. discover, inspect, simulate, save a plan, and export a report

## 1. Build The CLI

```bash
go mod tidy
make build
./bin/viaduct version
```

## 2. Seed A Local Config

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
```

The lab config points the KVM source at the local fixture set. For any non-demo environment, configure `state_store_dsn` and use PostgreSQL for persistence.

## 3. Start The API With An Admin Bootstrap Key

```bash
export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

On Windows PowerShell:

```powershell
$env:VIADUCT_ADMIN_KEY = "lab-admin"
.\bin\viaduct.exe serve-api --port 8080
```

## 4. Create The Lab Tenant

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants
```

This seeds the deterministic tenant key `lab-tenant-key`.

## 5. Create The Lab Service Account

```bash
curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://localhost:8080/api/v1/service-accounts
```

This seeds the deterministic service-account key `lab-operator-key`.

## 6. Start The Dashboard

```bash
cd web
npm ci
npm run dev
```

The dashboard expects the API at `/api` and opens on the pilot workspace route. The first screen is a runtime auth bootstrap flow.

Authenticate with:
- `Service account`: `lab-operator-key`
- `Tenant key`: `lab-tenant-key`

For normal dashboard and pilot use, prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`. Keep `VITE_VIADUCT_API_KEY` for tenant bootstrap or break-glass admin access.

If you create additional tenants or service accounts, verify the active identity with:

```bash
curl -H "X-Service-Account-Key: <service-account-key>" http://localhost:8080/api/v1/tenants/current
```

Use the tenant key only when you are bootstrapping the tenant or intentionally using tenant-admin access:

```bash
curl -H "X-API-Key: <tenant-key>" http://localhost:8080/api/v1/tenants/current
```

## 7. Run The Workspace-First Flow

In the dashboard:

1. Create the first pilot workspace from the prefilled lab defaults.
2. Run discovery.
3. Inspect the discovered workloads and dependency graph.
4. Run simulation.
5. Save the plan.
6. Export the pilot report.

The seeded API equivalent for workspace creation lives in [examples/lab/pilot-workspace-create.json](examples/lab/pilot-workspace-create.json).

## 8. Optional CLI Corroboration

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

## Next Steps
- Review [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md) for the full workspace-first operator guide.
- Review [docs/operations/migration-operations.md](docs/operations/migration-operations.md) for execution and rollback workflows.
- Review [docs/operations/backup-portability.md](docs/operations/backup-portability.md) for Veeam portability guidance.
- Review [docs/operations/multi-tenancy.md](docs/operations/multi-tenancy.md) before enabling shared-tenant operation.
- Review [docs/operations/auth-role-audit-model.md](docs/operations/auth-role-audit-model.md) for the early-product trust model and attribution boundaries.
- Review [examples/deploy/README.md](examples/deploy/README.md) if you want to move from local evaluation into a packaged pilot environment.
- Review [examples/plugin-example/README.md](examples/plugin-example/README.md) if you want to validate plugin-backed connector loading.
