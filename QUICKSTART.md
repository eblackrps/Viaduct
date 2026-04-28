# Quickstart

The fastest local path is Docker Compose from the repo root. It builds the dashboard and API, starts PostgreSQL, uses the shipped KVM fixtures, and opens the dashboard without tenant keys, service-account keys, or copied secret files. The current release/install reference for `v3.2.1` also lives in [docs/releases/current.md](docs/releases/current.md).

The default path is browser-first: start Viaduct, open the dashboard, create an assessment, discover, inspect, simulate, save a plan, and export a report.

If you are deploying a shared or production environment, use [INSTALL.md](INSTALL.md) and [docs/operations/docker.md](docs/operations/docker.md).

## 1. Start Viaduct

```bash
docker compose up -d --build
```

Or, if you have `make`:

```bash
make local-up
```

The checked-in `compose.yaml` is intentionally local-only. It publishes Viaduct on `127.0.0.1:8080`, persists state in a Docker PostgreSQL volume, and uses [deploy/local/config.yaml](deploy/local/config.yaml), which contains no API keys.

## 2. Open The Dashboard

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). The same local runtime serves the dashboard at `/` and the API at `/api/v1/`.
Live Swagger UI is also available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

For the default local Docker path, the dashboard starts a local browser session automatically. There is no key to paste.

If you want the Vite development server instead of the packaged local shell:

```bash
cd web
npm ci
npm run dev
```

Use that only for frontend development. The default path is the same-origin dashboard served by `viaduct start`.

## 3. Run The Assessment Workflow

In the dashboard:

1. Create the first assessment from the prefilled lab defaults.
2. Run discovery.
3. Inspect the workload and graph state.
4. Run readiness simulation.
5. Save the plan.
6. Export the assessment report.

The matching seeded request body for API-driven creation is in [examples/lab/pilot-workspace-create.json](examples/lab/pilot-workspace-create.json).

## 4. Check Or Stop The Stack

```bash
docker compose ps
docker compose logs -f viaduct
docker compose down
```

If you want the browser half of the evaluator smoke from the repo root:

```bash
make web-e2e-setup
make pilot-smoke
```

## 5. Optional Native Source Path

If you want to run the binary directly instead of Docker:

```bash
go mod tidy
make build
make web-build
./bin/viaduct start
```

On Windows PowerShell:

```powershell
go mod tidy
make build
make web-build
.\bin\viaduct.exe start
```

`viaduct start` writes `~/.viaduct/config.yaml` automatically when it does not exist and points it at the shipped `examples/lab/kvm` fixtures.

## 6. Optional CLI Corroboration

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

## Next Steps

- Detailed quickstart: [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md)
- Assessment guide: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)
- Installation guide: [INSTALL.md](INSTALL.md)
- Configuration reference: [docs/reference/configuration.md](docs/reference/configuration.md)
- Deployment examples: [examples/deploy/README.md](examples/deploy/README.md)
