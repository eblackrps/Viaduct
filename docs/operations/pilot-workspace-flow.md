# Pilot Workspace Flow

This guide documents the first-class operator flow Viaduct ships for workspace-first migration assessment, planning, and supervised pilot execution.

A pilot workspace is the persisted assessment record that ties together:
- source connections and credential references
- discovery snapshots
- dependency graph output
- target assumptions
- simulation and readiness results
- saved migration plans
- approvals, notes, and exported reports

Use this workflow when you want one durable operator-owned document instead of bouncing between disconnected discovery, graph, simulation, and reporting surfaces.

## Recommended Backends

- Local evaluation and demos: in-memory store is acceptable and keeps the `examples/lab` flow fast.
- Any serious pilot: PostgreSQL is the recommended backend so workspace state, jobs, approvals, and reports persist across restarts.

## Deterministic Lab Bootstrap

This path is the default local evaluation route from a fresh clone.

## 1. Build And Start The API

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
make build
```

Start the API with an admin bootstrap key:

```bash
export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

On Windows PowerShell:

```powershell
$env:VIADUCT_ADMIN_KEY = "lab-admin"
.\bin\viaduct.exe serve-api --port 8080
```

## 2. Create The Lab Tenant

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants
```

This seeds a deterministic tenant key of `lab-tenant-key`.

## 3. Create The Dashboard Service Account

```bash
curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://localhost:8080/api/v1/service-accounts
```

This seeds a deterministic service-account key of `lab-operator-key`.

Use the tenant key only for bootstrap or break-glass admin actions. Use the service-account key for the normal dashboard and pilot flow.

## 4. Start The Dashboard

```bash
cd web
npm ci
npm run dev
```

Open the Vite URL shown in the terminal. The dashboard now starts on the pilot workspace route and presents a runtime authentication bootstrap screen.

Authenticate in one of two ways:
- Preferred: choose `Service account` and paste `lab-operator-key`
- Bootstrap only: choose `Tenant key` and paste `lab-tenant-key`

The dashboard stores the runtime key in session storage by default. Use the optional remember toggle only on a trusted browser that should keep the key across restarts.

You can still pre-seed development credentials with:
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`
- `VITE_VIADUCT_API_KEY`

The runtime bootstrap flow is the canonical operator path because it works for packaged environments and does not require a rebuild to rotate credentials.

The API accepts browser requests from the default local dashboard origins. If you serve the dashboard from another host or port, set `VIADUCT_ALLOWED_ORIGINS` before starting the API.

## 5. Run The Workspace-First Operator Flow

Inside the dashboard:

1. Create a workspace from the prefilled lab defaults.
2. Run discovery to save workspace snapshots.
3. Inspect the workload table, dependency graph, and selected workload set.
4. Run simulation to derive readiness and recommendations.
5. Save the plan to persist a dry-run migration record.
6. Export the pilot report.

The workspace keeps the discovery baseline, readiness result, saved plan, notes, approvals, report history, and job correlation detail attached to the same object.

Read-only operators can inspect workspace state and export reports with viewer access, but only operator-level principals can mutate workspace state or start jobs.

## API Equivalents

If you want to exercise the same flow through the REST API, the seeded request body below matches the default dashboard intake:

```bash
curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/pilot-workspace-create.json \
  http://localhost:8080/api/v1/workspaces
```

Then progress the workspace with persisted background jobs:

```bash
curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"discovery"}' \
  http://localhost:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"graph"}' \
  http://localhost:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"simulation"}' \
  http://localhost:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"plan"}' \
  http://localhost:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"format":"markdown"}' \
  http://localhost:8080/api/v1/workspaces/<workspace-id>/reports/export
```

## Error Handling Expectations

The workspace flow is correlation-aware:
- job records persist the originating `correlation_id`
- API errors include a request ID and retryability signal
- dashboard error panels expose technical details instead of flattening failures into generic toasts

If a step fails, capture the request ID and workspace/job identifier together. That is the intended operator handoff bundle for troubleshooting.

Queued or running workspace jobs are recovered when the API starts again, and each job is subject to the server-side timeout configured by `VIADUCT_WORKSPACE_JOB_TIMEOUT`.

If you want to discard a completed evaluation workspace, delete it through the dashboard or `DELETE /api/v1/workspaces/{workspaceID}`. That removes the workspace record and its job history, but it does not purge persisted snapshots or migration records outside the workspace document.

## Smoke Coverage

The deterministic end-to-end lab smoke now lives in:
- `tests/integration/pilot_workspace_smoke_test.go`

Run the focused smoke when you want a tight workspace regression pass:

```bash
go test ./tests/integration -run PilotWorkspace_LabFlow_CreateDiscoverGraphSimulatePlanReport_Expected -v
```

For release work, `make release-gate` remains the canonical verification path.
