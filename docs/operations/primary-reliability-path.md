# Primary Reliability Path

This document defines the one end-to-end workflow Viaduct should treat as the primary reliability path for post-Phase-5 hardening.

This is not a strategy memo and it is not a broad product wishlist.

It is an execution artifact for maintainers, product leads, and engineers deciding what to harden first, what to test first, what to demo first, and what to refuse to broaden until the core path is dependable.

## 1. Current-State Summary

### What Exists Today

Viaduct already has enough product surface to support one credible end-to-end operator path:

- discovery through the CLI and persisted snapshots
- tenant-scoped API auth and state
- dashboard surfaces for `Settings`, `Inventory`, `Migrations`, `Reports`, and `Dependency Graph`
- workload detail and inventory-to-plan handoff in `web/src/features/inventory/`
- a staged migration workflow in `web/src/features/migrations/`
- declarative migration specs, preflight, approval gates, execution windows, checkpoints, resume, and rollback in `internal/migrate/`
- audit, summary, and migration reporting through the API and dashboard exports

Relevant current repo surfaces:

- CLI and API workflow guide: `docs/operations/migration-operations.md`
- product wedge and narrowing: `docs/early-product-focus.md`
- v1 support boundary: `docs/v1-scope.md`
- contract hardening priorities: `docs/backend-contract-hardening.md`
- inventory surface: `web/src/features/inventory/InventoryPage.tsx`
- workload detail: `web/src/features/inventory/WorkloadDetailPanel.tsx`
- migration workflow: `web/src/features/migrations/MigrationWorkflow.tsx`
- migration workspace state: `web/src/features/migrations/useMigrationWorkspace.ts`
- runtime/auth visibility: `web/src/features/settings/SettingsPage.tsx`
- reports surface: `web/src/features/reports/ReportsPage.tsx`
- API workflow routes: `internal/api/server.go`

### What Prior Phases Likely Improved

Prior phases likely delivered the prerequisites that make this path worth hardening now:

- a real shared backend instead of dashboard-only state
- persisted migration state with checkpoints and resume behavior
- approval and execution-window controls
- tenant-aware auth, audit, reporting, and metrics
- a product-grade dashboard shell rather than a throwaway demo UI
- release-gate and packaging discipline

### What Is Still Weak, Ambiguous, Or Risky

The current product still has reliability gaps that directly affect this path:

- discovery is CLI-first; there is no first-class in-product "connect source" flow
- inventory readiness is still partially composed in the frontend rather than served as a single backend contract
- inventory-to-plan handoff starts as a local planning draft in `inventoryPlanningDraft.ts`
- execute and resume still depend on the in-memory `Server.specs` cache in `internal/api/server.go`
- inventory and summary contracts still do not expose enough source provenance for operator trust
- the strongest repeatable repo rehearsal path is the KVM lab, while the supported live motion is VMware to Proxmox

### What Should Be Preserved

This path must preserve the current Viaduct direction:

- current architecture
- current route families
- CLI, API, and dashboard sharing one backend truth
- declarative specs as the planning and execution model
- tenant-scoped state and auth
- approval, checkpoint, resume, and rollback behavior
- the explicit v1 promise of VMware source to Proxmox target for live supervised pilot work

### Smallest Credible Next Move

The smallest credible move is to freeze one primary path and use it to drive:

- contract hardening
- UX tightening
- release criteria
- test coverage
- demo scripts
- manual pilot validation

Viaduct does not need ten polished paths. It needs one path that the team can defend.

## 2. Path Selection Reasoning

## Selected Path

**Tenant-scoped VMware vSphere source to Proxmox VE target first-wave supervised pilot**

The canonical operator sequence is:

1. establish tenant and auth context
2. provide source and target discovery inputs
3. discover source and target inventory
4. review inventory
5. inspect a candidate workload
6. create a first-wave migration plan
7. validate with preflight
8. execute with explicit approval
9. monitor, resume, or roll back
10. export reports and audit evidence

## Why This Path Should Be Hardened First

This path is the best first reliability target because it is the highest-overlap path across:

- the stated beachhead
- the explicit v1 support promise
- real operator value
- real demo value
- the repo's existing workflow surfaces

It is the only path that simultaneously:

- starts from the VMware-exit problem Viaduct is positioned around
- ends in a supervised live motion Viaduct explicitly names in `docs/v1-scope.md`
- touches the main operator surfaces without demanding equal maturity across every feature area
- produces useful outputs even before broad automation trust exists

## Paths That Are Important But Not Primary

- KVM lab is the primary rehearsal path, not the primary product path
- multi-platform discovery breadth is supporting context, not the first reliability promise
- lifecycle and backup portability remain supporting signals, not the core end-to-end proof path
- any other live migration motion is explicitly outside the first hardening target

## 3. Canonical Path At A Glance

| Step | Primary Operator Surface | Source Of Truth | Release-Blocking Outcome |
| --- | --- | --- | --- |
| 0. Tenant and auth context | Dashboard `Settings` + API | `/api/v1/about`, `/api/v1/tenants/current` | Operator knows which tenant, auth mode, role, and API instance are active |
| 1. Discovery input setup | CLI + docs/config | config, credential refs, connector flags | Operator can run discovery without editing code |
| 2. Source and target discovery | CLI | persisted snapshots | VMware and Proxmox snapshots saved to the correct tenant |
| 3. Inventory review | Dashboard `Inventory` | `/api/v1/inventory`, `/api/v1/summary`, `/api/v1/snapshots` | Operator can trust the current inventory baseline enough to plan from it |
| 4. Workload inspection | Dashboard `Inventory` detail + `Dependency Graph` | inventory + graph + lifecycle routes | Operator can justify first-wave inclusion or exclusion for one workload |
| 5. Plan creation | Dashboard `Migrations` | persisted migration state | Plan exists in backend state, not only local draft state |
| 6. Validation | Dashboard `Migrations` + API | `/api/v1/preflight` | Operator can distinguish blocking failures from warnings |
| 7. Execute | Dashboard `Migrations` + API | `/api/v1/migrations/{id}/execute` + persisted migration state | Approved run is accepted or cleanly blocked with explicit reason |
| 8. Monitor / resume / rollback | Dashboard `Migrations` + API | `/api/v1/migrations`, `/api/v1/migrations/{id}`, `/resume`, `/rollback` | Operator can safely decide next action from persisted state alone |
| 9. Report and handoff | Dashboard `Reports` + API | `/api/v1/reports/*`, `/api/v1/audit` | Operator can export stakeholder-ready evidence without manual DB access |

## 4. Step-By-Step Workflow Definition

Each step below defines the current implementation, success criteria, failure modes, required contracts, required UX states, backend expectations, and the immediate hardening implication.

## Step 0. Establish Tenant And Auth Context

### Current Implementation

- dashboard runtime context in `web/src/features/settings/SettingsPage.tsx`
- API build metadata and tenant context through `/api/v1/about` and `/api/v1/tenants/current`
- tenant key and service-account key handling in `web/src/api.ts`

### Success Criteria

- operator can identify the active tenant
- operator can identify the auth method in use
- operator can identify the current API build and store backend
- required permissions are visible before planning or execution

### Failure Modes

- missing credentials
- invalid credentials
- correct tenant but insufficient permission
- operator cannot tell if the dashboard points at the intended API

### Required Data And Contracts

- `GET /api/v1/about`
- `GET /api/v1/tenants/current`
- structured auth errors with request IDs

### UX States Required

- loading runtime context
- authenticated and scoped
- unauthenticated
- permission denied
- backend unavailable

### Backend Expectations

- auth failures are structured and request-correlated
- tenant role and effective permissions are explicit
- runtime metadata is always safe to expose and easy to interpret

### Immediate Hardening Implication

- do not add planning or execution shortcuts that bypass visible tenant/auth context

## Step 1. Provide Source And Target Discovery Inputs

### Current Implementation

- CLI discovery commands documented in `docs/operations/migration-operations.md`
- config and connector resolution behavior in the backend
- no dashboard-managed source or target connection objects

### Success Criteria

- operator knows which VMware source and Proxmox target are in scope
- discovery can be run without editing repo files
- credentials remain externalized through config or environment

### Failure Modes

- connector address ambiguity
- missing or wrong credential reference
- operator expects the dashboard to manage connectors when it does not

### Required Data And Contracts

- CLI discovery flags
- documented address formats
- documented credential-resolution behavior

### UX States Required

- this step is doc-driven today, not product-driven
- `Settings` should be treated as the runtime confirmation surface, not a connection manager

### Backend Expectations

- discovery config resolution is explicit and deterministic
- secrets never leak into reports, planning drafts, or saved migration metadata

### Immediate Hardening Implication

- keep docs honest; do not imply Viaduct already has a productized "connect source" screen

## Step 2. Discover VMware And Proxmox Inventory

### Current Implementation

- `viaduct discover --type vmware --source ... --save`
- `viaduct discover --type proxmox --source ... --save`
- persisted snapshots surfaced through `/api/v1/snapshots`
- merged inventory surfaced through `/api/v1/inventory`

### Success Criteria

- source snapshot saved
- target snapshot saved
- snapshots are tenant-scoped
- dashboard inventory reflects the latest snapshots consistently

### Failure Modes

- auth/connectivity failure
- partial discovery with missing infrastructure context
- snapshot stored under the wrong tenant
- stale snapshot treated as current without enough provenance

### Required Data And Contracts

- `/api/v1/inventory`
- `/api/v1/snapshots`
- tenant-scoped snapshot persistence
- latest-snapshot merge semantics

### UX States Required

- no inventory
- loading
- inventory current
- inventory partial
- discovery baseline visible

### Backend Expectations

- discovery remains read-only
- merged inventory is deterministic
- source provenance becomes explicit enough for operator trust

### Immediate Hardening Implication

- add source provenance and partial-data semantics to inventory and summary before broadening discovery stories

## Step 3. Review Inventory

### Current Implementation

- dashboard `Inventory` page in `web/src/features/inventory/InventoryPage.tsx`
- posture signals derived from inventory, graph, policies, and remediation

### Success Criteria

- operator can identify workload counts, risk counts, and baseline timing quickly
- operator can tell whether the current view is complete or partial
- operator can move directly into workload inspection or planning

### Failure Modes

- posture is partial but not clearly signaled
- baseline timing is unclear
- inventory appears trustworthy even when major inputs are missing

### Required Data And Contracts

- `/api/v1/inventory`
- `/api/v1/summary`
- source provenance and baseline timing

### UX States Required

- loading
- unavailable
- empty
- current
- partial

### Backend Expectations

- inventory and summary remain consistent
- the frontend should not permanently own readiness logic

### Immediate Hardening Implication

- prioritize a backend readiness contract after provenance hardening

## Step 4. Inspect One Candidate Workload

### Current Implementation

- workload detail panel in `web/src/features/inventory/WorkloadDetailPanel.tsx`
- graph navigation through the `Dependency Graph` route

### Success Criteria

- operator can inspect overview, dependency, risk, and recent context for one workload
- operator can decide "include", "exclude", or "investigate further"
- operator can open planning directly from the workload view

### Failure Modes

- graph or lifecycle signals missing without explanation
- risk state changes between screens without clear cause
- activity context is too thin for first-wave confidence

### Required Data And Contracts

- inventory workload fields
- graph relationships
- policy and remediation signals

### UX States Required

- no workload selected
- detail available
- partial signals available
- explicit action to open graph
- explicit action to open migration plan

### Backend Expectations

- workload identity is stable from inventory into planning
- dependency and readiness gaps are not hidden by optimistic UI labels

### Immediate Hardening Implication

- use the future readiness contract to reduce per-workload inference drift

## Step 5. Create The First-Wave Migration Plan

### Current Implementation

- inventory selection creates a local planning draft
- `Open migration plan` routes into the `Migrations` workflow
- staged workflow implemented in `web/src/features/migrations/MigrationWorkflow.tsx`
- local workflow state managed in `web/src/features/migrations/useMigrationWorkspace.ts`
- persisted plan creation through `POST /api/v1/migrations`

### Success Criteria

- operator can carry selection from inventory into migrations
- source and target endpoints are explicit in the plan
- mappings, approvals, windows, and waves are configurable
- a persisted migration plan exists in backend state

### Failure Modes

- local draft diverges from persisted plan
- selected workloads do not match saved scope
- required source or target context is hidden in the UI
- plan appears saved when it is still only a local draft

### Required Data And Contracts

- stable workload identity handoff
- `POST /api/v1/migrations`
- stable `MigrationState` and `MigrationPlan`

### UX States Required

- no scope
- draft loaded
- plan editing
- plan saved
- plan stale

### Backend Expectations

- persisted plan state is the source of truth
- spec validation is structured and field-addressable

### Immediate Hardening Implication

- keep local planning draft as a convenience only; do not let it become a hidden source of truth

## Step 6. Validate Through Preflight

### Current Implementation

- preflight invoked from the migration workflow
- API route `/api/v1/preflight`
- structured invalid-spec errors now available through the API

### Success Criteria

- operator can see pass, warn, and fail clearly
- blocking failures are obvious
- preflight result is clearly tied to the current draft

### Failure Modes

- stale preflight looks current
- warnings and blocking failures are mixed together
- spec validation errors are opaque

### Required Data And Contracts

- `/api/v1/preflight`
- structured `PreflightReport`
- field-level validation errors

### UX States Required

- not run
- running
- passed
- warning
- blocked
- stale

### Backend Expectations

- preflight remains deterministic
- checks continue collecting actionable output

### Immediate Hardening Implication

- treat preflight freshness as part of execution trust, not just a nice-to-have UI signal

## Step 7. Execute With Explicit Approval

### Current Implementation

- execution from the `Migrations` workflow
- command acknowledgement through `/api/v1/migrations/{id}/execute`
- approval and window validation in the API

### Success Criteria

- execution starts only from a persisted plan
- approval requirement is explicit
- execution acknowledgement is structured
- blocked execution returns a clear reason

### Failure Modes

- approval missing
- window not open or already closed
- operator cannot tell whether the run was accepted or just locally initiated
- API restart invalidates execution because plan identity is stored in `Server.specs`

### Required Data And Contracts

- `/api/v1/migrations/{id}/execute`
- stable command acknowledgement
- persisted migration state

### UX States Required

- ready
- approval required
- window blocked
- accepted
- failed to start

### Backend Expectations

- approval and window conflicts are machine-readable
- command acknowledgement is stable
- execute must stop depending on volatile in-memory spec state

### Immediate Hardening Implication

- this is the highest-priority backend hardening item on the path

## Step 8. Monitor, Resume, And Roll Back

### Current Implementation

- migration history and status views in the dashboard
- `/api/v1/migrations`
- `/api/v1/migrations/{id}`
- `/resume`
- `/rollback`

### Success Criteria

- operator can decide next action from persisted state alone
- checkpoint progress is visible
- resume skips completed phases
- rollback results are explicit

### Failure Modes

- list and detail views disagree
- checkpoint data is too thin to support safe resume decisions
- rollback appears successful despite partial cleanup or restoration failure

### Required Data And Contracts

- stable migration detail schema
- stable list/detail phase semantics
- persisted checkpoint state
- rollback result contract

### UX States Required

- running
- waiting on approval
- blocked
- failed
- resumed
- rolled back
- completed

### Backend Expectations

- migration detail is the stable polling contract
- rollback must not mask partial failure

### Immediate Hardening Implication

- tighten list/detail lifecycle semantics before adding richer orchestration polish

## Step 9. Export Reports And Audit Evidence

### Current Implementation

- dashboard `Reports` page in `web/src/features/reports/ReportsPage.tsx`
- API exports through `/api/v1/reports/summary`, `/migrations`, `/audit`
- direct audit route through `/api/v1/audit`

### Success Criteria

- operator can export summary, migrations, and audit without leaving the product flow
- exports reflect the same tenant-scoped state visible in the dashboard
- exports are usable for pilot review and change evidence

### Failure Modes

- export payloads differ from on-screen state
- audit output lacks actor, outcome, or request correlation
- export failures are opaque

### Required Data And Contracts

- `/api/v1/reports/summary`
- `/api/v1/reports/migrations`
- `/api/v1/reports/audit`
- `/api/v1/audit`

### UX States Required

- loading
- history available
- export running
- export failed
- export complete

### Backend Expectations

- report payloads remain documented and stable
- audit events retain actor, action, outcome, resource, and request ID

### Immediate Hardening Implication

- keep reports aligned with the same lifecycle and approval semantics used elsewhere on the path

## 5. Reliability Checklist

Treat the path as not reliable if any of the following are false:

- there is one recommended operator sequence
- each step has one primary operator surface
- each step has one persisted source of truth
- tenant and auth context are visible before sensitive actions
- source and target identity are explicit before planning and execution
- discovery provenance is visible enough to trust the current baseline
- workload selection survives the jump from inventory to planning
- preflight freshness is explicit
- approval and execution-window state are explicit before execute
- execution starts only from persisted plan state
- monitoring does not require direct store inspection
- resume does not re-run completed phases
- rollback does not hide partial failure
- reports and audit output match operator-visible state
- failures that matter to the operator include a code, message, and request ID
- the same path shape is used in docs, demos, tests, and pilot validation

## 6. Definition Of "Boringly Reliable"

For this path, "boringly reliable" has two bars.

## Repo-Rehearsal Bar

Viaduct is repo-boring for this path when:

- a maintainer can follow the documented path without reading code
- no step requires direct database or store inspection
- the operator always knows whether the system is waiting, blocked, failed, or complete
- the same documented sequence works repeatedly in rehearsal using the lab or equivalent non-production validation setup
- all non-success API responses on the path are structured and request-correlated

## Supported-Pilot Bar

Viaduct is pilot-boring for this path when:

- the same sequence works in a real VMware-to-Proxmox pilot
- the operator can discover, plan, validate, execute, monitor, and export evidence without custom code
- one predictable failure can be handled using only the documented resume or rollback path
- the team can truthfully say "supervised first-wave pilot ready" without implying fleet-wide automation breadth

If the repo rehearsal passes but the supported pilot bar is not met, Viaduct is still a credible pilot candidate, not a fully hardened early product.

## 7. Recommended Implementation Order

This order is intentionally strict. It prioritizes the items that most directly reduce operator ambiguity or execution fragility.

## Work Package 1. Durable Execute And Resume Identity

### Why First

The current execute/resume path still depends on `Server.specs` in `internal/api/server.go`. That is the weakest reliability point on the primary path because it breaks trust after restarts.

### Repo Areas

- `internal/api/server.go`
- `internal/migrate/`
- `internal/store/`
- `docs/reference/openapi.yaml`

### Acceptance

- execute and resume work after API restart for a saved plan
- no command step depends on in-memory spec lookup

## Work Package 2. Discovery Provenance In Inventory And Summary

### Why Second

Operators cannot safely plan from inventory they cannot attribute to a specific saved baseline or source set.

### Repo Areas

- `internal/api/server.go`
- `internal/models/`
- `web/src/features/inventory/InventoryPage.tsx`
- `docs/reference/openapi.yaml`

### Acceptance

- inventory and summary show source descriptors and latest snapshot provenance
- partial-data state is explicit

## Work Package 3. Backend Readiness Contract

### Why Third

The current inventory review and workload inspection path still relies too heavily on frontend composition.

### Repo Areas

- `internal/api/`
- `internal/lifecycle/`
- `internal/deps/`
- `web/src/features/inventory/`
- `docs/reference/openapi.yaml`

### Acceptance

- one backend readiness contract explains review-required and blocked states
- inventory and workload screens stop inferring the critical gate logic independently

## Work Package 4. Tighten Inventory-To-Plan Handoff

### Why Fourth

The local planning draft is useful, but it is still a reliability risk if maintainers start treating it as more authoritative than persisted plan state.

### Repo Areas

- `web/src/features/inventory/inventoryPlanningDraft.ts`
- `web/src/features/migrations/useMigrationWorkspace.ts`
- `web/src/features/migrations/MigrationWorkflow.tsx`

### Acceptance

- the UI makes it obvious when the operator is looking at local draft state versus persisted plan state
- persisted plan state wins every conflict

## Work Package 5. Converge Monitor And Report Semantics

### Why Fifth

Execution is not boring if `Migrations`, `Reports`, and audit views disagree on lifecycle meaning.

### Repo Areas

- `internal/api/server.go`
- `internal/api/reports.go`
- `web/src/features/migrations/`
- `web/src/features/reports/`

### Acceptance

- lifecycle state, approval state, checkpoints, and failure state are consistent across list, detail, and report surfaces

## Work Package 6. Validation Assets And Release Gate Alignment

### Why Sixth

The path is not real if docs, demos, tests, and packaged flows use different sequences.

### Repo Areas

- `tests/integration/`
- `docs/operations/`
- `README.md`
- `QUICKSTART.md`
- `docs/reference/support-matrix.md`

### Acceptance

- the same path shape appears in docs, integration coverage, and release validation guidance

## 8. Validation Plan

## API And Contract Tests

Add or expand tests for:

- `/api/v1/about` and `/api/v1/tenants/current` auth-context reliability
- `/api/v1/inventory` provenance and partial-data semantics
- `/api/v1/preflight` freshness and validation semantics
- `/api/v1/migrations/{id}/execute` and `/resume` durable command behavior
- `/api/v1/migrations/{id}` lifecycle and checkpoint contract
- `/api/v1/reports/*` and `/api/v1/audit` lifecycle-aligned output

Suggested repo locations:

- `internal/api/*_test.go`
- `tests/integration/`

## End-To-End Test Case Outlines

### Path Happy Case

- create/select tenant
- discover VMware source
- discover Proxmox target
- load inventory
- inspect one workload
- create plan
- run preflight
- execute with approval
- monitor to completion
- export summary, migrations, and audit

### Path Blocked Case

- create plan with approval required
- attempt execute without approval
- verify explicit block state
- add approval
- execute successfully

### Path Resume Case

- interrupt after persisted checkpoint
- resume
- verify completed phases are skipped

### Path Rollback Case

- force execution failure after partial progress
- trigger rollback
- verify rollback result remains explicit if cleanup is incomplete

### Path Partial-Data Case

- save incomplete or stale discovery state
- verify inventory clearly signals partial trust
- verify operator can see why planning confidence is reduced

## Manual Validation Runbook

Use the same shape every time.

1. Start the API server.
2. Confirm tenant and runtime context in `Settings`.
3. Run VMware discovery and save the snapshot.
4. Run Proxmox discovery and save the snapshot.
5. Open `Inventory` and confirm the current baseline is visible.
6. Inspect one workload in the detail panel.
7. Open `Migrations` from inventory selection or workload detail.
8. Set source, target, mappings, approval, window, and wave controls.
9. Run preflight and resolve blockers.
10. Save the plan.
11. Execute with explicit approval metadata.
12. Monitor checkpoints in `Migrations`.
13. Exercise resume or rollback if needed.
14. Export summary, migration, and audit evidence from `Reports`.

Manual validation is not complete if any step requires:

- direct database inspection
- code reading to determine next action
- undocumented cleanup or reset behavior

## 9. Definition Of Done For This Artifact

This document is doing its job only if it helps the team reject weak work.

That means a proposed change should now be easy to classify:

- strengthens the primary reliability path
- neutral supporting work
- outside the current path and should wait

If a change does not make this path more trustworthy, more explicit, or easier to validate, it should not outrank the work packages above.

## 10. Out-Of-Path Work That Should Not Jump The Queue

Do not let these outrank the primary path until the work packages above are complete:

- additional live source-target motion promises
- a productized connection-manager UI
- broader lifecycle automation
- plugin ecosystem polish
- warm migration as the headline path
- prettier reporting without stronger report semantics
- dashboard-only orchestration behavior that is not backed by persisted state
