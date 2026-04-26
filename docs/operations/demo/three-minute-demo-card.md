# 3-Minute Demo Card

Use this card live. Keep it tight.

## Audience

- first meetings with infrastructure or platform leads
- short founder intros
- conference or meetup lightning demos
- top-of-funnel design-partner calls

## Goal

Create clarity and interest fast:

- what Viaduct is for
- why it is credible now
- why a longer evaluation is worth the time

## Screen Order

1. `Inventory`
2. `Workload detail`
3. `Migrations`
4. `Migrations` execute state
5. `Reports`

## Timebox

| Time | Screen | Show |
| --- | --- | --- |
| 0:00-0:20 | `Inventory` | status cards, discovery context, visible baseline timestamp |
| 0:20-0:50 | `Workload detail` | one candidate workload, readiness, risk, dependency, baseline context |
| 0:50-1:45 | `Migrations` | planning workflow, preflight summary, saved plan or runbook |
| 1:45-2:20 | `Execute` | migration ID, failed or blocked state, checkpoints or per-workload state |
| 2:20-2:45 | `Reports` | summary, migrations, and audit exports |
| 2:45-3:00 | close | support boundary and invitation to longer demo |

## What To Say

### 0:00-0:20

"Viaduct is for VMware-exit teams that need to turn a multi-platform environment into a credible first migration wave before they trust automation. I'll start with the payoff: which workloads look ready, which need review, and which are blocked."

### 0:20-0:50

"Here's one candidate workload. Viaduct pulls readiness, dependency, and baseline context together so an operator can justify inclusion in a first wave instead of guessing from disconnected tools."

### 0:50-1:45

"Once we have a candidate set, Viaduct moves into a real backend planning flow: scope, prepare, validate, and save the plan. Preflight is what turns that draft into something operationally meaningful by separating blockers from warnings before execution."

### 1:45-2:20

"And if a pilot is not clean, the operator can still reason from saved state: migration ID, checkpoints, per-workload status, and next action. The story here is supervised pilot control, not blind automation."

### 2:20-2:45

"Viaduct also closes the loop with summary, migration, and audit exports, so the team can hand an approver or reviewer real evidence instead of a screenshot deck."

### 2:45-3:00

"The honest boundary is that Viaduct is strongest today in discovery, readiness reduction, approval-ready planning, and supervised pilot control. If that is your problem, the longer demo goes much deeper."

## Do Not Do

- do not open with `Settings`
- do not spend time in the CLI
- do not tour secondary features
- do not imply broad unattended-migration maturity
- do not imply multiple supported live target motions

## Likely Questions

- Is this live VMware data or seeded demo state?
- How much of this is planning versus real execution?
- Why is discovery CLI-first?
- What target is actually supported?

## Short Answers

- "This is a workflow-trust demo. The important point is how Viaduct turns inventory into a supervised first-wave plan."
- "The named v1 live motion is VMware to Proxmox, and we frame execution as supervised pilot work."
- "Discovery is still CLI-first today. The dashboard is the review and control surface on top of persisted backend state."
