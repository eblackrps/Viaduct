# Backend Observability

Viaduct's operator workflows cross HTTP requests, queued background jobs, connector calls, and long-running migration phases. Because of that, traces are the first signal to wire up. They show where time is spent across discovery, planning, simulation, report generation, and migration execution without forcing operators to reconstruct a story from scattered logs.

This repo now includes:
- backend OpenTelemetry tracing for the Go API and migration engine
- a local Grafana + Tempo stack under [`deploy/observability/`](../../deploy/observability)
- a lightweight validation script that checks Grafana, Tempo, and a real Viaduct trace by ID

Loki is intentionally not included in this first pass. Viaduct already emits structured logs through `slog`, but the bigger operator gap here was request-to-background trace continuity. Adding a log stack now would add more moving parts than value for the current evaluator path.

## What Is Instrumented

The backend now emits spans for:
- inbound HTTP requests and response status
- PostgreSQL-backed snapshot, migration, recovery-point, and audit persistence
- workspace discovery, graph, simulation, plan, and report export jobs
- summary, migration, and audit report generation
- async migration execution handoff
- migration discovery and execution phases
- outbound connector HTTP calls for Proxmox, Nutanix, Veeam, and VMware SOAP transport

## Metrics Contract

The existing `/metrics` endpoint remains the official Viaduct metrics contract for now.

OTel metrics export is intentionally deferred so the repo does not end up with two half-adopted metric paths at once. This pass standardizes traces first, while preserving the operator-visible Prometheus-style metrics surface that already exists for request, store, tenant, and queue health.

## Local Stack

Start the local observability stack:

```bash
make observability-up
```

That starts:
- Grafana on [http://127.0.0.1:3000](http://127.0.0.1:3000)
- Tempo query and UI datasource target on [http://127.0.0.1:3200](http://127.0.0.1:3200)
- Tempo OTLP ingest on `127.0.0.1:4317` and `127.0.0.1:4318`

Grafana is provisioned with a `Tempo` datasource automatically. The local default login is `admin` / `admin`.

Stop and clean up the local stack:

```bash
make observability-down
```

## Enable Telemetry In Viaduct

Viaduct telemetry is config-driven and safe to leave off.

```bash
export VIADUCT_OTEL_ENABLED=true
export VIADUCT_OTEL_ENDPOINT=http://127.0.0.1:4318
export VIADUCT_OTEL_SERVICE_NAME=viaduct-api
export VIADUCT_OTEL_ENVIRONMENT=local
export VIADUCT_OTEL_SAMPLER=parentbased_traceidratio
export VIADUCT_OTEL_SAMPLER_ARG=1
```

Then start Viaduct normally:

```bash
make build
./bin/viaduct start
```

If the OTLP exporter cannot be created at startup, Viaduct logs a clear warning and continues without telemetry. If Tempo is down after startup, request handling still continues; only trace export is degraded.

## Validation

Basic validation:

```bash
make observability-validate
```

That script checks:
- Tempo readiness
- Grafana health
- Grafana's provisioned `Tempo` datasource
- one real Viaduct trace ID from the API
- that the same trace can be fetched back from Tempo

By default the validation script exercises `GET /api/v1/about`, which is enough to prove ingestion. If you want a richer tenant-scoped span tree, pass a service account key or tenant key to the script:

```bash
go run ./scripts/observability_validate -service-account-key <key>
```

For the most representative local workflow, run the evaluator path with telemetry enabled:

```bash
make pilot-smoke
```

Then open Grafana Explore, select the `Tempo` datasource, and inspect recent traces for `service.name="viaduct-api"`. The highest-value traces to inspect first are:
- workspace discovery
- workspace simulation
- workspace plan generation
- migration phase traces
- report export traces

GitHub Actions now runs the same smoke at merge time: it boots the local Grafana + Tempo stack from `deploy/observability`, starts `viaduct start` with telemetry enabled, and runs `make observability-validate` to confirm a real trace arrives in Tempo.

## Intentionally Deferred

- Loki log aggregation
- OTel metrics export
- any custom dashboard diagnostics page

Those are good follow-ups once the current trace path is proven useful in real operator runs.
