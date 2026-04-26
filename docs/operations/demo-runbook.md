# Demo Runbook

This document defines the first serious demo assets for Viaduct after Phase 5.

It is not a feature tour. It is a constrained demo kit for the current Viaduct focus:

**VMware-exit multi-platform inventory collection and migration readiness assessment with approval-ready pilot planning**

The demos in this document are built around the same path Viaduct is already trying to harden:

1. confirm tenant and auth context
2. confirm discovery baseline
3. review inventory
4. inspect one candidate workload
5. create and validate a first-wave plan
6. review saved execution state
7. export evidence

Use this runbook for:
- founder or maintainer-led demos
- design-partner calls
- serious product evaluation calls
- recorded short demos

Do not use it to imply that every implemented feature in the repo is part of the current product promise.

Companion presenter assets live in [demo/README.md](demo/README.md).

## 1. Current-State Summary

### What Exists Today

Viaduct already has enough real product surface to support two serious demos:

- product focus and support boundary in:
  - `docs/initial-use-case-analysis.md`
  - `docs/v1-scope.md`
- one primary reliability path in:
  - `docs/operations/primary-reliability-path.md`
- dashboard surfaces for:
  - `Settings`
  - `Inventory`
  - `Migrations`
  - `Reports`
  - `Dependency Graph`
- CLI discovery and planning docs in:
  - `docs/operations/migration-operations.md`
- trust and observability framing in:
  - `docs/operations/auth-role-audit-model.md`
  - `docs/operations/observability-requirements.md`

### What Prior Phases Likely Improved

Prior phases likely made these demos possible by delivering:

- a shared backend instead of frontend-only demo state
- persisted migration plans, checkpoints, resume, and rollback state
- tenant-scoped auth, reporting, and audit surfaces
- a product-grade dashboard shell with credible workflow pages
- contract hardening and better API error behavior

### What Is Still Weak, Ambiguous, Or Risky

The current demo story still has real risks:

- the strongest repo rehearsal path is still the KVM lab, not a polished live VMware demo kit
- discovery is still CLI-first, which is honest but can feel less polished if handled poorly
- the repo has broader feature surface than the current focus should demo
- execution maturity must still be framed as supervised pilot control, not zero-touch automation
- demos can easily drift into lifecycle, plugins, or multi-platform breadth that weakens the first-wave story

### What Should Be Preserved

These demos should preserve:

- the VMware-exit focus
- the v1 support promise of VMware source to Proxmox target for the named live motion
- the CLI/API/dashboard split as it exists today
- explicit controls, approvals, checkpoints, and reporting
- honest framing around pilot scope and supported proof

### Smallest Credible Next Move

The smallest credible Step 8 move is:

- one 3-minute demo for clarity and initial interest
- one 15-minute demo for serious operator evaluation
- one recommended seeded state
- one smoothness and credibility checklist

Viaduct does not need ten demo variants yet. It needs two that tell the same story at different depths.

## 2. Demo Strategy Framing

Viaduct should not be demoed as:

- “watch us migrate anything anywhere”
- “one-click hypervisor replacement”
- “a broad virtualization control plane”
- “a lifecycle platform with migration as one tab”

Viaduct should be demoed as:

**the product that helps a VMware-exit team go from environment visibility to an approval-ready first migration wave, with enough trust to supervise a pilot instead of guessing**

### Core Demo Rule

The short demo should create clarity and interest.

The long demo should create trust.

Neither demo should attempt to prove every implemented feature in the repository.

### Demo Narrative Spine

Every Viaduct demo should keep the same narrative spine:

1. we know exactly which tenant and operator context we are in
2. we know what inventory baseline we are working from
3. we can identify a first-wave candidate and explain why
4. we can turn that into a real saved plan with preflight evidence
5. we can show what happens when a pilot run needs supervision
6. we can export stakeholder-ready evidence without manual database access

If a section does not support that spine, it should probably not be in the demo.

## 3. Recommended Seeded Demo State

The compact staging version of this section is in [demo/demo-scenario-brief.md](demo/demo-scenario-brief.md).

Use one prepared tenant-scoped demo environment for both demos.

### Environment

- persistent store, not in-memory
- named tenant, for example `vmware-exit-demo`
- named dashboard credential, preferably a service account
- stable API build and dashboard build
- low-latency environment with the dashboard already running

### Baseline Inventory State

Prepare:

- one VMware source baseline
- one Proxmox target baseline
- enough inventory to look real without becoming visually noisy

Recommended shape:

- 60 to 180 workloads total
- 2 VMware clusters
- 1 Proxmox target environment
- a mix of application, infrastructure, and database workloads
- non-zero visible status counts for `ready`, `needs review`, and `blocked`

### Three Standardized Workload Archetypes

Use the same three named examples in both demos:

1. **Ready candidate**
   Example label: `web-portal-01`
   Desired state in UI: `ready`, low risk, simple network/storage context
2. **Needs-review candidate**
   Example label: `orders-app-02`
   Desired state in UI: `needs-review`, medium risk, mapping or dependency complexity
3. **Deferred workload**
   Example label: `sql-legacy-01`
   Desired state in UI: blocked or high risk because of snapshots, backup relationships, policy warnings, or partial signals

### Saved Migration State

Prepare:

- one saved first-wave plan containing the ready candidate and the needs-review candidate
- one preflight result set with:
  - exactly 1 visible blocker
  - at least 2 visible warnings
  - visible wave output
- one saved failed pilot run with:
  - phase set to `failed`
  - earlier checkpoints completed through `configure`
  - `verify` shown as failed
  - visible checkpoint history
  - visible per-workload state
  - one clear per-workload failure reason that sounds operationally plausible

Recommended storyline:

- blocker: one target network mapping or readiness issue that is easy to understand
- warning 1: dependency or backup review still needed on the medium-risk candidate
- warning 2: execution still requires review before the first wave is considered ready
- failed run: the medium-risk candidate reaches `verify` and fails boot or post-cutover validation, so resume versus rollback becomes a real discussion

### Reports State

Ensure:

- summary export works
- migrations export works
- audit export works
- at least one saved snapshot is visible in `Reports`

### Honesty Rule

If the demo uses:

- sanitized real data
- internally prepared tenant data
- fixture-backed data

say which one it is.

Do not present fixture-backed or synthetic demo data as proof of live pilot runtime maturity.

## 4. Three-Minute Demo

The presenter-ready card for this demo is in [demo/three-minute-demo-card.md](demo/three-minute-demo-card.md).

### Target Audience

Use the 3-minute demo for:

- first meetings with infrastructure or platform leads
- conference or meetup lightning demos
- short founder intros
- top-of-funnel design-partner calls

This audience needs:

- a clear buyer problem
- a believable product shape
- a reason to take the longer demo

### Goal

Create quick clarity:

- what Viaduct is for
- who it is for
- why it is credible now
- why it is not just another dashboard

### Exact Flow

| Time | Screen | Purpose |
| --- | --- | --- |
| 0:00-0:20 | `Inventory` | Show the VMware-exit problem in plain terms immediately |
| 0:20-0:50 | `Workload detail` | Show how one first-wave candidate gets justified |
| 0:50-1:45 | `Migrations` | Show planning, preflight, and saved plan state |
| 1:45-2:20 | `Migrations` execute state | Show supervised pilot control, not blind automation |
| 2:20-2:45 | `Reports` | Show evidence export and handoff |
| 2:45-3:00 | Close on support boundary | State exactly what is real today and invite the longer demo |

### Why This Order

The 3-minute demo should start with the buyer-visible payoff, not the internal trust surface.

`Settings` matters, but it is a trust-building detail for the longer demo. In the short demo, opening on `Inventory` creates faster clarity about what Viaduct actually does.

### What To Show On Screen

#### 0:00-0:20

Show:

- `Inventory` page header
- status cards
- discovery context card
- one visible baseline timestamp

#### 0:20-0:50

Show:

- one selected workload in `Workload detail`
- readiness and risk badges
- dependency section
- activity section with snapshot or baseline context

#### 0:50-1:45

Show:

- `Migrations`
- `Planning workflow`
- preflight summary
- execution runbook or saved plan state

#### 1:45-2:20

Show:

- `Execute` stage
- migration ID
- failed or blocked run state
- checkpoints or per-workload state

#### 2:20-2:45

Show:

- `Reports`
- summary, migrations, and audit exports
- one visible migration-history or discovery-snapshot panel

#### 2:45-3:00

Show:

- stay on `Reports` or return to `Migrations`
- no extra navigation

### What To Say

Use this script closely. Small wording changes are fine, but keep the structure.

#### 0:00-0:20

"Viaduct is for VMware-exit teams that need to turn a multi-platform environment into a credible first migration wave before they trust automation. So I'll start with the workflow payoff: which workloads look ready, which need review, and which are blocked."

#### 0:20-0:50

"Here's one candidate workload. Viaduct pulls readiness, dependency, and baseline context together so the team can justify inclusion in a first wave instead of guessing from disconnected tools."

#### 0:50-1:45

"Once we have a candidate set, Viaduct moves into a real backend planning flow: scope, prepare, validate, and save the plan. Preflight is what turns that draft into something operationally meaningful by separating blockers from warnings before execution."

#### 1:45-2:20

"And if a pilot is not clean, the team can still reason from saved state: migration ID, checkpoints, per-workload status, and next action. The story here is supervised pilot control, not blind automation."

#### 2:20-2:45

"Viaduct also closes the loop with summary, migration, and audit exports, so the team can hand an approver or reviewer real evidence instead of a screenshot deck."

#### 2:45-3:00

"The honest boundary is that Viaduct is strongest today in discovery, readiness reduction, approval-ready planning, and supervised pilot control. If that is your problem, the longer demo goes much deeper."

### What To Avoid

Do not:

- open with `Settings`
- open with lifecycle, policy, drift, or backup portability
- spend time in the CLI
- claim broad unattended-migration maturity
- show multiple source or target motions
- click around randomly between tabs
- dwell on unsupported or secondary features

### Likely Questions Or Objections

Expect:

- “Is this a live VMware demo or seeded data?”
- “How much of this is planning versus real execution?”
- “Why is discovery CLI-first?”
- “What target is actually supported for the named pilot path?”
- “What happens when a migration fails?”

### Fast Answers

- “This demo is focused on the workflow trust path. Discovery is real, but the important point here is how Viaduct turns inventory into a supervised first-wave plan.”
- “The named live motion in v1 is VMware to Proxmox, and we still frame execution as supervised pilot work, not zero-touch fleet automation.”
- “Discovery is still CLI-first today. The dashboard is the operator review and control surface on top of the persisted backend truth.”

## 5. Fifteen-Minute Demo

The presenter-ready card for this demo is in [demo/fifteen-minute-demo-card.md](demo/fifteen-minute-demo-card.md).

### Target Audience

Use the 15-minute demo for:

- serious design-partner evaluation calls
- platform or virtualization operators
- technical decision-makers
- technical architecture review sessions

This audience needs:

- proof that the workflow is serious
- workflow continuity
- honest boundaries
- visible failure handling and evidence export

### Goal

Create trust strong enough that the audience believes:

- Viaduct understands the assessment and supervised pilot workflow
- the current product has a real path from discovery to a supervised first wave
- the team is honest about what is hardened and what is still pilot-scoped

### Exact Flow

| Time | Screen | Purpose |
| --- | --- | --- |
| 0:00-1:15 | `Settings` | Establish trust status, tenant context, and environment |
| 1:15-2:15 | Prepared terminal | Show exact discovery boundary without wasting time |
| 2:15-5:00 | `Inventory` | Confirm baseline, status, and candidate selection |
| 5:00-6:30 | `Workload detail` | Show why one workload belongs in the first wave |
| 6:30-10:30 | `Migrations` `Scope`, `Prepare`, `Validate` | Show how a first-wave plan becomes approval-ready |
| 10:30-13:00 | `Migrations` `Execute` + `Migration Progress` | Show failed `verify` state, checkpoints, and next-action reasoning |
| 13:00-15:00 | `Reports` | Show summary, migration, and audit evidence, then close on support boundary |

### What To Show On Screen

#### 0:00-1:15

Show:

- `Settings`
- dashboard connection
- tenant context
- permissions and quotas

#### 1:15-2:15

Show:

- a prepared terminal with the discovery commands visible
- one completed command history line or copyable command
- no live waiting

#### 2:15-5:00

Show:

- `Inventory` status
- `Discovery context`
- `Assessment notes`
- select the ready candidate

#### 5:00-6:30

Show:

- open `Workload detail`
- keep the focus on readiness, dependency, and activity context

#### 6:30-10:30

Show:

- `Migrations`
- `Planning workflow`
- `Scope`, `Prepare`, and `Validate` sections
- source and target context
- execution controls
- network mappings
- preflight results
- execution runbook / wave plan

#### 10:30-13:00

Show:

- `Execute` stage
- saved plan state
- migration ID
- failed `verify` state
- execution blockers or advisories
- `Migration Progress`
- checkpoints
- per-workload progress or errors

#### 13:00-15:00

Show:

- `Reports`
- `API exports`
- `Migration History`
- `Discovery snapshots`

### What To Say

Use this script structure closely.

#### 0:00-1:15

“Viaduct is not trying to be a generic virtualization control plane. The current focus is VMware-exit discovery and migration-readiness assessment with approval-ready first-wave planning. So before I show anything else, I want to show the tenant context: which tenant we’re in, how the dashboard is authenticated, what permissions we have, and what backend we’re talking to.”

#### 1:15-2:15

“Discovery is still CLI-first today, and I want to be explicit about that instead of hiding it. The dashboard is not pretending to own connector setup yet. What matters is that discovery lands in persisted snapshots and the rest of the workflow reads from that same backend truth.”

#### 2:15-5:00

“Now I’m confirming that we have both the source baseline and the target baseline available, and that the user can tell how fresh the current view is before planning from it. The key point is that this is an assessment page, not just an asset list.”

#### 5:00-6:30

“I’ll pick one workload to show the decision logic. In the detail panel we can see the workload’s current status, dependency context, and baseline activity clues. That is the bridge from ‘interesting asset’ to ‘candidate for the first wave.’”

“If I want to act on this, I can open the migration plan directly from here.”

#### 6:30-10:30

“This is the real planning workflow: bring scope in, prepare the target and execution controls, validate, then save plan state. The important thing here is that Viaduct keeps the backend as the current reference.”

“I’m showing a prepared first-wave draft with explicit target details, execution controls, approvals, and mappings. Then preflight turns that draft into an operational decision. We can see blockers, warnings, and the derived runbook before execution starts.”

“That is the real product moment: the team gets an approval-ready plan, not just a dashboard impression.”

#### 10:30-13:00

"For a serious evaluation, I also want to show how Viaduct handles supervision when a pilot is not clean. This saved run reached `verify` and failed after earlier checkpoints completed."

“What matters here is that the team can see the migration ID, current phase, checkpoints, per-workload state, and next-action options from persisted state alone. That is a much more credible early-product story than pretending everything always goes green.”

#### 13:00-15:00

“Finally, the workflow has to end in evidence. The team can export summary, migration, and audit outputs directly from the product. That supports internal review and change evidence without pulling data manually from the backend.”

“The honest boundary is this: Viaduct is strongest today in discovery, readiness reduction, first-wave planning, and supervised pilot control. That is the product we are intentionally hardening first.”

### What To Avoid

Do not:

- open by showing every connector in the repo
- lead with KVM lab unless the audience explicitly wants the evaluation path
- spend time tweaking form fields live unless it proves a point
- show lifecycle, policy, drift, backup portability, or plugins before the main path is trusted
- claim that the dashboard already manages source connections
- imply that a prepared failed run state is proof of live-production runtime certification
- get stuck in a long live discovery or export wait

### Likely Questions Or Objections

Expect these after the 15-minute demo:

- “How much of this is prepared versus live?”
- “Can you show the exact VMware and Proxmox discovery path?”
- “Why is discovery CLI-first rather than in-app?”
- “What is actually supported today for the named pilot?”
- “What happens to state if the API restarts mid-run?”
- “How do approvals and auditability work?”
- “Can operators distinguish warnings from blockers cleanly?”
- “What would you need to harden before this feels ready for my team?”

### Suggested Answers

- “The current product is strongest from discovery through approval-ready planning and supervised pilot control. We use prepared state in the demo so we can focus on operator reasoning instead of environment latency.”
- “The named v1 live motion is VMware vSphere source to Proxmox VE target. Other implemented surfaces in the repo are real, but they are not the current support promise.”
- “Discovery is CLI-first today because we have not productized an in-app connection manager yet. We prefer being explicit about that boundary.”
- “Approvals, audit, and reporting are part of the core trust story, not a later add-on, which is why I showed tenant context and exports instead of hiding them.”
- “There are still hardening gaps. For example, execute and resume durability are part of the current reliability-hardening work, which is exactly why we present this as supervised pilot control rather than mature fleet automation.”

## 6. Setup And Smoothness Checklist

Use this checklist before any serious Viaduct demo.

The copy-ready working version of this checklist is in [demo/demo-state-checklist.md](demo/demo-state-checklist.md).

### Environment

- use a persistent store
- use a named service account if possible
- use a prepared tenant with clean data
- verify the dashboard opens directly to a stable environment
- close unrelated browser tabs and terminal windows

### Screen Flow

- pre-open `Settings`, `Inventory`, `Migrations`, and `Reports`
- know the exact click path between them
- keep browser zoom readable
- avoid popups, notifications, or desktop clutter
- use the same workload names every time

### Data

- baseline timestamps look recent and believable
- ready / needs-review / blocked examples are all visible
- saved plan is present
- preflight shows both blockers and warnings
- saved failed `verify` run shows checkpoints and per-workload state
- exports work before the call starts

### Talk Track

- open with the VMware-exit problem, not the architecture
- say the support boundary early
- say the live-motion boundary early
- say discovery is CLI-first before someone asks
- keep the narrative on readiness reduction and supervised pilot control

### Credibility

- do not claim broad automation breadth
- do not oversell fixture-backed data
- do not improvise product promises
- answer “what is real today?” more clearly than “what could exist later?”
- if something is prepared state, say so

### Fallbacks

- have one screenshot or backup browser window for each main page
- have one prepared migration ID ready to paste or reference
- have one export file already downloaded in case export latency appears
- have the CLI commands visible even if you do not run them

## 7. What Should Be Avoided Across Both Demos

Avoid these patterns completely:

- feature-bingo demo flow
- connector matrix bragging
- long YAML editing
- long terminal usage
- KVM-lab-first positioning in a VMware-exit conversation
- “trust us, it also does X” language
- showing half-finished secondary features before the core path lands
- pretending prepared state equals production proof

## 8. Demo Success Criteria

The 3-minute demo succeeded if the audience can answer:

- what Viaduct is for
- who it is for
- why it is different from a generic inventory dashboard
- why a longer evaluation is worth their time

The 15-minute demo succeeded if the audience can answer:

- where the operator trust points are
- how the workflow moves from inventory to a real saved plan
- how Viaduct exposes supervision when a pilot is not clean
- what the current support boundary is
- what they would want to validate next in a real pilot

## 9. Self-Review

This runbook is intentionally narrow.

It does not:

- create a flashy broad-platform feature tour
- pretend the current KVM evaluation path is the same as a live VMware pilot
- claim that every repo capability belongs in the first product story

That is deliberate. The risk in Viaduct demos is not lack of material. The risk is overselling breadth and weakening trust.

The main limitation of this runbook is that it still depends on a prepared seeded tenant state rather than a first-class packaged VMware demo environment. That is an honest gap in the current repo and should remain visible until a stronger pilot kit exists.

Another deliberate choice is that the 3-minute demo now opens on `Inventory`, not `Settings`. That is better for clarity and initial interest, even though the longer demo still needs to prove operator trust and environment context explicitly.
