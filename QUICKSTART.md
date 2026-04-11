# Quickstart

This is the fastest source-based evaluation path for Viaduct. It uses the local lab so you can reach the workspace-to-report flow without a live hypervisor estate.

## Maturity Note

Viaduct is evaluation-ready. Start in the lab or a supervised pilot first.

## 1. Build The CLI

```bash
go mod tidy
make build
./bin/viaduct version
```

On Windows PowerShell:

```powershell
go mod tidy
make build
.\bin\viaduct.exe version
```

## 2. Seed The Local Config

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
```

The lab config uses the local KVM fixture set. For any persistent non-demo environment, configure `state_store_dsn` and use PostgreSQL.

## 3. Start The API

```bash
export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

On Windows PowerShell:

```powershell
$env:VIADUCT_ADMIN_KEY = "lab-admin"
.\bin\viaduct.exe serve-api --port 8080
```

The API accepts browser requests from the default local dashboard origins (`http://localhost:5173`, `http://127.0.0.1:5173`, `http://localhost:4173`, `http://127.0.0.1:4173`). If you serve the dashboard from a different origin, set `VIADUCT_ALLOWED_ORIGINS` before starting the API.

## 4. Seed The Lab Tenant And Service Account

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

This creates:
- tenant key: `lab-tenant-key`
- service-account key: `lab-operator-key`

Use the service-account key for the normal operator flow. Keep the tenant key for bootstrap or break-glass admin work.

## 5. Start The Dashboard

```bash
cd web
npm ci
npm run dev
```

Open the Vite URL shown in the terminal. The dashboard starts on the pilot workspace route and asks for a runtime key.

Authenticate with:
- preferred: `lab-operator-key`
- bootstrap only: `lab-tenant-key`

The dashboard stores the runtime key in session storage by default. Use the remember option only when you intentionally want the browser to keep a local copy across restarts.

## 6. Run The Workspace-First Flow

In the dashboard:

1. Create the first pilot workspace from the prefilled lab defaults.
2. Run discovery.
3. Inspect the workload and graph state.
4. Run readiness simulation.
5. Save the plan.
6. Export the pilot report.

The matching seeded request body for API-driven creation is in [examples/lab/pilot-workspace-create.json](examples/lab/pilot-workspace-create.json).

## 7. Optional CLI Corroboration

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

## Next Steps

- Detailed quickstart: [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md)
- Pilot workspace guide: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)
- Installation guide: [INSTALL.md](INSTALL.md)
- Configuration reference: [docs/reference/configuration.md](docs/reference/configuration.md)
- Deployment examples: [examples/deploy/README.md](examples/deploy/README.md)
