# Initial Use Case Analysis

This document locks the first real product focus for Viaduct. It is written to be used by the maintainer, product lead, demo owner, and early design-partner team when deciding what Viaduct should say, ship, and refuse to promise.

## Decision Goal

Pick the narrowest buyer problem that Viaduct can credibly win first.

The chosen focus must:
- fit the strongest current product behavior in the repo
- map to an urgent and funded operator problem
- work with the current evaluation and demo path, even if that path still needs tightening
- create a believable bridge from discovery and planning into supervised pilot execution
- avoid forcing claims of broad hands-off migration breadth that the current repo does not yet prove

## Viaduct-Specific Decision Constraints

These constraints matter more than market size slides or category language.

### What The Repo Already Proves
- Viaduct already proves multi-platform discovery, normalized inventory, dependency graphing, lifecycle signals, declarative planning, preflight checks, approval gates, checkpoints, resume behavior, rollback visibility, tenant-scoped APIs, and packaged release discipline.
- The current quickstart and release-readiness path prove an end-to-end evaluation loop through the local KVM lab, not a first-class live VMware migration pilot.
- The support matrix already states that production migration usage should be treated as lab or pilot work first, especially for connector-specific runtime actions beyond discovery.
- The strongest story today is not "one-click migration." It is "understand the environment, reduce uncertainty, define the first safe move, and keep controls explicit."

### What The Repo Does Not Yet Prove Strongly Enough
- a broad VMware-to-anything cutover motion
- equal execution maturity across all source and target combinations
- a first-class packaged VMware-exit pilot kit that is as polished as the current KVM lab evaluation path
- an MSP-first or ecosystem-first operating model

These constraints should drive the focus choice. If the focus ignores them, it becomes positioning debt.

## Candidate Focus Positions

### Option 1: VMware To Proxmox Migration Planning And Supervised Execution

**Buyer**
- infrastructure teams actively trying to reduce VMware cost by moving defined workloads into Proxmox

**What Viaduct Can Credibly Support Today**
- VMware discovery inputs
- Proxmox as a plausible and visible target in the repo
- planning, preflight, approvals, checkpoints, and supervised pilot execution mechanics

**Why This Should Not Be The First Focus**
- it forces the product story too quickly into target-specific execution confidence
- it asks buyers to trust the riskiest part of the workflow before Viaduct has a polished, repo-backed evaluation path for this exact motion
- it narrows the story to one destination even though many real buyers are still comparing targets

**Verdict**
- strong future packaged offer
- not the safest opening focus

### Option 2: VMware-Exit Multi-Platform Inventory Collection And Migration Readiness Assessment With Approval-Ready Pilot Planning

**Buyer**
- platform and virtualization teams under VMware renewal pressure who need to understand what they have, what is risky, and what the first supervised migration wave should look like

**What Viaduct Can Credibly Support Today**
- multi-platform inventory collection and normalized inventory
- dependency, backup, policy, cost, and drift context
- declarative planning, preflight validation, and explicit approval and checkpoint state
- shared CLI, API, and dashboard workflows that make assessment and pilot planning visible

**Why This Is Strongest**
- it matches the current proof surface in the repo
- it speaks to a painful, funded problem
- it does not require overclaiming autonomous production execution
- it gives Viaduct a clear path from evaluation into supervised pilot use

**Verdict**
- best initial focus

### Option 3: Dependency-Aware Migration Orchestration For Platform Teams

**Buyer**
- platform teams that care about workload interdependencies and blast radius during migration

**What Viaduct Can Credibly Support Today**
- dependency graphing
- wave planning structure
- planning and execution control concepts

**Why This Should Not Be The First Focus**
- it is a feature-centered message, not a clean buyer problem
- it sounds differentiated to builders but still reads abstractly to operators and buyers
- it pulls the story toward orchestration sophistication before the simpler trust story is fully earned

**Verdict**
- important differentiator inside the chosen focus
- weak opening focus by itself

### Option 4: Lifecycle And Risk Assessment For Inherited Virtualization Environments

**Buyer**
- teams inheriting a messy virtualization environment through acquisition, centralization, or reorganization

**What Viaduct Can Credibly Support Today**
- discovery, cost, policy, drift, and reporting views
- tenant-scoped operational visibility

**Why This Should Not Be The First Focus**
- it is credible, but urgency is usually weaker than VMware-exit pressure
- it risks positioning Viaduct as an assessment-only tool
- it underuses Viaduct's planning and supervised migration strengths

**Verdict**
- credible secondary entry point
- weaker than the VMware-exit focus for first adoption

### Option 5: Multi-Tenant Migration Control Plane For MSPs And Migration Partners

**Buyer**
- MSPs or migration specialists managing multiple customer environments

**What Viaduct Can Credibly Support Today**
- tenant-scoped state, service accounts, packaging, and API surfaces

**Why This Should Not Be The First Focus**
- it adds support, trust, audit, and workflow demands before the core direct-use path is proven
- it depends on stronger operator maturity than the current repo should promise first
- it creates go-to-market and product complexity too early

**Verdict**
- expansion path
- not an opening focus

## Evaluation Criteria

The initial focus should be judged against these exact questions:

1. Does it map to a buyer who already has a funded problem right now?
2. Can Viaduct demonstrate the core workflow from the current repo, docs, dashboard, API, and release artifacts without hand-waving?
3. Does it let the product earn trust before asking for production cutover authority?
4. Does it produce a concrete operator outcome, not just a dashboard insight?
5. Does it expand naturally into deeper execution, lifecycle, and repeatable operations later?

## Decision Matrix

| Candidate | Urgency | Current product fit | Trust requirement | Concrete first outcome | Expansion path | Decision |
| --- | --- | --- | --- | --- | --- | --- |
| VMware to Proxmox planning and supervised execution | High | Medium | High | Pilot execution on a named target | Strong | Not first |
| VMware-exit multi-platform assessment with approval-ready pilot planning | High | High | Medium | Approved first-wave plan plus supervised pilot path | Strong | Choose now |
| Dependency-aware migration orchestration | Medium | Medium | High | Better orchestration story | Strong | Keep as differentiator |
| Lifecycle and risk assessment for inherited environments | Medium | High | Medium | Environment health and rationalization view | Medium | Secondary entry point |
| Multi-tenant control plane for MSPs | Medium | Medium | High | Partner operating surface | Strong | Much later |

## Recommendation

Lock Viaduct's initial focus to:

**VMware-exit multi-platform inventory collection and migration readiness assessment with approval-ready pilot planning**

That is the sharpest initial focus because it is the narrowest statement that still matches what the repo can defend today.

## Why This Recommendation Is Correct

### It Aligns With Viaduct's Actual Proof Path
- the repo already supports discovery, visibility, planning, preflight, and supervised pilot controls better than it supports a broad unattended-automation claim
- the current lab and quickstart prove product mechanics end to end, even though they are KVM-based rather than VMware-live
- the support matrix already asks operators to treat migration execution as pilot-scoped work first

### It Produces A Concrete Buyer Outcome
The buyer does not just get "visibility." They get:
- a normalized view of the current environment
- a clearer picture of dependency and policy risk
- a first migration wave definition
- an approval-ready pilot plan they can review internally before handing over production authority

That is a real deliverable with decision value.

### It Keeps Viaduct Honest
- it avoids pretending that every connector pair is equally execution-ready
- it avoids reducing the story to a generic lifecycle dashboard
- it avoids locking the product too early to Proxmox-only messaging when many teams are still deciding the target

### It Leaves Room For The Right Expansions
If Viaduct wins the readiness and pilot-planning step, it can later grow into:
- stronger VMware-to-Proxmox and VMware-to-KVM pilot kits
- deeper execution certification and runbooks
- repeated wave execution with tighter target-specific automation
- post-cutover lifecycle and partner workflows

## Final Positioning Statement

**Viaduct helps VMware-exit teams turn a multi-platform virtualization environment into a dependency-aware, approval-ready first migration wave before they trust full execution automation.**

## Who It Is For

- platform, infrastructure, and virtualization leads facing VMware renewal pressure
- teams with a real existing environment, not a greenfield buildout
- operators who need one place to inspect inventory, dependencies, backup exposure, policy risk, and pilot planning inputs
- teams evaluating likely targets such as Proxmox or KVM but not yet ready to standardize the entire environment around one destination
- design partners willing to run a supervised first wave instead of demanding a zero-touch migration factory

## Who It Is Not For

- buyers who only want a passive inventory or CMDB-style dashboard
- teams demanding fully autonomous production migration across every source and target pair on day one
- organizations with no migration or environment-rationalization pressure
- MSPs expecting a fully finished multi-customer operating platform as the opening product

## What Viaduct Does Now Versus Later

### What Viaduct Does Now
- discovers and normalizes inventory across the supported connectors
- surfaces dependency, backup, cost, policy, and drift context that influences migration readiness
- lets operators define workloads, waves, approvals, windows, and preflight conditions in declarative plans
- provides tenant-scoped visibility through the CLI, API, dashboard, reports, metrics, and audit-oriented routes
- supports supervised pilot execution mechanics with explicit checkpoints, resume behavior, and rollback visibility
- proves the product workflow end to end through the current KVM-based evaluation path, while live migration use remains pilot-scoped

### What Viaduct Does Later
- adds stronger target-specific pilot kits for VMware-exit programs, especially around likely destinations such as Proxmox and KVM
- deepens connector-pair execution confidence with clearer certification and runbooks
- supports repeated wave execution with more operational automation after the first pilot path is trusted
- expands post-cutover lifecycle enforcement, remediation, and partner-oriented operating workflows

## Non-Goals For The Current Stage

Do not lead Viaduct with:
- "platform for all virtualization operations"
- "one-click cross-hypervisor migration"
- "MSP control plane" as the first product story
- lifecycle optimization as a standalone opening message

Those may become true or useful later, but they are not the initial focus.

## What This Means For The Real Repo

If this focus is real, the next product work should make these things truer:

1. Viaduct should have a clearer first-wave planning path from discovery to preflight to approval-ready pilot output.
2. The repo should gain a more explicit VMware-exit evaluation story instead of relying mainly on a KVM lab for the headline narrative.
3. Docs, demos, and dashboard copy should emphasize readiness reduction and supervised pilot planning before execution breadth.
4. Contract and observability work should focus first on the assessment and supervised pilot workflow, not on adding more product surface.
