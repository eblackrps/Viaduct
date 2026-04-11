# Commercialization Options

This document defines the realistic commercialization paths for Viaduct at its current stage and the product consequences of each.

It is not a generic startup strategy memo. It is a steering document for the current repo, current support boundary, and current early-product wedge:

**VMware-exit mixed-estate discovery and migration readiness assessment with approval-ready pilot planning**

## 1. Current-State Summary

Viaduct already has several important product and commercialization foundations:

- an Apache 2.0 open source repo with public docs, packaging, and release discipline
- a credible early-product wedge in [early-product-focus.md](early-product-focus.md), [beachhead-use-case.md](beachhead-use-case.md), and [v1-scope.md](v1-scope.md)
- a packaged evaluation path, release bundles, install docs, upgrade docs, rollback docs, and support docs
- tenant-scoped auth, service accounts, auditability, observability requirements, and a primary reliability path
- a serious demo and validation kit for design-partner conversations

That is enough to discuss commercialization seriously. It is not enough to pretend Viaduct is already a polished enterprise platform.

### What prior phases likely improved

Prior phases appear to have created the minimum product surface needed for commercial learning:

- a shared backend across CLI, API, and dashboard
- persistent state and tenant-scoped workflows
- release-gated packaging and deployable artifacts
- public operator documentation
- a narrowed beachhead and v1 boundary instead of a broad repo-first story

### What is still weak or risky

The current repo still has commercialization constraints that matter:

- the strongest repeatable evaluation path is still the KVM lab, while the named live motion is VMware to Proxmox
- live pilot hardening is still in progress, especially around durable execution, supportability, and real-user trust
- auth and RBAC are pilot-grade, not enterprise-identity-grade
- support is explicitly best effort community support today
- there is no productized billing, entitlement, licensing, or telemetry posture beyond the open-source repo
- the current value is strongest in reducing migration uncertainty and supporting supervised pilots, not in broad autonomous operations

### What should be preserved

Whatever commercialization path Viaduct chooses next should preserve:

- the current Apache 2.0 core and open-source credibility
- the VMware-exit assessment-to-pilot wedge
- the one-path-first hardening discipline
- the existing packaging and release-gate discipline
- honest support and maturity boundaries

## 2. Commercialization Framing

The right commercialization model for Viaduct now is the one that:

1. monetizes the product where it is already strongest
2. does not force enterprise-platform obligations before the product can support them
3. keeps the public story aligned with the current wedge and v1 scope
4. preserves trust with operators who expect serious infrastructure software to be transparent about support, telemetry, and licensing
5. leaves room for a stronger recurring product business later if field evidence supports it

That rules out any path that depends immediately on:

- broad support promises the maintainer cannot yet honor
- aggressive license gating around the current Apache 2.0 core
- mandatory phone-home telemetry
- roadmap sprawl toward MSP breadth, hosted control-plane complexity, or enterprise IAM before the primary pilot path is boringly reliable

## 3. Commercialization Paths

### Comparison Snapshot

| Path | Fit With Current Repo | Revenue Speed | Product Leverage | Operational Burden | Overall View |
| --- | --- | --- | --- | --- | --- |
| Open source plus paid design-partner services and pilot delivery | Strong | Fast | Medium | Medium | Best near-term path |
| Open-core control plane | Medium | Medium | High if successful | High | Too early now |
| Commercial packaged platform or appliance | Medium-low | Slow | High | Very high | Plausible later, not now |
| Portfolio product for consulting lead generation | Medium | Fast | Low | Low-medium | Viable fallback, weak product outcome |
| Internal-only or showcase product | Low | None | Low | Low | Not aligned with current repo direction |

## Path 1: Open Source Plus Paid Design-Partner Services And Pilot Delivery

### What it is

Keep Viaduct as an Apache 2.0 open-source product and monetize through:

- fixed-scope assessment engagements
- paid design-partner pilots
- migration-readiness workshops and implementation help
- optional support retainers for teams using the supported pilot path

### Why it fits Viaduct now

- it matches the current maturity of the repo
- it monetizes the exact place where buyers still need human trust and operator guidance
- it does not require inventing a premium edition before the product boundary is clear
- it keeps the public repo, docs, demos, and design-partner feedback loop aligned

### Pros

- fastest honest path to revenue
- consistent with the current Apache 2.0 and community-support posture
- lets field learning directly harden the product
- works well with the current beachhead and pilot framing
- avoids premature license and entitlement complexity

### Cons

- scales more slowly than a pure software subscription
- risks roadmap drift into bespoke consulting work
- founder or maintainer time can become the bottleneck
- revenue quality depends on disciplined scoping and repeatable delivery
- commercial outcomes can collapse into generic consulting if the offer shape is not kept narrow

### Product implications

| Area | Consequence |
| --- | --- |
| Roadmap | Prioritize one boringly reliable pilot path, supportability, packaging, and operator trust over breadth |
| Auth / RBAC | Current tenant model plus named service accounts is enough for pilots; do not jump to SSO-first work yet |
| Packaging / deployment | Double down on release bundles, deployment references, and a repeatable pilot install path |
| Documentation | Invest heavily in install, migration, support, troubleshooting, and demo/validation runbooks |
| Support model | Keep community support best effort; define a paid pilot-support path outside the repo |
| Telemetry | Favor opt-in diagnostics, logs, exports, and support packets over mandatory usage tracking |
| Licensing | Keep the core Apache 2.0 |
| Pricing | Fixed assessment package, paid pilot package, optional retainer, and scoped implementation work are all credible |

### What this model would prioritize next

- durable VMware-to-Proxmox pilot hardening
- better packaging and installability for pilot deployments
- stronger support and observability workflows
- repeatable demo and validation environments
- clearer design-partner onboarding and pilot runbooks

### What this model would delay

- enterprise SSO and directory integration
- per-feature entitlement systems
- hosted-control-plane complexity
- heavy billing infrastructure
- broad MSP-first workflow investment

## Path 2: Open-Core Control Plane

### What it is

Keep a meaningful community edition open, but reserve premium value for paid modules or a paid distribution.

For Viaduct, that premium line would likely need to live in one of these places:

- new proprietary modules added after the current Apache 2.0 core
- a paid supported distribution with extra packaging, identity, and governance features
- a hosted or managed control plane layered on top of the current core

### Important Viaduct-specific constraint

Viaduct is already published under Apache 2.0. That means the current open codebase cannot simply be “taken back” into a closed product. Open-core is only realistic if new premium value is created intentionally and cleanly on top of the already-open core.

There is also no current repo boundary that cleanly separates premium-only value from the core. Viaduct does not yet have:

- edition-aware packaging
- entitlement checks
- a proprietary control-plane layer
- a clearly separate premium plugin or module boundary

That makes open-core more than a pricing decision. It would require product and repository restructuring that the current stage does not justify.

### Pros

- creates a clearer path to recurring software revenue
- keeps a community edition alive
- gives a future mechanism for monetizing enterprise features without fully closing the project

### Cons

- Viaduct does not yet have a clean premium boundary
- premature gating would likely put the most important hardening work into the wrong bucket
- can damage contributor and evaluator trust if the line between open and paid feels artificial
- adds licensing, entitlement, and packaging complexity now

### Product implications

| Area | Consequence |
| --- | --- |
| Roadmap | Pressure to define premium boundaries before the core pilot path is fully hardened |
| Auth / RBAC | Likely pushes enterprise identity, deeper RBAC, and audit history into the paid line |
| Packaging / deployment | Requires distinct community and commercial distributions or modules |
| Documentation | Requires explicit feature-boundary docs, entitlement docs, and edition comparison tables |
| Support model | Community plus paid support tiers become formal, not implied |
| Telemetry | Increased pressure for entitlement and usage tracking, even if it should remain opt-in |
| Licensing | Must preserve the existing Apache 2.0 core while carefully licensing new premium surfaces |
| Pricing | Likely annual subscription per tenant, site, cluster, or managed environment |

### What this model would prioritize next

- identity, governance, and operationally credible packaging features that can justify a paid tier
- clearer multi-tenant administration boundaries
- edition-aware docs and release packaging

### What this model would delay

- pure hardening of the existing core path if it does not map cleanly to paid versus free
- simplicity in community messaging
- contributor trust if the boundary is introduced clumsily

## Path 3: Commercial Packaged Platform Or Appliance

### What it is

Sell Viaduct primarily as a supported commercial distribution or appliance-like deployment:

- signed release bundles or images
- supported upgrade path
- defined deployment architecture
- paid support and commercial commitments

This could still coexist with open source, but the commercial distribution becomes the primary product.

### Why it is attractive

- serious infrastructure buyers often want a supported package, not just a repo
- it gives a clear story for commercial support, deployment, and accountability
- it can eventually support recurring subscription revenue better than services alone

### Why it is early for Viaduct

- Viaduct is not yet at the point where it should promise appliance-grade operational polish
- current auth, upgrade, support, and deployment expectations are still pilot-grade
- the product still needs more real-user and pilot evidence before a heavy commercial wrapper is the main story

### Product implications

| Area | Consequence |
| --- | --- |
| Roadmap | Prioritize installability, upgrades, rollback, deployment architecture, and support lifecycle hardening |
| Auth / RBAC | Serious pressure for SSO, stronger admin separation, credential lifecycle, and support-safe access patterns |
| Packaging / deployment | Requires signed artifacts, versioned upgrade guarantees, deployment topology docs, and likely image-based install paths |
| Documentation | Docs must read like a product manual, not just repo docs |
| Support model | Requires formal severity definitions, response expectations, and support channels |
| Telemetry | Strong pull toward opt-in health reporting and support bundles; mandatory telemetry would be risky for trust |
| Licensing | Could remain Apache 2.0 for the core but usually needs a commercial support agreement and possibly a proprietary distribution layer |
| Pricing | Annual per site, per cluster, per host band, or per managed environment is plausible |

### What this model would prioritize next

- product-grade installers and upgrade guarantees
- support lifecycle and issue response discipline
- stronger deployment references and packaged evaluation-to-pilot transitions

### What this model would delay

- experimental breadth
- consulting-heavy custom work
- community-first messaging if the commercial distro becomes dominant too early

## Path 4: Portfolio Product For Consulting Lead Generation

### What it is

Keep Viaduct mostly as a public proof asset and lead magnet for consulting work, with no strong push to become a deeply supported product.

### Pros

- lowest product-operating burden
- easy to align with bespoke advisory and migration consulting
- keeps open-source credibility if handled honestly

### Cons

- weak long-term product outcome
- makes it easier to defer hardening, supportability, packaging discipline, and trust controls
- buyers who want a real platform will quickly sense the difference
- design-partner feedback will skew toward services asks rather than product truth

### Product implications

| Area | Consequence |
| --- | --- |
| Roadmap | Favors demos, breadth, and consulting-friendly examples over deep product hardening |
| Auth / RBAC | Pilot-grade controls remain “good enough” longer |
| Packaging / deployment | Evaluation packages matter more than supported deployment discipline |
| Documentation | Sales-enablement and showcase docs tend to outrank operator docs |
| Support model | Mostly ad hoc consulting support |
| Telemetry | Minimal need beyond diagnostics |
| Licensing | Apache 2.0 remains easy |
| Pricing | Consulting day rates, assessments, and bespoke project work |

### What this model would prioritize next

- polished demos
- consulting collateral
- breadth that helps conversations start

### What this model would delay

- repeatable product operations
- boring reliability
- serious support commitments

## Path 5: Internal-Only Or Showcase Product

### What it is

Treat Viaduct mainly as an engineering showcase, internal tool, or research platform rather than a product with a real external commercial path.

### Pros

- maximum freedom to experiment
- lowest commercial expectation burden
- no need to solve pricing or support now

### Cons

- directly conflicts with the repo’s current open-source, packaging, docs, and pilot direction
- wastes the work already done to narrow the wedge and harden the operator path
- produces little pressure to fix the real trust and support gaps that matter for external users

### Product implications

| Area | Consequence |
| --- | --- |
| Roadmap | Breadth and experimentation can outrank reliability and packaging |
| Auth / RBAC | Can remain informal longer, which weakens external credibility |
| Packaging / deployment | Packaging becomes optional instead of product-critical |
| Documentation | Public docs become less trustworthy as a support promise |
| Support model | None or purely informal |
| Telemetry | Whatever is convenient internally |
| Licensing | Apache 2.0 can remain, but the public repo becomes strategically underused |
| Pricing | None |

### What this model would prioritize next

- experimentation
- architecture exploration
- connector breadth

### What this model would delay

- real customers
- serious pilots
- supportable releases

## 4. Recommendation

Viaduct should choose:

**Open source plus paid design-partner services and pilot delivery**

This is the strongest path right now because it is the only model that cleanly matches all of these realities at once:

- the product is real enough to solve an urgent problem
- the hardest current value is still trust-heavy and operator-guided
- the repo is already public and Apache 2.0
- the pilot path still needs more hardening before a heavier product wrapper makes sense

### Why this recommendation is correct now

### It monetizes the strongest current value

Viaduct is strongest today at:

- mixed-estate discovery
- readiness reduction
- approval-ready first-wave planning
- supervised pilot visibility and control

Those are high-value services-led outcomes right now, not yet a self-serve enterprise-platform outcome.

### It keeps the roadmap honest

This model says:

- harden the core path first
- learn from paid pilots
- keep the public product promise tight
- avoid building licensing machinery before the product boundary is stable

That is much healthier than trying to invent a premium edition around incomplete hardening work.

### It preserves option value

If Viaduct wins real pilots, it can still evolve later into:

- a supported commercial distribution
- a commercial packaged platform
- selective premium modules layered on top of the open core

Starting with services does not close those doors. Starting with open-core or appliance positioning too early does create avoidable product and trust debt.

### It fits serious infrastructure-software expectations

Infrastructure operators expect:

- transparent support boundaries
- clear licensing
- stable packaging
- opt-in, support-oriented diagnostics rather than hidden telemetry

The services-first path can satisfy those expectations now. The heavier commercial paths would promise more than the current repo should yet claim.

### Immediate commercial offer shape

If Viaduct follows this recommendation, the first commercial offers should be simple and repeatable:

- assessment package: discovery, readiness review, first-wave definition, and approval-ready outputs
- pilot enablement package: supported deployment help plus supervised VMware-to-Proxmox pilot support for the named path
- support retainer: bounded remote support for the supported pilot workflow and packaged deployment

That is more credible right now than pretending Viaduct should already be sold as an annual per-host platform subscription.

### Offer boundaries and red lines

These offers should stay narrow:

- assessment package:
  - includes supported discovery, readiness review, first-wave definition, report export, and operator handoff
  - does not include custom connector promises, unsupported target motions, or product-roadmap commitments
- pilot enablement package:
  - includes packaged deployment help, tenant and service-account setup, observability setup, runbook review, and supervised pilot support for the named VMware-to-Proxmox motion
  - does not include a promise of zero-touch cutover, unsupported target motions, or fleet-scale migration factory behavior
- support retainer:
  - includes bounded remote support for the supported pilot path and packaged deployment
  - does not imply 24x7 coverage, MSP-style NOC obligations, or a full enterprise support organization

If a prospect wants unsupported motions, enterprise IAM, MSP workflow breadth, or bespoke control-plane work immediately, that should be treated as out of the current product scope rather than quietly pulled into the roadmap.

### What Viaduct should actively avoid right now

Do not do these in the next commercialization phase:

- do not reposition Viaduct as an MSP-first multi-customer platform
- do not introduce open-core gating around the current Apache 2.0 pilot path
- do not sell the KVM lab or prepared demo state as proof of live production certification
- do not promise enterprise support SLAs that the current support model cannot meet
- do not take custom feature work for unsupported motions and describe it as near-term product commitment

## 5. Decision Needed Now Versus Later

### Decide now

- Viaduct is becoming a real open-source product with paid pilot and design-partner services, not just a showcase repo.
- The VMware-exit assessment-to-pilot wedge stays primary.
- The Apache 2.0 core stays intact.
- Mandatory phone-home telemetry and aggressive license gating are out of bounds for the current stage.
- The roadmap should favor pilot hardening, packaging, supportability, docs, and field validation over breadth.

### Decide later

Do not force these decisions yet:

- whether Viaduct should become open-core
- whether there should be a supported commercial distribution or appliance
- whether enterprise SSO, deeper RBAC, or hosted control-plane features become a paid line
- whether pricing should shift from services packages to annual software subscriptions
- whether opt-in product telemetry becomes valuable enough to formalize beyond diagnostics and support packets

Revisit those decisions after:

- at least two or three paid design-partner or pilot engagements
- stronger evidence that the primary reliability path survives real usage
- clearer proof that buyers want a supported product package, not just help getting through the first wave

### Explicitly not recommended now

These are not neutral alternatives at the current stage:

- open-core now: wrong time, no clean premium boundary, and too likely to distort hardening priorities
- appliance-first now: too support-heavy and too packaging-heavy for current pilot maturity
- consulting-lead-gen-only posture: undercuts the product discipline already established in the repo

## 6. What The Recommended Path Should Prioritize Next

If Viaduct follows the recommended path, the next commercialization-supporting product work should be:

1. harden the VMware-to-Proxmox pilot path until it is boringly reliable
2. improve installability, packaging, upgrade confidence, and rollback clarity for pilot deployments
3. make support and observability workflows strong enough for paid pilot work
4. keep docs, demos, and validation assets tightly aligned to the one wedge
5. define repeatable paid pilot deliverables instead of drifting into custom consulting

### What this means in the repo in the next 90 days

- keep [SUPPORT.md](../SUPPORT.md) honest as best-effort community support; do not blur public GitHub support with paid pilot support
- keep [v1-scope.md](v1-scope.md) and [support-matrix.md](reference/support-matrix.md) aligned to one supported live motion
- keep release and packaging work centered on [RELEASE.md](../RELEASE.md), `make release-gate`, and `make package-release-matrix`
- keep auth work centered on the current tenant and service-account model, not SSO-first expansion
- keep telemetry limited to observability, request correlation, audit/report exports, and support packets unless a later explicit decision changes that
- reject feature additions that do not make the assessment-to-pilot path more supportable, more trustworthy, or easier to deliver repeatedly

## 7. What The Recommended Path Should Delay

This path should deliberately delay:

- enterprise-platform identity work that exceeds the current pilot trust model
- feature gating and license enforcement inside the current core path
- hosted control-plane or SaaS-first assumptions
- MSP-first workflow expansion
- roadmap additions that do not strengthen the assessment-to-pilot path

## 8. Self-Review And Corrections

This recommendation is intentionally conservative.

- It does sacrifice near-term recurring-software ambition in favor of product honesty and field learning.
- It assumes the maintainer wants Viaduct to become a real external product, not only a consulting funnel.
- It risks services-led roadmap drift if design-partner work is not aggressively filtered through [v1-scope.md](v1-scope.md) and [primary-reliability-path.md](operations/primary-reliability-path.md).

That conservatism is still the right call now.

The current repo does not yet justify:

- enterprise-platform promises
- aggressive open-core segmentation
- appliance-grade support commitments

If Viaduct proves the pilot path in the field, the right next decision is not “how do we monetize every feature?” It is “which paid product wrapper best matches the proven path?” That is a later decision. The right decision now is to monetize pilot success and product hardening directly.
