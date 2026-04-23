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

Viaduct `v3.1.1` treats the signed OCI image as the canonical packaged deployment path. The source-based lab flow below remains the fastest way to evaluate the operator console from a fresh clone.

## Recommended Backends

- Local evaluation and demos: in-memory store is acceptable and keeps the `examples/lab` flow fast.
- Any serious pilot: PostgreSQL is the recommended backend so workspace state, jobs, approvals, and reports persist across restarts.

## Deterministic Lab Bootstrap

This path is the default local evaluation route from a fresh clone.

## 1. Build And Start Viaduct From Source

```bash
make build
make web-build
./bin/viaduct start
```

On Windows PowerShell:

```powershell
.\bin\viaduct.exe start
```

On a fresh source checkout, `viaduct start` creates `~/.viaduct/config.yaml` automatically when it is missing and points it at the shipped `examples/lab/kvm` fixtures.

The local runtime serves the WebUI at [http://127.0.0.1:8080](http://127.0.0.1:8080) and the API under `/api/v1/`.

## 2. Open The Dashboard

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). The dashboard opens on the pilot workspace route and starts on the Get started screen. For this default local lab path, choose `Start local session` from a direct `127.0.0.1` browser request; no pasted browser key is required.
Live operator API docs remain available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs) throughout the flow.

If you intentionally configure key-based access instead, open `Use a key instead` from the same screen:
- `Service account key` for normal operator access
- `Tenant key (advanced)` for tenant setup or break-glass administrative recovery

The dashboard runtime auth flow now creates a server-backed session. The browser stores only an opaque session marker, while any tenant or service account key stays server-side for that session instead of landing in browser storage. Local operator sessions do not use an API key at all. Use the keep-signed-in option only on a trusted browser that should keep that marker across restarts.

You can still pre-seed development credentials with:
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`
- `VITE_VIADUCT_API_KEY`

The Get started flow is the canonical operator path because it works for packaged environments and does not require a rebuild to rotate credentials.

The same-origin local path does not need special CORS configuration. If you serve the dashboard from another host or port, set `VIADUCT_ALLOWED_ORIGINS` before starting the API.

## 3. Run The Workspace-First Operator Flow

Inside the dashboard:

1. Create a workspace from the prefilled lab defaults.
2. Run discovery to save workspace snapshots.
3. Inspect the workload table, dependency graph, and selected workload set.
4. Run simulation to derive readiness and recommendations.
5. Save the plan to persist a dry-run migration record.
6. Export the pilot report.

The workspace keeps the discovery baseline, readiness result, saved plan, notes, approvals, report history, and job correlation detail attached to the same object.

Read-only operators can inspect workspace state and export reports with viewer access, but only operator-level principals can mutate workspace state or start jobs.

## 4. Runtime Checks

If you want CLI confirmation for the local browser-first runtime:

```bash
./bin/viaduct status --runtime
./bin/viaduct doctor
```

Stop the local runtime when you are done:

```bash
./bin/viaduct stop
```

## 5. API Equivalents

If you want to exercise the same flow through the REST API, seed the lab tenant and service account first:

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://127.0.0.1:8080/api/v1/admin/tenants

curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://127.0.0.1:8080/api/v1/service-accounts
```

Then the seeded request body below matches the default dashboard intake:

```bash
curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/pilot-workspace-create.json \
  http://127.0.0.1:8080/api/v1/workspaces
```

Then progress the workspace with persisted background jobs:

```bash
curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"discovery"}' \
  http://127.0.0.1:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"graph"}' \
  http://127.0.0.1:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"simulation"}' \
  http://127.0.0.1:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"type":"plan"}' \
  http://127.0.0.1:8080/api/v1/workspaces/<workspace-id>/jobs

curl -X POST \
  -H "X-Service-Account-Key: lab-operator-key" \
  -H "Content-Type: application/json" \
  --data '{"format":"markdown"}' \
  http://127.0.0.1:8080/api/v1/workspaces/<workspace-id>/reports/export
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
