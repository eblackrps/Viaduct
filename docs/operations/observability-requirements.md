# Observability Requirements

This document is the original product-facing requirements artifact for observability hardening. The current implemented backend tracing and local Grafana + Tempo runbook now live in [observability.md](observability.md).

This document defines the minimum observability model Viaduct needs after Phase 5 so operators and maintainers can answer four questions quickly:

1. what failed
2. where it failed
3. why it failed
4. what to do next

This is a product and engineering artifact, not a generic telemetry wishlist. It is tied to Viaduct's real workflows, current repo surfaces, and the supported early-product path.

## 1. Current-State Summary

### What exists today
- `internal/api/observability.go` already gives the API a request-level observability foundation:
  - `X-Request-ID` is accepted or generated
  - every API response returns `X-Request-ID`
  - a request-completion log line is emitted with `request_id`, method, path, status, duration, tenant, and auth method
  - `/api/v1/metrics` exposes normalized route metrics and store/tenant gauges
- `internal/api/error_response.go` and `web/src/api.ts` already support structured API failures with an operator-visible `request_id`.
- `internal/migrate/orchestrator.go`, `internal/migrate/plan.go`, and `internal/migrate/preflight.go` already persist or return workflow-native diagnostics:
  - migration phase
  - checkpoints
  - checkpoint diagnostics
  - preflight check names, statuses, messages, and durations
  - migration-level `errors`
- `internal/models/audit.go` and `internal/api/reports.go` already provide tenant-scoped audit events and audit export.
- `internal/store/store.go` and `internal/api/server.go` already expose store diagnostics through `/api/v1/about`.
- The dashboard already exposes pieces of the story:
  - `web/src/features/settings/SettingsPage.tsx` shows runtime context, auth mode, role, and store backend
  - `web/src/components/MigrationProgress.tsx` shows checkpoint status and checkpoint messages
  - `web/src/features/reports/ReportsPage.tsx` exposes exports and migration history
  - `web/src/features/inventory/WorkloadDetailPanel.tsx` already admits that there is no VM-scoped activity feed yet

### What prior phases likely improved
- Phase 3 appears to have established the tenant-aware API/store model, which is why request correlation, audit, and metrics are already tenant-scoped.
- Phase 4 appears to have established execution windows, approvals, checkpoints, resume behavior, and rollback state, which gives Viaduct a real execution history surface instead of a fire-and-forget command flow.
- Phase 5 appears to have improved contract clarity with structured API errors and request correlation that the frontend can already surface.

### What is still weak, ambiguous, over-scoped, or risky
- Observability is real but fragmented. Operators can see some state in `Settings`, some in `Migrations`, some in `Reports`, some in raw API errors, and some only in backend logs.
- The backend has request logs and metrics, but not yet a clearly defined structured logging contract for discovery, migration phases, report export, or authorization failures.
- There is no generic trace model today, and there should not be a generic tracing platform bolted on casually. The repo already has workflow-native signals, but they are not yet treated as the canonical execution trace.
- The current frontend error model collapses most failures to plain strings. `web/src/api.ts` preserves request IDs in the message, but the UI does not consistently retain the raw error code, retryability, field errors, or route/action context.
- Discovery history is represented by snapshots, not by a first-class discovery job/event history. That is acceptable for v1 if the expectations remain explicit.
- CLI-driven discovery does not have an HTTP `request_id`, so the first version of this spec was too API-centric for the actual product path.
- `GET /api/v1/migrations` and `GET /api/v1/migrations/{id}` expose state and checkpoints, but there is no clear product contract yet for "last failure summary", "operator next step", or "most recent blocking condition."
- `web/src/features/inventory/WorkloadDetailPanel.tsx` explicitly notes the absence of a VM-scoped activity or audit feed.
- There is no separate in-product maintainer/support diagnostics view yet. Support depends on logs, request IDs, metrics, audit exports, and migration detail.
- The biggest current correlation gap is the async boundary in `internal/api/server.go`: execute and resume launch background work with `context.Background()`, which drops the originating request context and weakens request-to-execution traceability.

### What should be preserved
- `X-Request-ID` as the ingress-level correlation primitive.
- `migration_id` as the long-running workflow identifier instead of inventing a generic jobs platform.
- migration checkpoints, preflight checks, audit events, and summary/report exports as the workflow-native observability signals.
- the current Settings, Migrations, Reports, and Workload detail surfaces instead of replacing them with a brand-new observability console.
- tenant-scoped metrics, audit, and persisted migration state.

### The smallest credible next move
- Freeze one observability model that treats:
  - `request_id` as the request correlation primitive
  - `migration_id` as the execution correlation primitive
  - a discovery-specific run identifier as the CLI/API discovery correlation primitive
  - audit events plus checkpoints as the operator-visible execution trace
  - `/api/v1/metrics` and structured backend logs as the maintainer/support truth
  - existing Settings, Migrations, and Reports surfaces as the minimum in-product observability UI

## 2. Observability Framing

Viaduct does not need a generic enterprise observability platform for v1.

It does need workflow-native observability that fits the supported product path:

1. establish tenant and auth context
2. run discovery
3. review inventory and workload context
4. run preflight
5. save a plan
6. execute, resume, or roll back
7. export evidence

Observability must serve two audiences without forcing them into the same view.

### Operator-facing observability
Operators need:
- the current state of the workflow
- the blocking reason, if any
- whether the failure is safe to retry
- what action to take next
- a request or execution identifier they can hand to support

### Maintainer/support-facing observability
Maintainers and support need:
- the exact request, route, and tenant context
- the exact migration or snapshot identifier
- the raw backend or connector failure detail
- the audit and checkpoint timeline around the failure
- enough metadata to reproduce, debug, or explain the issue

### The guiding rule
For any meaningful failure, Viaduct should produce:
- one human-readable explanation for the operator
- one raw diagnostic record for support and maintainers
- one correlation path that links them together

## 3. Workflow-Native Observability Model

Viaduct should treat the following product-native records as its primary observability backbone.

| Workflow step | Current source of truth | Supported observability record |
| --- | --- | --- |
| Tenant/auth context | `/api/v1/about`, `/api/v1/tenants/current` | runtime context record |
| Discovery | persisted snapshots and discovery results | snapshot history record |
| Inventory/workload inspection | `/api/v1/inventory`, `/api/v1/graph`, lifecycle routes | inventory provenance and signal-completeness record |
| Preflight | `/api/v1/preflight` | preflight result record |
| Saved plan | `/api/v1/migrations`, `/api/v1/migrations/{id}` | migration record with plan summary |
| Execute/resume | `/api/v1/migrations/{id}/execute`, `/resume`, audit events, checkpoints | execution trace record |
| Rollback | `/api/v1/migrations/{id}/rollback`, rollback result, checkpoints | rollback trace record |
| Report and handoff | `/api/v1/reports/*`, `/api/v1/audit` | export and evidence record |

### Do not invent a generic tracing platform for v1
- No generic distributed tracing requirement
- No separate jobs platform
- No generic execution-service abstraction

### What counts as the trace in v1
For Viaduct v1, the supported correlation chain is:

`request_id` -> `audit_event` -> `migration_id` -> `checkpoint timeline` -> `report/export evidence`

That chain is enough for the supported early-product path if it is made explicit and consistent.

For discovery, the equivalent chain is:

`discovery_run_id` -> `snapshot_id` -> `inventory baseline` -> `report/export evidence`

The first version of this document did not make that CLI-first discovery path explicit enough.

## 4. Operator-Facing Observability Needs

### Operator observability requirements by workflow step

| Step | Operator must be able to answer | Current signal | Required v1 observability behavior |
| --- | --- | --- | --- |
| Tenant/auth context | Am I in the right tenant, against the right API, with the right permission? | `Settings`, `/api/v1/tenants/current`, `/api/v1/about` | show tenant, role, auth mode, store backend, build metadata, and request ID on auth failures |
| Discovery | Did discovery run, when, against which platform, and is the result fresh enough to trust? | snapshots, inventory timestamps, discovery result errors | show latest snapshot time, source/platform, stale state, and any non-fatal discovery errors or missing context |
| Inventory/workload review | Is this workload ready enough to plan from? | inventory, graph, lifecycle signals, workload detail panel | show partial-signal warnings clearly when graph/policy/remediation data is missing |
| Preflight | What failed, is it blocking, and what do I do next? | preflight check list with status/message/duration | keep named checks, classify as block vs warn, and show next-step guidance tied to the failing check |
| Saved plan | What was saved, for which workloads, and under which execution controls? | migration state, plan data | show migration ID, wave summary, approval state, execution window, and freshness relative to current draft |
| Execute/resume | Was the command accepted, blocked, or failed? | command response, audit event, migration state | show accepted vs blocked state, request ID, approval/window blockers, and where to monitor next |
| Monitoring | What phase is running, what already completed, and what is currently broken? | checkpoints, phase, workload errors, migration `errors` | show checkpoint timeline, workload failures, last updated time, and a short operator-safe failure summary |
| Rollback | Did rollback finish cleanly or partially? | rollback result, migration state, checkpoint diagnostics | show removed/restored counts, returned errors, and explicit "safe to retry" vs "manual follow-up needed" guidance |
| Reports/handoff | Can I export evidence that matches what the product is showing? | reports and audit export | reports, audit, and migration state must agree on phase, outcome, actor, and correlation IDs |

### Operator-visible failure summary pattern

Every major failure surface should be able to render a summary in this shape:

```text
State: Execution blocked
Reason: Migration requires approval before execution
Scope: migration migration-42
Next step: Record approved_by and ticket, then retry execute
Request ID: req-123
Technical details available
```

### Operator-safe wording rules
- Lead with current state, not raw exception text.
- Separate blockers from warnings.
- Explain retry safety when possible.
- Always show a correlation handle.
- Do not make operators parse raw connector or store errors to decide their next action.

## 5. Maintainer And Support-Facing Observability Needs

Maintainers and support need a stricter view than operators.

### Minimum support triage inputs
- `request_id`
- `tenant_id`
- `migration_id` when the issue involves planning or execution
- `snapshot_id` when the issue involves discovery or stale inventory
- API route and method
- auth method and effective role when relevant
- current build and commit from `/api/v1/about`
- store backend and persistence mode from `/api/v1/about`
- relevant audit events from `/api/v1/audit`
- relevant migration state, checkpoints, and rollback results
- relevant metrics from `/api/v1/metrics`

### Support must be able to answer
- Was the failure request-scoped, background execution-scoped, or data-state-scoped?
- Did the request fail before the command was accepted, or did the command start and fail later?
- Was the failure caused by auth, validation, execution controls, connector behavior, persistence, or UI rendering?
- Is the current persisted state still trustworthy enough for the operator to continue?
- What exact next step should support recommend: retry, wait, resume, roll back, refresh discovery, or escalate?

### Minimum support packet for a serious issue
- timestamp of the failure
- operator-reported symptom
- `request_id`
- `migration_id` or `snapshot_id`
- tenant ID
- relevant route and HTTP status
- current migration phase and checkpoint statuses
- audit events around the failure
- store backend and version/build info

## 6. Key Logs, Traces, Events, And Correlation Points

| Signal | Current repo source | Audience | Use | Current gap |
| --- | --- | --- | --- | --- |
| request completion log | `internal/api/observability.go` | maintainers/support | route, status, duration, tenant, auth correlation | no explicit severity or migration/snapshot IDs |
| structured API error envelope | `internal/api/error_response.go` | operators and support | operator-safe message plus `request_id`, `error.code`, retryability | frontend currently collapses most rich fields into a single string |
| audit event | `internal/models/audit.go`, `/api/v1/audit` | operators and support | action attribution and evidence trail | export auditing and authz-denial coverage are still incomplete |
| migration state | `internal/migrate/orchestrator.go`, `/api/v1/migrations/{id}` | operators and support | current phase, errors, execution controls | no explicit last-failure summary contract |
| checkpoint timeline | `internal/migrate/plan.go`, `MigrationProgress.tsx` | operators and support | resumable phase trace | diagnostics are present but the async request-to-phase correlation boundary is still weak |
| preflight report | `internal/migrate/preflight.go` | operators and support | named check outcomes and durations | not persisted as a first-class history record |
| rollback result | `internal/migrate/rollback.go` | operators and support | cleanup/restoration outcome | no unified in-product rollback summary view beyond raw result/state |
| snapshot metadata | store snapshots, `/api/v1/snapshots` | operators and support | discovery history baseline | no first-class discovery failure history view |
| discovery run log | CLI discovery path and discovery engine | maintainers/support | failed-before-save or partial discovery correlation | no `discovery_run_id` exists yet |
| inventory result errors | `models.DiscoveryResult.Errors` | operators and support | partial discovery diagnostics | not consistently surfaced as a first-class UI status |
| metrics | `/api/v1/metrics` | maintainers/support | fleet health and backend behavior | not yet surfaced in-product for operators |

## 7. Correlation ID And Execution ID Model

### The recommended model

Viaduct should standardize on a small set of IDs, each with a clear purpose.

| Identifier | Scope | Current status | Required use |
| --- | --- | --- | --- |
| `request_id` | one HTTP request | already implemented | every API response, API error, request log, and audit event |
| `origin_request_id` | one background execution chain launched by an HTTP request | missing | background migration logs and events must retain the execute/resume request that launched them |
| `migration_id` | one saved migration and its execution lifecycle | already implemented | every migration command, migration log/event, and operator handoff |
| `discovery_run_id` | one discovery attempt, especially CLI-first or failed-before-save | missing | discovery logs and support triage for discovery failures before `snapshot_id` exists |
| `snapshot_id` | one persisted discovery baseline | already implemented | discovery troubleshooting, freshness checks, and inventory provenance |
| `audit_event.id` | one audit record | already implemented | event export and support reference |
| `tenant_id` | one security boundary | already implemented | every persisted event and every log when known |

### Execution ID rule
- Do not introduce a generic `operation_id` or jobs abstraction for v1.
- For migration execution, `migration_id` is the execution ID.
- For discovery, `snapshot_id` is the baseline/history ID after persistence, while `discovery_run_id` is the correlation handle during the attempt itself.
- If a later background worker model is added, it should still preserve `migration_id` as the operator-facing execution handle for migration work.

### Correlation rules
- Every mutation request must produce a `request_id`.
- Every async execute or resume path must preserve the accepted request as `origin_request_id` across the background boundary. The current `context.Background()` handoff in `internal/api/server.go` is not sufficient.
- Every migration-related log or event must include `migration_id`.
- Every discovery attempt should generate a `discovery_run_id`.
- Every discovery-related log or event must include `snapshot_id` once a snapshot exists; before persistence, use `discovery_run_id` plus source platform and address in the log context.
- Every support-facing summary must include `tenant_id`.
- The UI must preserve `request_id` and `migration_id` instead of flattening them into unstructured prose.

## 8. Job And Event History Expectations

### Migration history
Migration is the only true long-running job type Viaduct needs to optimize first.

Minimum history expectation:
- saved migration metadata in `/api/v1/migrations`
- full migration detail in `/api/v1/migrations/{id}`
- checkpoint history
- audit events for plan, execute, resume, rollback, and relevant failures
- rollback result visibility

### Discovery history
Viaduct does not need a generic discovery jobs system for v1.

Minimum history expectation:
- snapshot list
- snapshot time, source, platform, and VM count
- visible discovery freshness
- visible non-fatal discovery errors from the current baseline when present

### Preflight history
Viaduct does not need a separate persisted preflight jobs ledger for v1.

Minimum history expectation:
- the latest preflight report used in the current planning flow
- clear block vs warn classification
- migration ID association once a plan is saved

### Report/export history
Viaduct does not need a separate export jobs platform for v1.

Minimum history expectation:
- export actions are auditable
- exported outputs match visible state
- the operator can tell what was exported, by whom, and under which request ID

## 9. Frontend Error Capture Expectations

### Current implementation reality
- `web/src/api.ts` converts API failures into `Error` objects and preserves `request_id` in the message when present.
- most page-level hooks and surfaces store a plain string and render it inline.
- the dashboard has route-specific loading and error states, but not a unified typed UI error model.
- this means the UI currently cannot reliably render richer operator-facing technical details without reparsing message strings or extending the client types first.

### Required v1 frontend behavior
- preserve structured API error fields in the client model, not just the rendered message
- attach page or action context to UI failures, for example:
  - `settings.load`
  - `inventory.refresh`
  - `migration.preflight`
  - `migration.execute`
  - `reports.export.audit`
- render a human-readable explanation plus `request_id`
- show retryability when the backend marks the error `retryable`
- retain field errors for form-like flows such as migration planning or tenant administration
- provide a copyable technical-details block for support without exposing secrets

### What v1 does not require
- mandatory third-party browser telemetry
- SaaS crash reporting
- a full client event pipeline

### What v1 should add inside the product
- route-level error boundaries for major surfaces so unexpected rendering failures do not collapse the entire app silently
- consistent "technical details" disclosure that includes:
  - request ID
  - error code
  - action scope
  - retryability

## 10. Raw Diagnostic Detail Versus Human-Readable Explanation

Viaduct should make a strict distinction between:
- human-readable operator explanations
- raw diagnostic detail for support and maintainers

### Human-readable explanation
Should answer:
- what state the workflow is in
- what failed
- what the operator should do next
- whether it is safe to retry

Should not require:
- reading raw stack traces
- reading connector response bodies
- guessing whether a conflict is recoverable

### Raw diagnostic detail
Should include:
- request ID
- route and method
- tenant ID
- migration ID or snapshot ID
- error code
- retryability
- auth method and effective role when relevant
- checkpoint diagnostics
- raw connector/store/backend error text

### Example split

Human-readable:

```text
Preflight failed because the target already contains VM "payments-db-01".
Next step: choose a different target name or remove the conflicting target VM.
Request ID: req-123.
```

Raw diagnostic detail:

```json
{
  "error_code": "invalid_request",
  "route": "POST /api/v1/preflight",
  "request_id": "req-123",
  "tenant_id": "tenant-a",
  "migration_id": "migration-42",
  "check": "name-conflicts",
  "raw_message": "target already contains VM \"payments-db-01\""
}
```

## 11. Minimum Dashboards Or Views Needed Inside The Product

Viaduct should not start with a dedicated observability console. It should strengthen the screens operators already use.

### Required in-product views

| View | Current surface | Required v1 role |
| --- | --- | --- |
| Runtime context | `Settings` | tenant, auth mode, role, build, store backend, current trust context |
| Migration run detail | `Migrations` and `MigrationProgress` | checkpoint timeline, blocking reason, last failure summary, request-linked command history |
| Reports and history | `Reports` | exports, recent failures, audit history, migration history, snapshot history |
| Workload provenance | `Inventory` detail | latest baseline, partial-signal warnings, limited activity context until a richer feed exists |

### Explicitly not required for v1
- full metrics dashboards inside the product
- a distributed traces page
- a dedicated VM activity feed for every workload
- a full log viewer in the browser

### Product guidance
- Settings is the runtime and trust context surface.
- Migrations is the primary execution observability surface.
- Reports is the handoff and evidence surface.
- Any future observability UI should extend those surfaces before adding a separate product area.

## 12. Minimum Structured Logging Expectations In The Backend

### Current reality
The backend already emits structured-looking `log.Printf` lines, but the shape is not yet defined as a product contract.

The first version of this spec was too loose here. Viaduct does not currently have a structured logger abstraction with native levels, so the first landable step is to standardize key-value output on top of the existing `log.Printf` usage rather than quietly assuming a logger migration.

### Required v1 logging contract
Every backend log that represents a meaningful operator or support event should include:
- `level`
- `component`
- `category`
- `action`
- `outcome`
- `message`
- `request_id` when there is one
- `tenant` or `tenant_id` when known
- `migration_id` when relevant
- `snapshot_id` when relevant
- `auth` or `auth_method` when relevant
- `status` for HTTP request completion logs
- `duration_ms` when timing matters

### Required backend logging points
- request completion
- authentication failure
- authorization denial
- discovery start and completion
- discovery partial failure
- preflight request failure
- migration command accepted, blocked, failed
- migration phase started, completed, failed
- rollback started, completed, failed
- report export completed or failed
- audit save failure
- store diagnostics failure

### Redaction rules
- never log credentials or API keys
- never log raw secrets from config or connector payloads
- avoid logging whole request bodies for migration specs
- log `credential_ref`, route, platform, and IDs, not passwords or tokens

### Repo-specific implementation guardrail
- Do not switch Viaduct to a new logging framework as part of the first observability hardening pass.
- First normalize the field set and key names in the existing `log.Printf` call sites in:
  - `internal/api/observability.go`
  - `internal/api/reports.go`
  - `internal/api/server.go`
  - `internal/migrate/orchestrator.go`
  - `internal/migrate/rollback.go`

## 13. Suggested Log Severity Conventions

| Level | Meaning | Viaduct examples |
| --- | --- | --- |
| `INFO` | expected lifecycle event or successful command | request served, migration execute accepted, report exported |
| `WARN` | degraded but continuing, operator action required soon | rate limit exceeded, partial discovery, preflight warning-only result, stale baseline |
| `ERROR` | request or phase failed, operator/support action required | execute blocked by invalid state, connector failure, rollback returned errors |
| `CRITICAL` | startup or persistence failure that undermines service trust | store unavailable at startup, unrecoverable schema mismatch |

### Debug note
`DEBUG` may be useful in local development, but it is not required as part of the early-product support contract.

## 14. Sample Structures

### Sample request-completion log

```text
level=INFO component=api category=http action=request-complete outcome=success request_id=req-123 method=POST path=/api/v1/migrations/:id/execute status=202 duration_ms=42 tenant=tenant-a auth=service-account migration_id=migration-42
```

### Sample migration phase failure log

```text
level=ERROR component=migrate category=execution action=phase-failed outcome=failure migration_id=migration-42 tenant=tenant-a phase=convert request_id=req-123 message="disk conversion failed" error_code=internal_error
```

### Sample operator-visible failure summary pattern

```text
State: Rollback incomplete
Reason: Target cleanup finished, but source restoration returned errors
Scope: migration migration-42
Next step: inspect rollback diagnostics, keep the run in failed state, and do not retry execute
Request ID: req-456
```

## 15. Runbook Notes For Common Failures

| Symptom | First operator surface | First maintainer/support check | Likely next action |
| --- | --- | --- | --- |
| auth failure in dashboard | `Settings` or inline page error | request ID plus `/api/v1/tenants/current` behavior | fix credential, role, or tenant selection |
| preflight fails on approval or window | `Migrations` preflight panel | preflight check list and request log | update approval payload or wait for window |
| execute returns conflict | `Migrations` execute stage | audit event plus migration detail | resolve blocking state, then retry or resume |
| migration stalls after acceptance | `Migrations` checkpoint timeline | migration state, audit events, request log, metrics | refresh state, determine if resume or rollback is required |
| rollback returns errors | rollback result in `Migrations` | rollback diagnostics and recovery point state | treat run as failed, investigate cleanup/restoration mismatch |
| reports do not match visible state | `Reports` and `Migrations` | compare migration detail, audit events, and export output | fix contract drift before expanding scope |
| stale or missing inventory | `Inventory`, `Settings`, snapshot metadata | snapshot list, discovery errors, store backend | rerun discovery or investigate snapshot persistence |

## 16. Concrete Repo Targets

The first version of this artifact did not tie the work tightly enough to the codebase. The following repo targets are the concrete ownership points for the first observability hardening pass.

| Area | Files | Why they matter |
| --- | --- | --- |
| API request correlation and metrics | `internal/api/observability.go` | request IDs, request completion logs, normalized route metrics |
| command acceptance and async handoff | `internal/api/server.go` | execute/resume/rollback request handling, migration command responses, background context handoff |
| auth failures and denials | `internal/api/middleware.go` | missing credentials, invalid credentials, authorization denials |
| audit and export history | `internal/api/reports.go` | audit route, report export behavior, audit save failures |
| migration phase and checkpoint state | `internal/migrate/orchestrator.go`, `internal/migrate/plan.go`, `internal/migrate/rollback.go` | phase lifecycle, checkpoint diagnostics, rollback result signals |
| preflight diagnostics | `internal/migrate/preflight.go` | named checks, durations, operator-safe blocking messages |
| frontend typed errors | `web/src/types.ts`, `web/src/api.ts` | structured UI error preservation instead of string flattening |
| operator execution UI | `web/src/features/migrations/useMigrationWorkspace.ts`, `web/src/features/migrations/MigrationWorkflow.tsx`, `web/src/components/MigrationProgress.tsx` | operator-visible blocking reason, retry guidance, technical details |
| evidence and history UI | `web/src/features/reports/ReportsPage.tsx`, `web/src/components/MigrationHistory.tsx` | recent audit/failure history and export correlation |
| support/operator docs | `SUPPORT.md`, `docs/reference/troubleshooting.md`, `docs/operations/migration-operations.md` | support packet expectations and operator debugging path |

## 17. Phased Implementation Order

### Phase 1: Close the correlation-model gaps first
- Add this observability requirements document and link it from the main docs entrypoints.
- Align support docs so issues include `request_id`, `migration_id`, and `snapshot_id` when relevant.
- Treat `request_id`, `migration_id`, and `snapshot_id` as the canonical existing handles.
- Introduce `origin_request_id` for async execute/resume chains and `discovery_run_id` for discovery attempts.

Acceptance gate:
- support can correlate an accepted execute or resume request to later background execution logs
- failed discovery attempts can be referenced even when no snapshot is saved

### Phase 2: Normalize backend signals
- Define the backend structured logging field set in `internal/api/observability.go` and adjacent migration/report paths.
- Add explicit logs for migration command acceptance/block/failure, report export, discovery start/completion/partial failure, and authorization denials.
- Keep the log shape consistent across API, migration, rollback, and reporting paths.

Acceptance gate:
- one maintainer can grep by `request_id`, `migration_id`, `origin_request_id`, or `discovery_run_id` and reconstruct the relevant failure path from logs alone

### Phase 3: Tighten workflow-native execution traces
- Treat migration checkpoints plus audit events as the supported execution trace.
- Normalize migration failure summaries in `internal/migrate/` and `internal/api/server.go` so operators see one stable explanation path.
- Surface discovery result errors and stale-baseline status more explicitly.

Acceptance gate:
- operators can answer "what failed and what next" from `Migrations` or `Reports` without needing raw backend logs
- maintainers can answer "where exactly did it fail" from checkpoints, audit events, and logs without reading the database manually

### Phase 4: Improve frontend observability handling
- Extend `web/src/types.ts` and `web/src/api.ts` so the client preserves structured error fields instead of only flattening to strings.
- Add route-level error boundaries for major product surfaces.
- Add request-linked failure summaries and recent audit/failure history to the existing `Migrations` and `Reports` surfaces.

Acceptance gate:
- the UI can render operator-safe text and a separate technical-details block without reparsing error-message strings
- `request_id` and relevant execution IDs remain visible and copyable in the product

### Phase 5: Operator-ready packaging and validation
- Update `SUPPORT.md` and troubleshooting guidance to ask for the right IDs and workflow context.
- Add tests that verify request IDs, normalized metrics paths, checkpoint visibility, structured errors, and tenant-scoped audit/export behavior.
- Validate the observability path manually against the primary reliability path: auth context, discovery, preflight, execute, monitor, rollback, report.

### Later, not now
- external tracing backends
- generic distributed tracing
- browser log streaming
- full in-product metrics dashboards
- a separate observability product area

## 18. Maintainer Notes

This model is intentionally narrow.

- It does not reset the architecture.
- It does not require a tracing vendor or a generic telemetry platform.
- It does not replace workflow-native state with abstract infrastructure jargon.
- It does not invent a separate jobs system when `migration_id` already exists.
- It does add one new identifier, `discovery_run_id`, because CLI-first discovery otherwise has no durable correlation handle before a snapshot exists.

It does make one hard promise: for the supported Viaduct path, operators and maintainers must be able to correlate request, action, execution state, and next step without reading the codebase or database by hand.
