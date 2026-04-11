# Backend Contract Hardening

This document defines the backend and API contracts that must become stable for Viaduct to be a credible early product, not just a broad technical repository.

It is intentionally scoped to the current beachhead and v1 promise:

**VMware-exit mixed-estate discovery and migration readiness assessment with approval-ready pilot planning**

This is not a request to redesign the API from scratch. The goal is to preserve the current route structure where it is already useful, make the operator contract explicit, and eliminate the mushy edges that currently force the dashboard or operator to guess.

## Contract Hardening Rules

- Keep current route families where possible. Favor additive change over churn.
- Stabilize the packaged dashboard contract before broadening connector or workflow breadth.
- Treat CLI, API, dashboard, OpenAPI, and report exports as one operator contract.
- If a field matters to operator trust or frontend state, it must be documented and tested.
- If a workflow is still pilot-only, the contract must say so without pretending it is generic unattended automation.

## Current Contract Reality

Viaduct already has meaningful backend state and useful route coverage:

- `internal/models/` gives the repo a real shared schema for inventory, tenants, and audit.
- `internal/migrate/` already defines strong internal types for migration specs, plans, checkpoints, preflight reports, and migration state.
- `internal/api/server.go` exposes the core routes the dashboard uses for inventory, graph, summary, preflight, migrations, reports, and tenant context.
- `web/src/` already behaves like a real operator client, which means its assumptions are a reliable signal for where the backend contract is clear or weak.

The weak part is the public operator contract:

- the OpenAPI spec only documents a subset of the routes the dashboard actually depends on
- several important routes are documented as generic `object` responses instead of stable schemas
- some documented request and response behaviors do not match the handler implementation
- many failures still return plain-text `http.Error` payloads instead of structured machine-readable errors
- the dashboard currently derives readiness and workflow state from partial backend data because the backend does not yet expose those states directly
- execute, resume, and rollback API flows depend on an in-memory spec cache in `Server.specs`, which is not a durable product-grade contract

## Immediate OpenAPI Mismatches To Fix First

These are the concrete mismatches already present in the repo today:

- `/api/v1/snapshots`, `/api/v1/graph`, `/api/v1/audit`, `/api/v1/migrations/{id}/execute`, `/api/v1/migrations/{id}/resume`, and `/api/v1/migrations/{id}/rollback` are used by the product surface but are not documented in `docs/reference/openapi.yaml`.
- `/api/v1/preflight`, `POST /api/v1/migrations`, `GET /api/v1/migrations/{id}`, and `/api/v1/reports/{name}` are documented with generic `object` response schemas instead of stable payload schemas.
- the OpenAPI document advertises YAML request bodies for preflight and migration creation, but `decodeSpec` only accepts JSON today.
- `POST /api/v1/migrations` is documented as `201 Created`, but `handleMigrations` currently returns `202 Accepted`.
- `/api/v1/reports/{name}` is documented as returning a generic JSON object, but the real JSON payloads are a summary object, a migration array, or an audit-event array depending on `name`.
- `make contract-check` currently verifies only a small subset of routes and does not validate response schemas, status-code alignment, or enum completeness.

## Contract Audit

| Domain | Current Surfaces | Current State | What Is Clear | What Is Weak Or Missing | Hardening Decision |
| --- | --- | --- | --- | --- | --- |
| Sources | `MigrationSpec.source`, `MigrationSpec.target`, snapshot metadata, merged inventory | Weak | Platform enum and source/target address fields already exist in shared models and specs. | There is no first-class source descriptor in the operator API. `/api/v1/inventory` merges latest snapshots by source but does not tell the caller which sources or snapshot IDs were included. | Do not add broad source CRUD for v1. Add stable source descriptors and provenance fields to inventory and summary responses. |
| Inventory / assets | `/api/v1/inventory`, `/api/v1/snapshots`, `/api/v1/graph`, `models.DiscoveryResult`, `models.VirtualMachine` | Mixed | Core VM and infrastructure fields are already normalized and used consistently across connectors and the web app. | OpenAPI only documents a partial inventory shape. Inventory merge semantics are not explicit. Raw `time.Duration` is exposed without a documented unit. Source provenance and partial-data state are missing. | Keep the current inventory route family, but formalize the full response shape, source provenance, and duration units. |
| Assessments / readiness | dashboard combines inventory + graph + policy + remediation + freshness locally | Missing | The dashboard logic proves Viaduct already has the raw ingredients for readiness assessment. | There is no first-class readiness contract. The frontend currently infers readiness from multiple routes and local heuristics. | Add a dedicated readiness contract for v1 instead of keeping this as frontend-only composition. |
| Migrations / plans | `/api/v1/preflight`, `/api/v1/migrations`, `/api/v1/migrations/{id}`, `/execute`, `/resume`, `/rollback`, `migrate.MigrationState`, `migrate.MigrationPlan` | Weak | Internal migration state, phases, checkpoints, approval gates, and plan structures are already real. | OpenAPI documents only a subset of the migration flow and uses generic `object` schemas. Request format is documented as YAML or JSON, but handlers only decode JSON. `POST /api/v1/migrations` is documented as `201` but returns `202`. Execute and resume depend on in-memory spec lookup. | Preserve the route family, but freeze a stable request and response contract, add durable spec identity, and stop treating list/detail/command surfaces as loosely related. |
| Jobs / operations | async execute/resume commands + polling `GET /api/v1/migrations/{id}` | Missing | The migration itself is already the real long-running unit of work. | There is no stable operation contract, no operation ID, and command responses only return a minimal `status` string. | Do not invent a generic jobs platform for v1. Stabilize command acknowledgements and make migration detail the supported polling contract. |
| Errors / failures | every handler and middleware path | Weak | HTTP status codes are already mostly sensible for auth, bad request, conflict, not found, and rate limiting. `X-Request-ID` already exists. | Error bodies are plain text, not structured. There are no machine-readable error codes, no field errors, and no body-level request correlation. OpenAPI does not describe failures as stable contracts. | Introduce one stable JSON error envelope and apply it across middleware, handlers, and reports. |
| Reports / summary | `/api/v1/summary`, `/api/v1/audit`, `/api/v1/reports/{summary|migrations|audit}` | Mixed | Report names are already narrow and useful. Audit events and summary fields already exist in code. | OpenAPI does not document `/api/v1/audit`. Report JSON payloads are documented as generic `object`, even when they are arrays. Migration reports only expose metadata, not operator-ready lifecycle state. | Keep the report family but formalize each report payload and align summary, audit, and migration report semantics. |

## Stable Contract Decisions

## 1. Sources

### Current State

- `source` and `target` are already explicit in `MigrationSpec`.
- snapshots already carry `source`, `platform`, and `discovered_at`
- merged inventory already loads the latest snapshot per source and platform

### Problem

The operator-facing API does not expose a stable source descriptor even though Viaduct internally reasons about source systems. The caller can fetch merged inventory, but cannot reliably answer:

- which sources were included
- which snapshot IDs were used
- whether any source was stale or partial

### Stable Contract Shape

For v1, the smallest credible move is **not** to add broad source management routes. It is to expose a stable source descriptor inside the existing inventory and summary surfaces.

```json
{
  "id": "vcsa-lab-local",
  "name": "vcsa.lab.local",
  "platform": "vmware",
  "address": "vcsa.lab.local",
  "latest_snapshot_id": "snap-123",
  "last_discovered_at": "2026-04-08T13:40:00Z",
  "discovery_status": "ready",
  "warning_count": 0
}
```

### Required Source Status Enum

- `ready`
- `partial`
- `failed`
- `stale`

### Recommendation

- add `sources[]` to `/api/v1/inventory`
- add `source_count` and `sources[]` or `source_status_counts` to `/api/v1/summary`
- keep source lifecycle outside the v1 API surface beyond those descriptors

## 2. Inventory And Assets

### Current State

- `models.DiscoveryResult` and `models.VirtualMachine` are already Viaduct-specific and useful
- the dashboard depends on VM, disk, NIC, snapshot, host, datastore, cluster, and resource-pool fields
- `/api/v1/inventory` and `/api/v1/snapshots` are already part of the day-to-day operator flow

### Problems

- OpenAPI documents only a partial inventory object
- merged inventory semantics are implicit, not operator-visible
- `duration` currently serializes from Go `time.Duration`, which produces a raw numeric value without a documented unit
- the route does not expose `partial` or `warnings` even though multi-source aggregation can be partial in practice

### Stable Contract Shape

The stable inventory contract should stay close to the current `DiscoveryResult`, but it needs additive fields that make it trustworthy in the UI and in exports.

```json
{
  "tenant_id": "default",
  "inventory_id": "inv-20260408-134000",
  "platform": "vmware",
  "discovered_at": "2026-04-08T13:40:00Z",
  "duration_ms": 1824,
  "partial": false,
  "warnings": [],
  "sources": [
    {
      "id": "vcsa-lab-local",
      "name": "vcsa.lab.local",
      "platform": "vmware",
      "address": "vcsa.lab.local",
      "latest_snapshot_id": "snap-123",
      "last_discovered_at": "2026-04-08T13:40:00Z",
      "discovery_status": "ready",
      "warning_count": 0
    }
  ],
  "vms": [],
  "networks": [],
  "datastores": [],
  "hosts": [],
  "clusters": [],
  "resource_pools": []
}
```

### Stable Decisions

- document the full VM and infrastructure object set already present in `internal/models/inventory.go`
- replace undocumented raw `duration` with a documented `duration_ms` integer in the public contract
- guarantee that `discovered_at` is the aggregate inventory timestamp and `sources[].last_discovered_at` is the per-source timestamp
- document that `/api/v1/inventory` returns the latest snapshot per source and platform within the current tenant scope

## 3. Assessments And Readiness

### Current State

Viaduct already has the ingredients for readiness:

- inventory
- graph
- policy report
- remediation report
- snapshot freshness
- workload-level backup and dependency signals

The dashboard currently composes these in `web/src/features/inventory/`.

### Problem

There is no backend readiness contract today. That means the frontend is deciding:

- what “ready” means
- what “blocked” means
- which missing inputs are acceptable
- how to summarize workload readiness

That is too much product logic to leave outside the shared backend contract.

### Stable Contract Shape

Add a first-class readiness contract for the beachhead and v1 path.

Recommended route:

- `GET /api/v1/assessments/readiness`

Recommended response:

```json
{
  "inventory_id": "inv-20260408-134000",
  "generated_at": "2026-04-08T13:41:30Z",
  "overall_status": "review_required",
  "input_status": {
    "inventory": "available",
    "graph": "available",
    "policies": "available",
    "remediation": "available"
  },
  "summary": {
    "workload_count": 42,
    "ready_count": 17,
    "review_required_count": 20,
    "blocked_count": 5,
    "high_risk_count": 9
  },
  "workloads": [
    {
      "vm_id": "vm-123",
      "name": "web-01",
      "platform": "vmware",
      "readiness_status": "review_required",
      "risk_level": "medium",
      "signals": ["snapshot", "backup-gap"],
      "blockers": [],
      "warnings": [
        {
          "code": "backup_gap",
          "message": "No backup relationship is exposed in the current graph."
        }
      ],
      "dependency_counts": {
        "networks": 2,
        "datastores": 1,
        "backups": 0
      },
      "discovered_at": "2026-04-08T13:40:00Z"
    }
  ]
}
```

### Required Enums

- `overall_status`: `ready`, `review_required`, `blocked`, `partial`
- `input_status`: `available`, `partial`, `unavailable`
- `risk_level`: `low`, `medium`, `high`

### Recommendation

This should be the first new backend contract added for frontend reliability after migration and error hardening. It replaces a large amount of frontend-only inference without resetting any of the underlying engines.

## 4. Migrations And Plans

### Current State

This is the most important contract area for the v1 wedge.

What already exists:

- `MigrationSpec`
- `MigrationPlan`
- `PreflightReport`
- `MigrationState`
- `MigrationCheckpoint`
- phase and checkpoint enums in `internal/migrate/`
- plan, execute, resume, rollback, and detail routes

### Problems

- OpenAPI does not fully document the migration lifecycle routes the dashboard depends on
- `POST /api/v1/migrations` is documented as `201 Created`, but the handler returns `202 Accepted`
- preflight and migration request bodies are documented as YAML or JSON, but `decodeSpec` only accepts JSON today
- the list contract is too thin for the packaged dashboard, because it exposes only `phase` and timestamps
- the detail contract is richer, but not formalized in OpenAPI
- execute, resume, and rollback are missing from the public spec
- execute and resume depend on `Server.specs`, an in-memory map keyed by tenant and migration ID; this is not a stable product contract across process restarts

### Stable Contract Shape

#### Request Contract

For the API:

- freeze `MigrationSpec` JSON as the stable request body for `/api/v1/preflight` and `POST /api/v1/migrations`
- do not advertise YAML request bodies in the API until the handler actually supports them

#### Stable Migration List Item

```json
{
  "id": "migration-123",
  "spec_name": "wave-1",
  "spec_fingerprint": "sha256:8bb1...",
  "source_platform": "vmware",
  "target_platform": "proxmox",
  "lifecycle_state": "approval_pending",
  "phase": "plan",
  "pending_approval": true,
  "started_at": "2026-04-08T13:43:00Z",
  "updated_at": "2026-04-08T13:44:00Z",
  "completed_at": null
}
```

#### Stable Migration Detail

```json
{
  "id": "migration-123",
  "spec_name": "wave-1",
  "spec_fingerprint": "sha256:8bb1...",
  "source_address": "vcsa.lab.local",
  "source_platform": "vmware",
  "target_address": "pve.lab.local",
  "target_platform": "proxmox",
  "lifecycle_state": "approval_pending",
  "phase": "plan",
  "pending_approval": true,
  "approval": {
    "required": true,
    "approved_by": "",
    "approved_at": null,
    "ticket": "CHG-1024"
  },
  "window": {
    "not_before": "2026-04-09T00:00:00Z",
    "not_after": "2026-04-09T04:00:00Z"
  },
  "plan": {},
  "checkpoints": [],
  "workloads": [],
  "errors": [],
  "started_at": "2026-04-08T13:43:00Z",
  "updated_at": "2026-04-08T13:44:00Z",
  "completed_at": null
}
```

### Required Migration Lifecycle Enums

Keep `phase` as the low-level execution phase and add a stable operator-facing lifecycle state.

#### Existing `phase` enum

- `plan`
- `export`
- `convert`
- `import`
- `configure`
- `verify`
- `complete`
- `failed`
- `rolled_back`

#### Required new `lifecycle_state` enum

- `planned`
- `approval_pending`
- `ready`
- `executing`
- `failed`
- `completed`
- `rolled_back`

### Command Contract

`POST /execute`, `POST /resume`, and `POST /rollback` should return a stable command acknowledgement, not a bare status string.

```json
{
  "migration_id": "migration-123",
  "action": "execute",
  "operation_state": "accepted",
  "lifecycle_state": "executing",
  "phase": "plan",
  "accepted_at": "2026-04-08T13:45:00Z",
  "request_id": "req-123"
}
```

### Required Hardening Decision

Viaduct should not keep API execution control dependent on the in-memory `specs` map as a v1 contract. The spec identity, or a persisted normalized execution request, needs to survive API restarts.

That is not optional hardening. It is a trust requirement for execute, resume, and rollback.

## 5. Jobs And Operations

### Current State

- execute and resume are asynchronous
- the API currently starts a goroutine and expects the caller to poll migration detail
- rollback runs inline and returns a result object

### Problem

There is no generic operation contract, but the product does not actually need a generic operation platform for v1.

The current risk is not “Viaduct lacks a jobs engine.”
The current risk is “the command contract is too weak to be dependable.”

### Stable Contract Decision

For v1:

- treat `migration` as the primary long-running operation
- use `GET /api/v1/migrations/{id}` as the stable polling contract
- standardize command acknowledgements for `execute`, `resume`, and `rollback`
- defer any generic `/api/v1/operations` resource until a later phase

### Required Operation State Enum

- `accepted`
- `running`
- `completed`
- `failed`

`operation_state` belongs on command acknowledgements now. A full operation resource can come later if needed.

## 6. Errors And Failures

### Current State

Viaduct already sets appropriate HTTP status codes in many places, and it already emits `X-Request-ID`.

### Problem

The error contract is still plain text. That hurts:

- frontend reliability
- support triage
- operator trust
- contract testing
- reportability

### Stable Error Envelope

Every non-2xx JSON API response should use one error shape.

```json
{
  "error": {
    "code": "approval_required",
    "message": "Migration requires approval before execution.",
    "request_id": "req-123",
    "retryable": false,
    "details": {
      "migration_id": "migration-123"
    },
    "field_errors": []
  }
}
```

### Required Error Fields

- `code`
- `message`
- `request_id`
- `retryable`
- `details`
- `field_errors`

### Required `field_errors` Shape

```json
{
  "path": "source.address",
  "message": "source.address is required"
}
```

### Required Error Codes

- `missing_credentials`
- `invalid_credentials`
- `permission_denied`
- `rate_limit_exceeded`
- `invalid_request`
- `invalid_spec`
- `migration_not_found`
- `service_account_not_found`
- `report_not_found`
- `approval_required`
- `window_not_open`
- `window_closed`
- `conflict`
- `internal_error`

### HTTP Expectations

- `400` malformed JSON, invalid query params, invalid spec
- `401` missing or invalid tenant credentials
- `403` valid caller without required role or permission
- `404` unknown migration, report, service account, or route
- `409` approval, window, state, or duplicate-resource conflict
- `429` tenant rate limit exceeded, with `Retry-After`
- `500` unexpected backend failure

## 7. Reports And Summary

### Current State

- `/api/v1/summary` already exposes a useful tenant summary
- `/api/v1/reports/summary`, `/migrations`, and `/audit` already exist
- `/api/v1/audit` already exists for direct audit access

### Problems

- OpenAPI does not document `/api/v1/audit`
- report JSON payloads are not explicitly typed
- migration report output is still tied to metadata rather than the richer lifecycle state the operator actually cares about
- summary semantics such as pending approvals are real, but not formalized as part of the stable operator contract

### Stable Contract Decisions

#### `/api/v1/summary`

Keep `TenantSummary` as the summary shell contract and fully document these fields:

- `tenant_id`
- `workload_count`
- `snapshot_count`
- `active_migrations`
- `completed_migrations`
- `failed_migrations`
- `pending_approvals`
- `recommendation_count`
- `platform_counts`
- `last_discovery_at`
- `quotas`
- `snapshot_quota_free`
- `migration_quota_free`

#### `/api/v1/reports/summary`

- JSON shape should match `TenantSummary` or a documented additive `SummaryReport`
- CSV should be a flattened representation of the same fields

#### `/api/v1/reports/migrations`

- JSON shape should be `MigrationListItem[]`
- CSV should expose the same lifecycle fields in a flattened row set

#### `/api/v1/audit` and `/api/v1/reports/audit`

- both should use the same `AuditEvent` schema
- `AuditEvent.outcome` enum is already clear: `success`, `failure`
- `request_id` must remain operator-visible for traceability

## Required Enums And Lifecycle States

The following enums should be treated as stable once documented in OpenAPI and contract tests.

### Platform

- `vmware`
- `proxmox`
- `hyperv`
- `kvm`
- `nutanix`

### Power State

- `on`
- `off`
- `suspended`
- `unknown`

### Tenant Role

- `viewer`
- `operator`
- `admin`

### Tenant Permission

- `inventory.read`
- `reports.read`
- `lifecycle.read`
- `migration.manage`
- `tenant.read`
- `tenant.manage`

### Auth Method

- `tenant-api-key`
- `service-account`
- `default-fallback`

`default-fallback` should remain explicitly documented as a lab and compatibility behavior, not a product-grade packaged auth story.

### Preflight Check Status

- `pass`
- `warn`
- `fail`

### Preflight Check Name

- `source-connectivity`
- `target-connectivity`
- `execution-window`
- `approval-gate`
- `disk-space`
- `network-mappings`
- `name-conflicts`
- `source-backup`
- `disk-formats`
- `resource-availability`
- `rollback-readiness`
- `execution-plan`

### Migration Phase

- `plan`
- `export`
- `convert`
- `import`
- `configure`
- `verify`
- `complete`
- `failed`
- `rolled_back`

### Migration Lifecycle State

- `planned`
- `approval_pending`
- `ready`
- `executing`
- `failed`
- `completed`
- `rolled_back`

### Checkpoint Status

- `pending`
- `running`
- `completed`
- `failed`

### Readiness Status

- `ready`
- `review_required`
- `blocked`
- `partial`

### Assessment Input Status

- `available`
- `partial`
- `unavailable`

### Audit Outcome

- `success`
- `failure`

### Report Name

- `summary`
- `migrations`
- `audit`

### Report Format

- `json`
- `csv`

## Frontend Assumptions To Eliminate

These are the specific dashboard assumptions that should move into the backend contract.

- The frontend should not infer full workflow state from `phase`, `pending_approval`, and `errors` alone. Add `lifecycle_state`.
- The frontend should not track preflight freshness and saved-plan freshness with local JSON stringification only. Add `spec_fingerprint` to preflight and migration responses.
- The frontend should not build readiness from `inventory + graph + policies + remediation` as if that composition were the stable product contract. Add a readiness assessment response.
- The frontend should not parse plain-text error bodies. All failures should use the stable JSON error envelope.
- The frontend should not treat `MigrationMeta.phase == "plan"` as enough to describe whether a plan is ready, blocked, or waiting on approval. Add lifecycle and approval fields to list items.
- The frontend should not guess whether missing dependency or remediation data is acceptable. Backend readiness input status should make partial data explicit.
- The frontend should not rely on undocumented report JSON shapes that are currently described in OpenAPI as generic objects.

## Priority Order For Hardening

## Priority 0: Migration And Command Contract

Reason:

- it is the most important contract for the v1 wedge
- it currently has the most dangerous trust gap because command flows depend on in-memory spec lookup

Required work:

- persist spec identity or normalized execution input needed for execute, resume, and rollback
- add `lifecycle_state` and `spec_fingerprint`
- align list, detail, execute, resume, rollback, and report migration shapes
- fix OpenAPI status-code and request-body mismatches

## Priority 1: Structured Error Contract

Reason:

- every operator and dashboard path depends on this
- it improves frontend reliability, supportability, and request-correlation trust immediately

Required work:

- replace plain-text `http.Error` responses with one JSON envelope
- surface `request_id` and `Retry-After` consistently
- add failure schemas to OpenAPI

## Priority 2: Preflight Contract

Reason:

- preflight is the decision gate between planning and pilot execution
- today it is internally useful but externally under-specified

Required work:

- formalize preflight check names as a stable enum
- add `generated_at` and `spec_fingerprint`
- document JSON-only API behavior until YAML support is real

## Priority 3: Inventory Provenance Contract

Reason:

- assessment credibility depends on knowing which sources and snapshots the inventory actually represents

Required work:

- add `inventory_id`, `sources[]`, `partial`, and `warnings`
- document merged-latest-by-source behavior
- replace ambiguous raw duration with `duration_ms`

## Priority 4: Readiness Assessment Contract

Reason:

- this removes one of the biggest remaining frontend-only product behaviors

Required work:

- introduce `/api/v1/assessments/readiness`
- make workload readiness, risk, and missing-input state backend-owned

## Priority 5: Summary, Audit, And Reports Alignment

Reason:

- operators need stable review and export outputs

Required work:

- document `/api/v1/audit`
- align report JSON shapes with documented schema
- upgrade migration report shape from thin metadata to lifecycle-aware report rows

## Priority 6: Contract Test Coverage

Reason:

- `make contract-check` is currently too shallow for a release-grade operator contract

Required work:

- verify the full v1 route set, not just a subset
- assert documented response codes match handlers
- assert required enums and object schemas exist in OpenAPI
- add route-specific response shape tests for inventory, preflight, migration detail, command errors, and reports

## Recommended Draft Contract Examples

These are the first examples worth promoting into OpenAPI once the hardening work starts.

### ErrorResponse

```json
{
  "error": {
    "code": "invalid_spec",
    "message": "Migration specification is invalid.",
    "request_id": "req-123",
    "retryable": false,
    "details": {},
    "field_errors": [
      {
        "path": "target.address",
        "message": "target.address is required"
      }
    ]
  }
}
```

### PreflightReport

```json
{
  "generated_at": "2026-04-08T13:42:00Z",
  "spec_fingerprint": "sha256:8bb1...",
  "can_proceed": false,
  "pass_count": 7,
  "warn_count": 2,
  "fail_count": 1,
  "checks": [
    {
      "name": "approval-gate",
      "status": "fail",
      "message": "Migration requires approval before execution.",
      "duration_ms": 1
    }
  ],
  "plan": {}
}
```

### MigrationCommandAccepted

```json
{
  "migration_id": "migration-123",
  "action": "resume",
  "operation_state": "accepted",
  "lifecycle_state": "executing",
  "phase": "convert",
  "accepted_at": "2026-04-08T13:46:00Z",
  "request_id": "req-456"
}
```

### AuditEvent

```json
{
  "id": "audit-123",
  "tenant_id": "default",
  "actor": "service-account:wave-runner",
  "request_id": "req-456",
  "category": "migration",
  "action": "execute",
  "resource": "migration-123",
  "outcome": "success",
  "message": "migration execution started",
  "details": {
    "approved_by": "ops@example.com"
  },
  "created_at": "2026-04-08T13:46:00Z"
}
```

## Final Recommendation

The backend hardening work should begin with the migration lifecycle and error contracts, not with more UI polish and not with broad new route families.

The smallest credible product move is:

1. make migration plan, detail, and command responses stable and durable
2. replace plain-text failures with one structured error envelope
3. formalize inventory provenance and preflight semantics
4. move readiness out of frontend-only composition and into a backend contract

That sequence preserves the current architecture, directly improves operator trust, and aligns with the VMware-exit assessment-to-pilot wedge Viaduct is actually trying to own.
