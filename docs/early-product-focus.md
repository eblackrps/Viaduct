# Early Product Focus

This document defines the current Viaduct product focus after the Phase 4 implementation work. The goal is not to reset the architecture or shrink the repository to a toy. The goal is to make the product promise narrower, more believable, and easier to harden with real operator feedback.

## Why Narrowing Matters Now

Viaduct already has broad technical surface area:
- multi-platform discovery
- dependency mapping
- migration planning and orchestration primitives
- lifecycle analysis
- multi-tenancy and service accounts
- CLI, API, dashboard, packaging, and release gating

That breadth is useful, but it creates a product risk: the repository can imply that every workflow is equally mature and equally ready for live operator trust. Phase 5 should correct that risk by defining a sharper early-product contract, then hardening against that contract instead of expanding the surface area further.

## What Should Be Preserved

The existing direction is worth keeping:
- the universal schema in `internal/models/`
- the shared backend that feeds the CLI, API, and dashboard
- mixed-hypervisor operations as a durable model instead of a temporary bridge
- declarative migration specs, approval gates, execution windows, and resume support
- tenant-scoped state, service accounts, and public operator documentation
- reproducible release verification through `make release-gate`

None of those foundations should be reset for Phase 5. The work now is to narrow claims, strengthen trust, and make one operator workflow feel dependable.

## What Is Still Weak

The current repo still has early-product gaps:
- the public narrative is broader than the most validated operator path
- live-environment confidence is not uniform across connector and execution combinations
- migration execution, lifecycle guidance, and portability features can look equally mature even when the safest current value is earlier in the workflow
- trust controls are present, but the repo needs stronger emphasis on diagnostics, auditability, approval behavior, and operator-visible caveats
- real-user validation loops are still lighter than the technical breadth of the platform

These are not arguments for a rewrite. They are arguments for a more disciplined product boundary.

## Recommended Initial Focus Use Case

Viaduct's best early focus is:

**VMware-exit mixed-estate discovery and migration readiness assessment with approval-ready pilot planning**

See [Initial Use Case Analysis](initial-use-case-analysis.md) for the candidate focus options, scoring criteria, final recommendation, and positioning guidance behind this choice.

In practice, that means helping an operator:
1. discover a mixed estate
2. normalize and inspect inventory
3. understand dependencies, backup coverage, and lifecycle signals
4. draft and validate migration plans
5. run supervised pilot migrations with explicit approvals and rollback visibility

This focus fits the repo better than a broader promise of hands-off production migration across every connector pair. It aligns with the project origin, the current dashboard and API surfaces, the dependency and lifecycle layers, and the operator need to make migration decisions before they trust full execution automation.

## Disciplined V1 Scope

See [V1 Scope Definition](v1-scope.md) for the authoritative must-have, nice-to-have, out-of-scope, and exit-criteria breakdown for the first release.

### In Scope Now
- tenant-scoped discovery and normalized inventory collection
- operator visibility through CLI, API, dashboard, reports, and metrics
- dependency and backup signals, plus the cost, policy, and drift context needed for migration-readiness decisions
- declarative migration specs, dry-run planning, preflight validation, approval gates, execution windows, checkpoints, resume state, and rollback visibility
- packaged evaluation and demo workflows using the local lab, source builds, and release bundles
- explicit release and contract artifacts such as OpenAPI, support matrix, runbooks, and release manifests

### Not The Default Promise Yet
- fully autonomous production cutover across every supported source and target combination
- equal live-environment confidence for every connector-backed execution path
- frontend-inferred workflow state that is not backed by the shared API and persisted state
- policy or lifecycle remediation that silently executes changes without operator review
- ecosystem sprawl that weakens the core evaluation and pilot workflow

## Early-Product Trust Contract

### What Viaduct Can Credibly Stand Behind Today
- evaluation from source or packaged artifacts using the documented quickstart and lab assets
- tenant-scoped inventory, planning, reporting, and operational visibility
- conservative operator workflows that keep planning and approval state explicit
- a shared backend contract across CLI, API, and dashboard
- release-gated builds, public docs, and operator-visible build metadata

### What Still Requires Pilot Framing
- live migration execution in heterogeneous real environments
- connector-specific runtime behavior beyond the current validated lab and fixture coverage
- backup portability execution in environments with non-trivial operational variance
- plugin ecosystem interoperability beyond documented manifest and certification rules
- deployment hardening beyond the provided examples and reference assets

## Primary Workflow To Harden In Phase 5

The product should optimize around one reliable end-to-end operator path:

1. create or select a tenant and use a tenant key or service-account key
2. run discovery against the lab or a supported pilot environment
3. review inventory, dependency graph, snapshot history, and lifecycle posture
4. author a migration spec and run dry-run planning plus preflight validation
5. review approval, execution-window, checkpoint, and diagnostic state through the API and dashboard
6. execute only in a supervised pilot environment with resume and rollback available
7. inspect migration history, audit output, reports, metrics, and packaged artifacts for stakeholder review

This workflow is wide enough to demonstrate real product value, but narrow enough to harden without pretending every route is production-proven.

## Phase 5 Priorities

The next round of work should favor:
- clearer public positioning around assessment, planning, and supervised pilots
- tighter API and error contracts for preflight, planning, execution, resume, rollback, and reporting
- better operator trust surfaces: approval clarity, audit trails, metrics, request correlation, and failure diagnostics
- one boringly reliable evaluation and pilot path that works the same in docs, demos, tests, and packaged artifacts
- design-partner feedback loops that help remove scope rather than add novelty

See [Backend Contract Hardening](backend-contract-hardening.md) for the specific backend and API contract work that should anchor this phase.

## Change Bar For New Work

Before adding new product surface, ask:
- does this make the assessment and supervised pilot workflow more trustworthy?
- does it reduce ambiguity in the operator contract?
- does it strengthen a shared backend capability instead of adding frontend-only behavior?
- does it improve demo, packaging, support, or real-user validation readiness?

If the answer is no, it is probably not Phase 5 work.
