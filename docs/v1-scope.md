# V1 Scope Definition

This document freezes the Viaduct v1 support promise. It is the reference for roadmap decisions, demo scope, release planning, and design-partner conversations.

The rule is simple:

**Implemented in the repo does not automatically mean supported in v1.**

Viaduct already contains more technical surface area than a credible first release should promise. This document narrows that surface to the minimum real product needed for the chosen wedge:

**VMware-exit mixed-estate discovery and migration readiness assessment with approval-ready pilot planning**

For v1, "mixed-estate" means Viaduct can help an operator inspect and reason about an estate that spans more than one platform.

It does **not** mean equal live migration support across many source and target combinations.

## 1. Product Objective For V1

Viaduct v1 should let a VMware-exit team go from estate discovery to an approval-ready first migration wave, with enough operator visibility and supervised pilot controls to run a small pilot without pretending the product is already a zero-touch migration factory.

In plain terms, v1 is successful if an operator can:
- discover the relevant estate
- understand workload and dependency risk
- define and validate the first migration wave
- produce stakeholder-ready summary, migration, and audit outputs
- execute a supervised pilot with explicit approvals, checkpoints, and rollback visibility

## 2. Target Users For V1

### Primary Users
- platform and infrastructure leads responsible for VMware-exit planning
- virtualization operators building the first migration wave and validating readiness
- technical design partners willing to run supervised pilot migrations

### Secondary Users
- engineering managers or architecture leads reviewing readiness, risk, and pilot status
- security or operations reviewers who need audit and change evidence

### Not The Target User For V1
- MSPs looking for a finished multi-customer operating product
- greenfield platform builders with no migration pressure
- buyers looking only for lifecycle analytics without a migration-readiness problem

## 3. Supported Use Cases For V1

V1 supports these use cases:

1. Discover a VMware-led estate and normalize inventory into a common model.
2. Inspect dependency, backup, and migration-readiness signals that affect first-wave selection.
3. Discover likely target inventory where needed for planning.
4. Define workloads, exclusions, network/storage mappings, approval requirements, execution windows, and waves in a declarative migration spec.
5. Run plan validation and preflight checks before execution.
6. Produce an approval-ready first-wave pilot plan and operator-facing status outputs.
7. Run a supervised pilot migration for the supported live migration motion with explicit approvals, checkpoint-aware resume support, and rollback visibility.

Anything broader than those use cases should be treated as post-v1 unless it is required to make the above path work cleanly.

### Single Supported Live Migration Motion

The only formal live migration motion in v1 is:

**VMware vSphere source to Proxmox VE target**

Everything else should be treated as one of the following:
- evaluation-only
- planning context only
- implemented but unsupported in the v1 promise
- future work

## 4. Supported Source Platforms For V1

### Core Supported Source
- VMware vSphere
  Use in v1: primary source platform for the VMware-exit wedge, including discovery and migration-readiness planning.

### Evaluation And Demo Source
- KVM/libvirt fixture lab
  Use in v1: packaged evaluation and demo path only. This is a real repo workflow, but it is not the headline production source story for the wedge.

### Implemented But Not Part Of The V1 Support Promise
- Proxmox as a source
- Hyper-V as a source
- Nutanix as a source
- community plugin connectors as a source

These may exist in the repo and remain useful internally, but they are not part of the v1 support contract.

## 5. Supported Target Platforms For V1

### Single Supported Live Target
- Proxmox VE
  Use in v1: primary named pilot target for teams that want a concrete VMware-exit landing zone.

### Evaluation And Demo Target Context
- KVM/libvirt
  Use in v1: evaluation and engineering validation only. Keep it in the demo and repo-validation story, but do not present it as a separate supported live target motion.

### Not Supported As V1 Targets
- KVM/libvirt as a formal live pilot target promise
- Hyper-V
- Nutanix AHV
- plugin-defined targets as part of the formal v1 promise

Those paths can remain implemented or exploratory, but v1 should not claim them as supported target motions.

## 6. Supported Operator Workflows For V1

These workflows must work well enough to be considered part of v1:

1. Discovery through the CLI
   `viaduct discover` must support the v1 source story and save snapshots to the configured state store.
2. Inventory and readiness review through the API and dashboard
   `/api/v1/inventory`, `/api/v1/snapshots`, `/api/v1/graph`, `/api/v1/summary`, and the corresponding dashboard views must expose enough context for first-wave decisions.
3. First-wave planning through declarative specs
   `viaduct plan` plus the migration spec model must support selectors, mappings, approvals, windows, and waves for the VMware-to-Proxmox path.
4. Preflight validation through the API and dashboard
   `/api/v1/preflight` and the migration workflow UI must expose actionable preflight results before execution.
5. Approval-ready pilot handoff
   The operator must be able to hand an internal reviewer a concrete first-wave plan, current summary, and audit trail without exporting data manually from the codebase.
6. Supervised pilot execution through explicit API or CLI action
   `viaduct migrate` and `/api/v1/migrations` plus `/execute` must support operator-triggered pilot runs for the supported live motion.
7. Monitor, resume, and roll back
   `viaduct status`, `/api/v1/migrations`, `/api/v1/migrations/<id>`, `/resume`, and `/rollback` plus the dashboard views must expose enough state to control a pilot run safely.

### Workflow Boundary For V1

For v1, the CLI and API are the source of truth for workflow state.

The dashboard is required as an operator surface, but it must stay faithful to persisted backend state rather than introducing frontend-only execution logic.

V1 does not require every workflow to be equally polished across CLI, API, and dashboard. It does require the dashboard to accurately reflect the supported backend workflow for the core assessment-to-pilot path.

## 7. Required Reports And Exports For V1

These outputs are required for v1 because they help an operator justify and control the first-wave decision.

### Required Operator Reports
- `/api/v1/reports/summary` in JSON and CSV
- `/api/v1/reports/migrations` in JSON and CSV
- `/api/v1/reports/audit` in JSON and CSV

### Required API Exports
- `/api/v1/inventory` and `/api/v1/snapshots` data in JSON
- `/api/v1/summary` in JSON
- `/api/v1/migrations/<id>` state and checkpoint detail in structured API responses
- `/api/v1/preflight` results in structured API responses

### Required Operator-Ready Evidence
- last discovery time and workload counts
- platform distribution and migration counts
- current migration phase, approval state, and checkpoint state
- audit events with actor, action, outcome, and request correlation

If a report is not useful for first-wave readiness, approval, or pilot control, it is not required for v1.

V1 does not require PDF exports, presentation-grade executive reporting, or custom report builders.

## 8. Must-Have Capabilities

These are the capabilities that make v1 real:

- VMware discovery that reliably normalizes source inventory
- Proxmox target discovery and planning inputs for the supported live migration motion
- dependency-aware inventory sufficient to choose and sequence the first migration wave
- backup exposure context sufficient to identify workloads that need pilot-planning caution
- declarative migration spec parsing and validation
- actionable preflight checks for the VMware-to-Proxmox first-wave path
- explicit approval gates and execution windows
- checkpoint-aware execution state, resume support, and rollback visibility
- operator access through CLI, API, and dashboard for the core assessment-to-pilot workflow
- tenant-scoped auth sufficient for a direct-use pilot deployment
- summary, migration, and audit reporting suitable for operator review
- stable documented contract for the supported planning, execution, resume, rollback, and reporting routes
- packaged release artifacts, install docs, upgrade docs, rollback docs, and OpenAPI/support-matrix alignment

## 9. Nice-To-Have Capabilities

These are useful but should not block v1 if the core path is strong:

- richer dependency visualization polish
- cost, policy, and drift views beyond the minimum readiness context needed for first-wave decisions
- broader source discovery beyond the v1 supported source promise
- additional CSV or report formats beyond the current JSON and CSV exports
- smoother packaged demo assets for VMware-exit storytelling
- service-account support for automation-heavy pilot teams
- stronger dashboard convenience flows that sit on top of existing API behavior
- deeper live-environment certification beyond the first supported migration motion

## 10. Explicitly Out-Of-Scope Items

These are out of scope for v1 even if parts already exist in the repo:

- claiming equal migration support across VMware, Proxmox, Hyper-V, KVM, Nutanix, and plugin connectors
- promising production-ready autonomous migration across every supported connector pair
- promising multiple supported live target motions in the first release
- MSP-first multi-customer operating workflows as the lead product story
- tenant administration polish beyond what is needed for a direct-use pilot deployment
- plugin ecosystem maturity as part of the v1 support promise
- warm migration as a required or marketed v1 capability
- lifecycle optimization as a standalone v1 buying reason
- policy auto-remediation or lifecycle automation as a core release commitment
- broad backup portability execution as a primary v1 buying reason
- enterprise-scale fleet orchestration claims beyond supervised first-wave and pilot workflows
- destination-agnostic "migrate anything anywhere" messaging

## 11. Demo-Only Or Mocked Items That Must Not Be Misrepresented

These must be called out honestly in demos, docs, and release discussions:

- the local KVM fixture lab is a real evaluation path, but it is still a fixture-driven lab path, not a live VMware-exit pilot
- fixture-backed connector certification is not the same thing as live-environment runtime certification
- soak coverage and simulated migration flows are not proof of broad production migration readiness
- implemented connectors outside the v1 support promise must not be sold as equal first-release commitments
- warm migration, backup portability, lifecycle remediation, and plugin support may exist in the repo, but they are not the core proof point for the v1 beachhead

## 12. Exit Criteria For Calling V1 "Real Enough"

Viaduct should only be called v1-ready when all of the following are true:

### Repository Release Gate

1. `README.md`, `docs/getting-started/quickstart.md`, `docs/operations/migration-operations.md`, `docs/reference/support-matrix.md`, and this scope document describe the same one-source, one-live-target v1 promise.
2. The discovery to plan to preflight to approval-ready pilot path works end to end through the supported CLI, API, and dashboard surfaces.
3. `/api/v1/reports/summary`, `/api/v1/reports/migrations`, and `/api/v1/reports/audit` produce usable operator outputs in JSON and CSV.
4. The supported pilot path can be executed, resumed, and rolled back with explicit approval and checkpoint visibility.
5. The supported planning, execution, resume, rollback, and reporting routes have stable documented contracts and passing repo verification coverage appropriate to the release gate.
6. Anything outside the v1 support promise is clearly labeled as future, exploratory, or non-core.

### Field Validation Gate

1. At least one real pilot or design-partner-style environment has validated the VMware-source first-wave workflow beyond the local fixture lab.
2. That validation includes the supported live motion, not just generic discovery or a fixture-backed demo.
3. The pilot evidence is strong enough that Viaduct can honestly claim "supervised first-wave pilot ready" without implying fleet-wide production automation.

If those conditions are not met, Viaduct is still an evaluation build or pilot candidate, not a disciplined v1.

## 13. Risks And Assumptions

### Assumptions
- VMware-exit remains the strongest urgent buyer problem.
- early design partners are willing to accept supervised pilot execution before full automation breadth
- Proxmox VE is the best named initial target because it creates a concrete landing story without requiring a broader target promise
- KVM remains valuable as the packaged evaluation and engineering validation path
- tenant API key authentication is enough for the first direct-use pilots, even if service-account-heavy automation comes later

### Risks
- the repo's strongest current demo flow is still KVM-lab-based, which can weaken the VMware-exit message if not handled carefully
- the phrase "mixed-estate" can create false expectations of equal live support across many sources and targets unless the one supported live motion stays explicit
- the current product surface may keep dragging lifecycle and multi-platform breadth back into the v1 conversation unless this document is used aggressively
- implemented non-v1 connectors and features may create roadmap pressure and support confusion
- lifecycle, backup portability, plugin, and multi-tenant stories can distract from the first-wave planning path if not intentionally deprioritized
- if the supported pilot path is not validated in a real environment, the v1 label will not be credible

## Recommended Use Of This Document

Use this document to stop these recurring mistakes:
- adding roadmap items just because code exists somewhere in the repo
- broadening the public story to match implementation breadth instead of support confidence
- calling a feature "v1" when it does not directly strengthen the assessment-to-pilot workflow
- letting demos center on impressive surfaces that are not part of the core release promise
