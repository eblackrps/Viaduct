# Demo Scenario Brief

Use this to stage one repeatable Viaduct demo environment.

This is the canonical prepared state behind both [3-minute-demo-card.md](three-minute-demo-card.md) and [fifteen-minute-demo-card.md](fifteen-minute-demo-card.md).

## Purpose

- keep the short and long demos on the same story
- prevent presenters from improvising seeded state
- make the VMware-exit assessment and supervised pilot lane feel consistent across calls

## Canonical Demo Lane

- initial focus: VMware-exit assessment leading to an approval-ready first wave
- named live-motion story: VMware vSphere source to Proxmox VE target
- operator promise: supervised pilot control, not fleet-wide autonomous migration
- current reference: persisted backend state already loaded before the demo starts

## Canonical Tenant And Identity

- tenant: `vmware-exit-demo`
- dashboard credential: named service account `demo-operator`
- service account role: `operator`
- explicit permissions:
  - `inventory.read`
  - `migration.manage`
  - `reports.read`
  - `tenant.read`
- fallback only: tenant API key for emergency access, not normal demo use

## Canonical Workloads

### `web-portal-01`

- status: `ready`
- risk: low
- role in story: first-wave candidate that looks easy to justify
- required visible cues:
  - clean readiness state
  - simple network and storage placement
  - no unresolved dependency warning dominating the panel

### `orders-app-02`

- status: `needs-review`
- risk: medium
- role in story: included in the first-wave draft, but requires operator judgment
- required visible cues:
  - dependency or mapping complexity is visible
  - at least one warning or review cue is visible before execution
  - this is the workload that later fails `verify` in the prepared pilot run

### `sql-legacy-01`

- status: `blocked`
- risk: high
- role in story: clear example of what Viaduct should not pull into the first wave
- required visible cues:
- blocked or high-risk status is obvious
  - the reason looks operationally plausible, for example snapshot, backup, or policy friction

## Canonical Screen State

| Screen | Must Be Visible | Why It Matters |
| --- | --- | --- |
| `Settings` | tenant `vmware-exit-demo`, named service account, operator-scoped permissions, backend/build context | proves this is a tenant-scoped workflow, not an anonymous lab page |
| `Inventory` | recent VMware and Proxmox baselines, non-zero status counts, the three named workloads | establishes that Viaduct is an assessment view, not just an asset list |
| `Workload detail` | `web-portal-01` selected with readiness, dependency, and baseline clues | shows why a workload belongs in a first wave |
| `Migrations` `Scope` / `Prepare` / `Validate` | saved first-wave draft containing `web-portal-01` and `orders-app-02`, with `sql-legacy-01` excluded | shows that planning is persisted and selective |
| `Migrations` `Execute` + `Migration Progress` | prepared run in `failed` state with earlier checkpoints complete and `verify` failed | shows supervised pilot control when the run is not clean |
| `Reports` | summary export, migration export, audit export, migration history, discovery snapshots | shows that the workflow ends in evidence, not screenshots |

## Inventory Status

Keep the status cards stable across the full session. If the dashboard shows top-line counts, use one fixed set for every serious demo:

- `ready`: 6
- `needs review`: 3
- `blocked`: 1

The exact totals are less important than consistency. Do not let the presenter see one set of counts on `Inventory` and a contradictory state later in `Migrations`.

## Canonical Planning State

The saved first-wave draft should show:

- included workloads:
  - `web-portal-01`
  - `orders-app-02`
- explicitly not included:
  - `sql-legacy-01`
- one visible blocker in preflight:
  - target network mapping for `orders-app-02` is incomplete
- two visible warnings in preflight:
  - `orders-app-02` has dependency or traffic-review complexity
  - operator approval is still required before the wave is ready to execute

This state is for the planning story. It should stop short of pretending the draft is already safe to run.

## Canonical Failed Pilot Story

The prepared pilot run should represent the same wave after the operator resolved the planning blocker and started execution.

Use this exact failure pattern:

- migration state: `failed`
- completed checkpoints:
  - `export`
  - `convert`
  - `configure`
- failed checkpoint:
  - `verify`
- successful workload:
  - `web-portal-01`
- failed workload:
  - `orders-app-02`
- visible reason:
  - post-cutover boot or service validation did not pass within the expected window

This is the right demo state because it creates a serious operator question: resume, investigate, or roll back. It is much more credible than showing a perfect green run every time.

## Presenter Guardrails

- do not change the named tenant during the demo
- do not switch to a tenant API key unless something is broken
- do not run live discovery unless the environment is exceptionally stable and the audience asked for it
- do not include `sql-legacy-01` in the first wave just to show more motion
- do not improvise new blockers or warnings that contradict the saved story
- do not claim the prepared failed run proves production-certified runtime durability

## Fast Pre-Demo Check

Before a call, confirm these exact items:

- `Inventory` shows the three named workloads with the expected status spread
- `Workload detail` opens directly on `web-portal-01`
- `Migrations` shows the saved first-wave draft with one blocker and two warnings
- `Migration Progress` shows `verify` failed for `orders-app-02`
- `Reports` can export summary, migrations, and audit outputs without waiting on background setup

If any one of those is missing, fix the environment before the demo instead of rewriting the story live.
