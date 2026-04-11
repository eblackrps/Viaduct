# Migration Operations Guide

This guide covers the operator path for planning, validating, executing, resuming, and rolling back migrations.

For the single primary end-to-end path Viaduct is currently hardening, see [Primary Reliability Path](primary-reliability-path.md).

For operator and maintainer debugging expectations across this workflow, see [Observability Requirements](observability-requirements.md).

For the workspace-first assessment flow that now precedes saved-plan handoff, see [Pilot Workspace Flow](pilot-workspace-flow.md).

## 1. Prepare Inventory

Collect discovery data for your source and target platforms first.

```bash
viaduct discover --type vmware --source https://vcenter.example.local --save
viaduct discover --type proxmox --source https://proxmox.example.local:8006/api2/json --save
```

For local evaluation, use the KVM fixture lab:

```bash
viaduct discover --type kvm --source examples/lab/kvm --save
```

## 2. Create Or Review A Migration Spec

Start from:
- `configs/example-migration.yaml`
- `configs/example-migration-minimal.yaml`
- `examples/lab/migration-window.yaml`

Validate the spec:

```bash
viaduct plan --spec examples/lab/migration-window.yaml
```

If you are operating through the dashboard, this same planning intent can now start from a pilot workspace. The workspace persists source connections, readiness state, and the saved dry-run migration plan before you move into explicit execution routes.

## 3. Run Preflight

Preflight is exposed through the API and the dashboard migration wizard. It validates:
- source and target connectivity
- execution windows
- approval gates
- disk space
- network mappings
- naming conflicts
- backup/snapshot presence
- disk format compatibility
- target resource availability
- rollback readiness
- wave-based execution planning

## 4. Execute

CLI:

```bash
viaduct migrate --plan configs/example-migration.yaml
```

Dry run:

```bash
viaduct migrate --plan configs/example-migration.yaml --dry-run
```

API create + execute flow:

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <tenant-key>" \
  --data @spec.json \
  http://localhost:8080/api/v1/migrations

curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <tenant-key>" \
  --data '{"approved_by":"operator","ticket":"CHG-1234"}' \
  http://localhost:8080/api/v1/migrations/<migration-id>/execute
```

## 5. Resume Long-Running Runs

Use resume when a migration stopped after persisting checkpoints.

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <tenant-key>" \
  --data '{"approved_by":"operator","ticket":"CHG-1234"}' \
  http://localhost:8080/api/v1/migrations/<migration-id>/resume
```

Resume support is checkpoint-aware and should skip already completed phases.

## 6. Monitor Progress

Use:
- `viaduct status`
- `GET /api/v1/migrations`
- `GET /api/v1/migrations/<migration-id>`
- the dashboard migration history and progress views

Look for:
- current phase
- per-workload failures
- approval state
- execution window issues
- checkpoint diagnostics

## 7. Roll Back If Needed

```bash
viaduct rollback --migration <migration-id>
```

Or:

```bash
curl -X POST \
  -H "X-API-Key: <tenant-key>" \
  http://localhost:8080/api/v1/migrations/<migration-id>/rollback
```

Rollback correctness is critical. If the result includes errors, treat the run as failed and investigate before retrying.
