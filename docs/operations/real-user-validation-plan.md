# Real User Validation Plan

This document defines the first practical user-validation plan for Viaduct after Phase 5.

It is not a generic discovery-interview guide. It is a field-learning kit for putting the current Viaduct product in front of five real infrastructure-minded users and learning where the core focus workflow breaks down.

The tested path is the same path Viaduct is already narrowing around:

**VMware-exit multi-platform inventory collection and migration readiness assessment with approval-ready pilot planning, plus interpretation of a prepared VMware vSphere to Proxmox VE pilot run state**

Use this document to:
- recruit the right five participants
- run comparable sessions without pitching or rescuing
- capture evidence about hesitation, misunderstanding, distrust, and blockers
- turn that evidence into product decisions for the real repo and v1 scope

Companion working files live in [validation/README.md](validation/README.md).

## 1. Current-State Summary

### What Exists Today

Viaduct already has enough real product surface to run workflow-based sessions:

- CLI discovery and planning flows documented in `docs/operations/migration-operations.md`
- dashboard runtime and trust context in `web/src/features/settings/SettingsPage.tsx`
- inventory review and workload inspection in `web/src/features/inventory/InventoryPage.tsx` and `web/src/features/inventory/WorkloadDetailPanel.tsx`
- migration planning and preflight in `web/src/features/migrations/MigrationWorkflow.tsx`
- monitoring, history, and report export in `web/src/features/reports/ReportsPage.tsx`
- focus, v1 boundary, and primary path already frozen in:
  - `docs/initial-use-case-analysis.md`
  - `docs/v1-scope.md`
  - `docs/operations/primary-reliability-path.md`
  - `docs/operations/auth-role-audit-model.md`
  - `docs/operations/observability-requirements.md`

### What Prior Phases Likely Improved

Prior phases likely made this step possible by delivering:

- a shared backend instead of frontend-only demo logic
- tenant-scoped auth, audit, reporting, and request IDs
- persisted migration state with approvals, checkpoints, resume, and rollback
- a product-grade dashboard shell with real operator pages
- enough docs and packaging discipline to run sessions honestly

### What Is Still Weak, Ambiguous, Or Risky

Viaduct still has field-validation risk:

- discovery is still CLI-first, so the end-to-end path is hybrid rather than fully in-product
- the strongest rehearsal path is still the KVM lab, while the named live motion is VMware to Proxmox
- inventory trust, provenance, and execution-state trust still need real-user proof rather than maintainer inference
- current docs and product framing are sharper than the product's real-world operator evidence
- without deliberate testing, Viaduct risks optimizing for demo smoothness instead of operator confidence

### What Should Be Preserved

This validation plan must preserve:

- the current focus instead of broadening into generic migration tooling
- the current v1 support promise instead of testing unsupported paths
- the primary reliability path instead of ten disconnected features
- the existing CLI, API, and dashboard division of responsibility
- honesty about what is productized versus doc-driven or pilot-scoped

### Smallest Credible Next Move

The smallest credible next move is to run five structured sessions against one repeatable Viaduct workflow and use the findings to reprioritize product work.

Do not broaden the test into:
- lifecycle as a separate product
- MSP workflows
- multiple target motions
- generic “what do you think of the dashboard” feedback

## 2. Validation Objective

The objective is not to collect compliments or prove that Viaduct is “cool.”

The objective is to answer these questions with observed behavior:

1. Can a real infrastructure operator understand the assessment and supervised pilot workflow without heavy explanation?
2. Where do they hesitate because the product feels unsafe, unclear, or incomplete?
3. What evidence do they need before trusting Viaduct to support a first-wave pilot decision?
4. Which parts of the current workflow are strong enough to preserve?
5. Which gaps are serious enough to block the current focus and v1 path?

### Validation Decision Rule

This round is successful if it produces evidence strong enough to do all three:

- cut or defer weak surface area
- prioritize the next 3 to 5 product changes on the main path
- update docs, demos, and positioning based on real operator behavior rather than internal guesses

### Non-Goals For This Round

This round is **not** for:

- validating pricing or packaging willingness
- collecting a broad roadmap wish list
- proving enterprise-scale rollout readiness
- certifying a live migration motion in front of the participant
- comparing Viaduct against a full competitive matrix
- testing unsupported source or target motions

This is a workflow-trust round. Keep it that narrow.

### What This Round Actually Validates

This round validates whether operators understand and trust the **migration assessment and supervised pilot control path**.

It does **not** validate that Viaduct has earned real-world runtime proof for live workload movement. That remains a separate pilot-validation step after workflow trust is stronger.

## 3. Participant Profile Criteria

Recruit **five** real participants. Use the same session structure for all five.

### Target Participant Mix

Aim for this mix:

- 3 virtualization operators or senior systems administrators with recent VMware responsibility
- 1 infrastructure or platform lead who reviews migration risk and approves first-wave decisions
- 1 technical reviewer or architect who can evaluate target-fit and operational risk for a VMware-exit first wave

Prefer direct in-house operators over consultants. Use a consultant only if they have recent hands-on migration-planning responsibility and you cannot fill the round with real operators.

### Good Participant Criteria

A participant counts as good for this round if they meet most of these:

- has worked with VMware vSphere in the last 18 months
- has participated in migration planning, infrastructure refresh, or VMware-exit evaluation
- regularly uses operational tooling, not just slideware or procurement checklists
- can talk concretely about approvals, rollback concerns, and change control
- is comfortable critiquing unclear tooling and does not need the maintainer to protect their feelings

### Poor Participant Criteria

Do **not** use these as primary participants:

- general software developers with no virtualization responsibility
- buyers who only evaluate through pricing decks
- security reviewers without operational ownership of migration planning
- friends who are likely to be supportive but not candid
- contributors who already know the Viaduct repo well
- people whose main reaction will be feature ideation instead of workflow behavior

### Recruiting Sources

Use recruiting channels that actually reach the initial focus:

- existing VMware-exit contacts
- platform and virtualization peers in your network
- design-partner prospects
- consultants only when they have recent hands-on migration-planning experience

Do not fill the round with general startup-friendly interview volunteers just to hit the number five.

### Fast Screener

Use these questions before scheduling:

1. “What is your current infrastructure or virtualization role?”
2. “Have you worked directly with VMware in the last 18 months?”
3. “Have you helped plan, approve, or execute a migration wave in the last 12 months?”
4. “Would you describe your work as hands-on operator work, technical review, or procurement/business review?”
5. “Are you comfortable walking through a CLI-plus-dashboard workflow?”

Only schedule participants who clearly fit the operator, approver, or technical-reviewer profile for the current focus.

## 4. Session Logistics

### Format

- 5 sessions
- 60 to 75 minutes each
- remote screen share or in-person desk session
- 1 moderator
- 1 silent note taker if possible

### Session Assets

Prepare one consistent test environment. Do **not** improvise the scenario between participants.

- one tenant-scoped Viaduct environment with working dashboard access
- saved VMware and Proxmox discovery snapshots already available to the tenant
- one prepared CLI terminal with a valid config and a copyable discovery command
- the same three workload archetypes for every session
- one migration saved in `plan` state
- one migration saved in a `failed` or `verify`-stage discussion state for monitoring/resume discussion
- report export endpoints working for summary, migrations, and audit

### Environment Fidelity Rule

Prefer this environment order:

1. sanitized VMware and Proxmox snapshots from a real or pilot-like environment
2. internally prepared tenant data that mirrors a plausible VMware-to-Proxmox first-wave scenario
3. fixture-backed or synthetic data only if it is clearly disclosed as evaluation data

Do not imply fixture-backed data is field proof. If the session uses synthetic or fixture-backed data, say so in the intro and treat the session as workflow-clarity and trust-shaping research, not runtime-certification evidence.

### Pre-Session Checklist

Before each session, verify:

- `Settings` loads and shows the intended tenant and auth mode
- the same saved VMware and Proxmox baseline is visible
- the three standardized workload candidates are present in `Inventory`
- the saved first-wave plan is available in `Migrations`
- the saved failed or late-stage pilot run is available in `Migrations`
- summary, migration, and audit exports are reachable from `Reports`
- the prepared CLI command is visible and copyable
- you can complete the whole scenario yourself in under 10 minutes without improvising

Use the copy-ready checklist in [validation/pre-session-checklist.md](validation/pre-session-checklist.md) during actual session prep.

### Standardized Session Scenario

Use the same operator-shaped scenario in every session:

- one **low-risk candidate** workload
  visible in the current UI as `ready` / low risk, with straightforward host, storage, and network context
- one **medium-risk candidate** workload
  visible in the current UI as `needs-review` / medium risk, with dependency or mapping complexity that still looks plausible with operator review
- one **high-risk defer** workload
  visible in the current UI as blocked or high risk, with reasons the current product actually shows today, such as snapshots, backup relationships, policy violations, or partial assessment signals

Use a prepared migration example with:

- a saved plan that includes the low-risk and medium-risk candidates
- preflight results that include both warnings and at least one blocking issue the participant must interpret
- a saved run state with completed checkpoints followed by a later-phase issue, so the participant must reason about “what failed, where, why, and what next”

The scenario must be stable enough that differences in participant behavior are about product understanding and trust, not random environment differences.

### Safety Rule

Do **not** run destructive live migration actions during these sessions.

For this round:

- discovery can be run or discussed
- planning and preflight can be real
- execution should be observed through prepared persisted state, not live workload movement
- resume and rollback should be discussed from saved run history unless you have an isolated lab specifically prepared for that purpose

### Why This Setup Is Right For Viaduct

This keeps the sessions honest without wasting time on environment debugging:

- discovery can still be tested as a real CLI step because that is the current product reality
- dashboard review, planning, validation, monitoring, and reporting can all be tested in the product
- the participant is evaluating the real supported path, not a mocked future workflow

## 5. Top 5 Workflows To Test

These are the only workflows to test in this round.

### Workflow 1: Confirm Workspace And Trust Context

**Purpose**
- test whether the participant can orient themselves before making changes

**Primary Surfaces**
- `Settings`
- `/api/v1/about`
- `/api/v1/tenants/current`

**Prompt**
- “Before doing anything, show me how you would confirm that you are in the right environment and have the right level of access.”

**What Good Looks Like**
- participant notices tenant, auth method, role, permissions, and build/runtime context
- participant can say whether they trust the current workspace enough to proceed

### Workflow 2: Refresh Or Verify Discovery Baseline

**Purpose**
- test whether the CLI-first discovery reality is understandable and trustworthy

**Primary Surfaces**
- `viaduct discover`
- `docs/operations/migration-operations.md`
- `Inventory`
- `Reports`

**Prompt**
- “You have been asked to verify that the current VMware and Proxmox baseline is recent enough and credible enough for first-wave planning. Show me what you would check first. If you think discovery should be rerun, explain how you would do it and what output you would need before continuing.”

**What Good Looks Like**
- participant understands discovery is CLI-driven today
- participant does not confuse a missing dashboard control with a missing product capability
- participant looks for freshness, scope, source provenance, and missing context
- participant can tell whether the current baseline is trustworthy enough for planning

**Moderator Note**
- prefer explanation plus baseline verification over spending the session on shell syntax
- only have the participant run the command if the environment is ready and it will not consume the task time

### Workflow 3: Inspect One Candidate Workload

**Purpose**
- test whether the participant can decide include, exclude, or hold based on the current inventory and workload detail surfaces

**Primary Surfaces**
- `Inventory`
- workload detail panel
- dependency graph

**Prompt**
- “Pick one workload you think is a possible first-wave candidate. Walk me through whether you would include it, exclude it, or investigate further.”

**What Good Looks Like**
- participant uses inventory, workload detail, and dependency/risk context together
- participant asks the right trust questions instead of guessing
- participant can explain the decision in operational language

### Workflow 4: Create And Validate A First-Wave Plan

**Purpose**
- test whether the participant can move from assessment into an approval-ready plan

**Primary Surfaces**
- `Migrations`
- migration planning workflow
- preflight results

**Prompt**
- “Create or adjust a first-wave plan for this workload set and tell me whether you would put it in front of an approver today.”

**What Good Looks Like**
- participant understands the planning model well enough to make changes
- participant can interpret blocking failures versus warnings
- participant knows what additional evidence is still needed before execution

### Workflow 5: Monitor A Pilot And Export Evidence

**Purpose**
- test whether the participant can understand current execution state and produce stakeholder-ready evidence

**Primary Surfaces**
- `Migrations` saved run detail and progress state
- `Reports`
- audit export

**Prompt**
- “A pilot run has already failed after earlier checkpoints completed. Show me what happened, whether you would resume, hold, or roll back, and what evidence you would export for a reviewer or maintainer.”

**What Good Looks Like**
- participant can explain execution state, approval state, and likely next action
- participant can find reporting and audit outputs without coaching
- participant can say what would make them trust or distrust this run

**Moderator Note**
- use the same saved migration state in every session
- prefer a failure in a later phase such as `verify` so the participant must reason about checkpoint history rather than just noticing an obvious early rejection
- test failure reasoning in `Migrations`, then test handoff and evidence in `Reports`

## 6. Moderator Script

Use this script as written unless the participant is genuinely blocked by environment issues.

### Opening Script

“Thanks for doing this. I’m testing how understandable and trustworthy Viaduct feels for real migration-readiness work. I’m not testing you. I want to see where the product is confusing, risky, or incomplete. Please think out loud as you work. I may stay quiet for stretches because I want to observe how you naturally interpret the workflow.”

### Reality Framing

“Viaduct is currently optimized around VMware-exit assessment, first-wave planning, and supervised pilot control. Some parts are productized, and some parts are still more operator-driven or docs-driven. If something feels incomplete or misleading, that is valuable feedback.”

### Before Starting Tasks

“Use this as if you were evaluating whether this tool is credible enough for a real first-wave pilot. If you would normally stop, verify, or ask for more proof, do that here too.”

### Transition Script Between Tasks

After each task:

“What, if anything, made you hesitate there?”

If they moved quickly:

“What evidence made that feel safe enough to continue?”

If they struggled:

“What were you looking for that you did not get?”

### Closing Script

“Looking back across the whole workflow, where would you trust this today, where would you stop, and what would need to change before you would use it in a real pilot?”

## 7. Session Structure

Use this structure for all five sessions.

| Time | Segment | Purpose |
| --- | --- | --- |
| 0-5 min | Intro and consent | Set expectations and ask for think-aloud behavior |
| 5-10 min | Background questions | Confirm participant fit and current workflow |
| 10-18 min | Workflow 1 | Trust and orientation |
| 18-30 min | Workflow 2 | Discovery and baseline trust |
| 30-42 min | Workflow 3 | Workload selection and risk understanding |
| 42-57 min | Workflow 4 | Planning and preflight interpretation |
| 57-68 min | Workflow 5 | Monitoring, evidence, and handoff |
| 68-75 min | Debrief | Capture trust thresholds and blockers |

If time is short, do not add extra feature tours. Cut breadth, not observation quality.

### Moderator Rescue Rule

Use at most one neutral rescue per workflow.

A neutral rescue is a prompt like:

- “Talk me through what you are looking for.”
- “Where would you expect that information to live?”

Do not convert a stall into a guided walkthrough. If the participant cannot continue without direct steering, mark the workflow as `blocked`.

### Session Invalidation Rules

Invalidate and reschedule a session if:

- the participant clearly does not fit the target profile
- environment setup consumes more than 15 minutes
- the participant never reaches the core workflows because of credential or lab issues
- the session turns into a sales conversation instead of product use

Do not mix invalid sessions into the synthesis just to keep the sample size at five.

## 8. Questions To Ask

### Before The Tasks

Ask these before touching the product:

1. “What is your current role in infrastructure or virtualization operations?”
2. “Have you been involved in VMware-exit planning, platform refresh, or migration approval in the last year?”
3. “What tools or evidence do you normally rely on before you trust a first migration wave?”
4. “How comfortable are you with CLI-driven operational steps versus UI-driven ones?”
5. “What would make you immediately distrust a tool in this category?”

### During The Tasks

Use only when useful. Do not ask every question every time.

- “What are you expecting to see here?”
- “What makes this feel current or stale?”
- “If this were wrong, what would the blast radius be?”
- “Would you proceed from this state, or would you stop?”
- “What would you need to explain this to another operator or approver?”
- “What part of this feels assumed rather than proven?”
- “What next action do you think the product expects from you?”

### After The Tasks

Ask these in debrief:

1. “Where in the workflow did you feel most confident?”
2. “Where did you feel the most risk or uncertainty?”
3. “What looked useful but not trustworthy enough yet?”
4. “Which step felt like the biggest gap between product promise and product proof?”
5. “If this were being evaluated seriously by your team, what would have to improve first?”

## 9. What To Observe Silently

Capture these without interrupting:

- first place they click or navigate when asked to orient themselves
- whether they trust `Settings` as a real control-context surface
- whether they look for discovery freshness, source provenance, and missing data signals
- whether they understand the CLI/dashboard split or assume missing UI capabilities
- where they reread labels, hesitate, or backtrack
- whether they seek evidence of approvals, checkpoints, rollback, or auditability before execution trust
- whether they notice exports naturally or only after prompting
- whether they ask for per-workload activity history, actor attribution, or failure provenance
- whether they use current operational language or “demo browsing” language

## 10. What Not To Do During Sessions

Do not:

- pitch the product while the participant is trying to work
- explain where controls are before they look
- defend unclear product behavior in real time
- promise future features to paper over current confusion
- turn the session into a brainstorming workshop
- ask “do you like it?” instead of observing behavior
- reward speed over caution in a trust-sensitive workflow
- silently translate current product gaps into future-state assumptions
- imply Viaduct already has a fully productized in-app source-connection flow

## 11. Note-Taking Template

Use one sheet per participant.

The copy-ready working version of this template is in [validation/participant-note-template.md](validation/participant-note-template.md).

```md
# Participant
- ID:
- Date:
- Role:
- Company type:
- VMware responsibility:
- Migration-planning responsibility:
- CLI comfort:
- Session moderator:
- Session note taker:

# Baseline Context
- Current tools used:
- Current approval/change process:
- Current migration target(s) under evaluation:
- Stated trust threshold:

# Workflow Notes
| Workflow | Goal | Observed behavior | Hesitation / misunderstanding | Trust or distrust signal | Severity | Evidence / quote |
| --- | --- | --- | --- | --- | --- | --- |
| 1. Workspace and trust context | Confirm tenant, auth, and environment |  |  |  |  |  |
| 2. Discovery baseline | Refresh or verify source/target discovery |  |  |  |  |  |
| 3. Workload inspection | Decide include / exclude / hold |  |  |  |  |  |
| 4. Plan and validate | Build and assess first wave |  |  |  |  |  |
| 5. Monitor and report | Interpret state and export evidence |  |  |  |  |  |

# Cross-Cutting Signals
- Asked for missing evidence:
- Expected a different workflow:
- Most trusted moment:
- Least trusted moment:
- Biggest blocker:
- Biggest doc or onboarding gap:
- Biggest observability gap:
- Biggest audit or accountability concern:

# Immediate Recommendations
- P0:
- P1:
- P2:
- Keep as-is:

# Session Score
- Workflow 1 status: clear / hesitant / blocked
- Workflow 2 status: clear / hesitant / blocked
- Workflow 3 status: clear / hesitant / blocked
- Workflow 4 status: clear / hesitant / blocked
- Workflow 5 status: clear / hesitant / blocked
- Overall trust rating: low / medium / high
- Would this participant put Viaduct into a serious pilot evaluation today: yes / maybe / no
```

## 12. Synthesis Template

Use this after all five sessions.

The copy-ready working version of this template is in [validation/round-synthesis-template.md](validation/round-synthesis-template.md).

```md
# Validation Round Summary
- Date range:
- Number of participants:
- Participant mix:
- Sessions completed:
- Sessions invalidated and why:

# Decision Summary
- Is the current focus still correct?
- Is the current primary reliability path still correct?
- What part of the workflow earned the most trust?
- What part of the workflow most damaged trust?
- What should be cut, clarified, or hardened next?

# Top Findings
| Finding | Affected participants | Workflow step | Type | Evidence | Severity | Confidence | Recommended change |
| --- | --- | --- | --- | --- | --- | --- | --- |
|  |  |  | hesitation / misunderstanding / distrust / blocker |  | P0 / P1 / P2 / P3 | high / medium / low |  |

# Pattern Breakdown
- Repeated misunderstanding:
- Repeated trust gap:
- Repeated missing evidence:
- Repeated UI wording issue:
- Repeated backend or state-model issue:
- Repeated doc or onboarding issue:

# Things To Preserve
- Existing behavior or surface that users consistently trusted:
- Existing behavior or surface that improved comprehension:

# Proposed Product Actions
1. 
2. 
3. 
4. 
5. 
```

## 13. How To Convert Findings Into Prioritized Product Changes

Use this rule set. Do not turn findings into an unranked ideas list.

### Priority Rules

**P0**
- blocks the core focus workflow outright
- causes unsafe interpretation of readiness or execution state
- makes the product feel untrustworthy on the main v1 path
- observed in 2 or more participants, or severe enough in 1 participant that a real pilot would stop

**P1**
- does not block the path completely but causes repeated hesitation, misinterpretation, or workarounds
- weakens approval-ready planning or pilot control
- clearly affects the chosen focus, not a side feature

**P2**
- creates friction but does not materially change trust in the main path
- worth fixing after the main blockers

**P3**
- interesting, adjacent, or future-facing
- not part of the current focus, v1 promise, or primary reliability path

### Decision Thresholds

Use these thresholds after the five sessions:

- **3 or more participants blocked on the same workflow step**: treat as `P0` or cut the claim tied to that step
- **2 or more participants hesitate for the same trust reason on the same workflow step**: treat as at least `P1`
- **1 participant raises a severe unsafe-interpretation risk**: treat as `P0` until disproven
- **requests that appear only as opinions, without observed workflow impact**: default to `P2` or `P3`
- **adjacent feature asks outside the initial focus**: record, then defer unless they explain a real blocker on the main path

### Evidence Weighting Rules

When findings conflict, rank them in this order:

1. observed behavior over stated opinion
2. direct operator pain over consultant feature suggestions
3. repeated breakdowns on the primary path over one-off comments on side features
4. trust and safety issues over speed and convenience requests
5. workflow blockers over cosmetic polish

### Required Tagging For Every Resulting Backlog Item

Every backlog item created from this validation round should include:

- workflow step affected
- observed behavior
- why it matters to the initial focus
- whether it is a trust, clarity, reliability, or scope problem
- repo area likely affected
- proposed acceptance check

### Viaduct-Specific Backlog Buckets

Sort findings into one of these buckets first:

1. **Trust controls**
   auth context, role clarity, approvals, auditability, action attribution
2. **Readiness clarity**
   discovery freshness, provenance, workload readiness, planning confidence
3. **Execution reliability**
   preflight interpretation, execution state, resume, rollback, failure understanding
4. **Observability and supportability**
   request IDs, failure summaries, diagnostics, operator-visible next steps
5. **Docs and onboarding**
   CLI/dashboard split, product promises, operator expectations, session setup friction
6. **Out-of-scope pressure**
   feature asks that are real but should be deferred under the current focus

### Repo Routing Map

Use this map when converting findings into work:

| Workflow area | Likely repo surfaces |
| --- | --- |
| Workspace and trust context | `web/src/features/settings/`, `internal/api/tenant_admin.go`, `internal/api/middleware.go` |
| Discovery baseline trust | `cmd/viaduct/`, `internal/discovery/`, `internal/store/`, `docs/operations/migration-operations.md`, `web/src/features/inventory/` |
| Workload inspection | `web/src/features/inventory/`, `internal/api/server.go`, `internal/deps/`, `internal/models/` |
| Planning and preflight | `web/src/features/migrations/`, `internal/migrate/`, `internal/api/server.go`, `docs/v1-scope.md` |
| Monitoring and evidence export | `web/src/features/reports/`, `web/src/components/MigrationHistory.tsx`, `internal/api/reports.go`, `internal/api/observability.go`, `docs/operations/observability-requirements.md` |

### Required Post-Session Output

Within 48 hours of the fifth session, produce:

- top 5 findings
- top 3 preserve decisions
- top 3 cuts or deferrals
- next 3 engineering tasks
- next 2 docs/demo changes

If the team cannot do that, the synthesis is not sharp enough.

## 14. Recommended Execution Cadence

Run this validation round in one short burst:

1. recruit all five participants within one week
2. run the sessions inside a 10-day window
3. synthesize findings within two days of the last session
4. convert findings into backlog work in the same week
5. do not start a second round until the top P0 and P1 findings are addressed or explicitly deferred

This keeps the learning tied to actual product decisions instead of creating research debt.

## 15. Viaduct-Specific Session Guardrails

These guardrails matter for this product:

- test the VMware-exit focus, not generic platform management
- if live migration is discussed, keep the candidate target motion limited to VMware source and Proxmox target and state that current release validation is still governed by the support matrix
- do not present KVM lab rehearsal as equivalent to live pilot proof
- do not imply that discovery is fully in-product today
- do not run real destructive migration actions as part of a validation session unless the session is explicitly a lab-only engineering rehearsal
- do not let lifecycle, policy, or backup portability become the headline if the participant has not yet trusted the core assessment and supervised pilot workflow
- if a participant strongly wants a workload activity feed, richer execution provenance, or clearer freshness evidence, treat that as a serious signal, not cosmetic feedback

## 16. Definition Of A Good Validation Round

This round is good if, after five sessions, Viaduct can answer these with evidence:

- which exact step most damages operator trust today
- which exact step already feels strong enough to preserve
- what proof operators need before they will trust a first-wave plan
- whether the current CLI-plus-dashboard split is acceptable for the initial focus
- what the next three product changes should be on the primary path

If the output is just:
- “users liked it”
- “they wanted more integrations”
- “they thought the UI was clean”

then the round failed.
