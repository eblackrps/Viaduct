# 15-Minute Demo Card

Use this for serious operator or design-partner demos.

## Audience

- design-partner evaluation calls
- platform or virtualization operators
- technical decision-makers
- technical architecture reviewers

## Goal

Create operator trust:

- the workflow is real
- the support boundary is honest
- the product can carry a team from assessment to an approval-ready first wave

## Exact Screen Order

1. `Settings`
2. prepared terminal
3. `Inventory`
4. `Workload detail`
5. `Migrations` `Scope`, `Prepare`, `Validate`
6. `Migrations` `Execute` + `Migration Progress`
7. `Reports`

## Timebox

| Time | Screen | Show |
| --- | --- | --- |
| 0:00-1:15 | `Settings` | tenant, auth mode, permissions, operator connection |
| 1:15-2:15 | prepared terminal | one completed discovery command line or copyable command, no live waiting |
| 2:15-5:00 | `Inventory` | status, discovery context, assessment notes, select ready candidate |
| 5:00-6:30 | `Workload detail` | readiness, dependency, activity, why this workload belongs in the first wave |
| 6:30-10:30 | `Migrations` | planning workflow, target prep, execution controls, preflight, runbook |
| 10:30-13:00 | `Execute` + `Migration Progress` | failed `verify` state, checkpoints, per-workload state, next-action reasoning |
| 13:00-15:00 | `Reports` | summary export, migration history, discovery snapshots, audit evidence |

## What To Say

### 0:00-1:15

"Viaduct is not trying to be a generic virtualization control plane. The current focus is VMware-exit discovery and migration-readiness assessment with approval-ready first-wave planning. So before I show anything else, I want to show the operator context: which tenant we're in, how the dashboard is authenticated, what permissions we have, and what backend we're talking to."

### 1:15-2:15

"Discovery is still CLI-first today, and I want to be explicit about that instead of hiding it. The dashboard is not pretending to own connector setup yet. What matters is that discovery lands in persisted snapshots and the rest of the workflow reads from that same backend truth."

### 2:15-5:00

"Now I'm confirming that we have both the source baseline and the target baseline available, and that the operator can tell how fresh the current view is before planning from it. The key point is that this is an assessment surface, not just an asset list."

### 5:00-6:30

"I'll pick one workload to show the decision logic. In the detail panel we can see the workload's current status, dependency context, and baseline activity clues. That is the bridge from interesting asset to candidate for the first wave."

"If I want to act on this, I can open the migration plan directly from here."

### 6:30-10:30

"This is the real planning workflow: bring scope in, prepare the target and execution controls, validate, then save plan state. The important thing here is that Viaduct keeps the backend as the current reference."

"I'm showing a prepared first-wave draft with explicit target details, execution controls, approvals, and mappings. Then preflight turns that draft into an operational decision. We can see blockers, warnings, and the derived runbook before execution starts."

"That is the real product moment: the operator gets an approval-ready plan, not just a dashboard impression."

### 10:30-13:00

"For a serious evaluation, I also want to show how Viaduct handles supervision when a pilot is not clean. This saved run reached `verify` and failed after earlier checkpoints completed."

"What matters here is that the operator can see the migration ID, current phase, checkpoints, per-workload state, and next-action options from persisted state alone. That is a much more credible early-product story than pretending everything always goes green."

### 13:00-15:00

"Finally, the workflow has to end in evidence. The operator can export summary, migration, and audit outputs directly from the product. That supports internal review and change evidence without pulling data manually from the backend."

"The honest boundary is this: Viaduct is strongest today in discovery, readiness reduction, first-wave planning, and supervised pilot control. That is the product we are intentionally hardening first."

## Do Not Do

- do not open by touring every connector
- do not lead with KVM lab unless explicitly asked for the evaluation path
- do not spend time tweaking forms that do not change the story
- do not drift into lifecycle, plugins, or backup portability before the main path lands
- do not imply the dashboard manages source connections today
- do not imply a prepared failed run is proof of live production certification

## Likely Questions

- How much of this is prepared versus live?
- Can you show the exact VMware and Proxmox discovery path?
- Why is discovery CLI-first rather than in-app?
- What is actually supported today for the named pilot?
- What happens if the API restarts mid-run?
- How do approvals and auditability work?
- What still needs hardening before a team like mine should trust this?

## Suggested Answers

- "We use prepared state in the demo so we can focus on operator reasoning instead of environment latency."
- "The named v1 live motion is VMware vSphere source to Proxmox VE target."
- "Discovery is CLI-first today because we have not productized an in-app connection manager yet."
- "Approvals, audit, and reporting are part of the core trust story, not a later add-on."
- "There are still hardening gaps. For example, execute and resume durability are part of the current reliability-hardening work, which is why we present this as supervised pilot control rather than mature fleet automation."
