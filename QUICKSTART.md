# Quickstart

This is the fastest source-based evaluation path for Viaduct. It uses the local lab so you can reach the workspace-to-report flow without a live hypervisor estate.

The default operator path is now browser-first: start the local runtime, open the WebUI, create a workspace, discover, inspect, simulate, save a plan, and export a report.

## 1. Build Viaduct

```bash
go mod tidy
make build
make web-build
./bin/viaduct version
./bin/viaduct start
```

On Windows PowerShell:

```powershell
go mod tidy
make build
make web-build
.\bin\viaduct.exe version
.\bin\viaduct.exe start
```

On a fresh source checkout, `viaduct start` writes `~/.viaduct/config.yaml` automatically if it does not exist and points it at the shipped `examples/lab/kvm` fixtures.

For any persistent non-demo environment, configure `state_store_dsn` and use PostgreSQL instead of the in-memory store.

## 2. Open The WebUI

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). The same local runtime serves the WebUI at `/` and the API at `/api/v1/`.
Live Swagger UI is also available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

For the default local lab path, the dashboard can use the built-in single-user fallback and does not require a pasted browser key.

If you intentionally configure tenant keys or service-account keys, the runtime bootstrap screen remains available. The runtime auth flow now creates a server-backed session: the browser keeps only a non-sensitive session marker, and the API credential itself stays in an `httpOnly` cookie. Use the remember option only on a trusted workstation.

If you want the Vite development server instead of the packaged local shell:

```bash
cd web
npm ci
npm run dev
```

Use that only for frontend development. The default operator path is the same-origin dashboard served by `viaduct start`.

## 3. Run The Workspace-First Flow

In the dashboard:

1. Create the first pilot workspace from the prefilled lab defaults.
2. Run discovery.
3. Inspect the workload and graph state.
4. Run readiness simulation.
5. Save the plan.
6. Export the pilot report.

The matching seeded request body for API-driven creation is in [examples/lab/pilot-workspace-create.json](examples/lab/pilot-workspace-create.json).

## 4. Check The Local Runtime

```bash
./bin/viaduct status --runtime
./bin/viaduct doctor
```

Stop the local runtime when you are finished:

```bash
./bin/viaduct stop
```

## 5. Optional CLI Corroboration

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
