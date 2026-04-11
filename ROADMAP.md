# Roadmap

Viaduct has completed its first four implementation phases. The current work is not a stack rewrite or a broadening of scope. It is a tightening pass around evaluation quality, operator clarity, and supervised pilot readiness.

## Current Status

- Phase 0 through Phase 4 are implemented in the repository.
- The current refinement track is evaluation-focused: packaging, installability, operator experience, documentation coherence, validation, and trust boundaries.
- Near-term work should favor compatibility, contract clarity, repeatable demos, and better proof over broad new surface area.

## Current Priorities

- keep the public story centered on assessment, planning, and supervised pilot workflows
- deepen confidence with live-environment certification, soak validation, and reproducible release verification
- improve install, upgrade, deployment, and rollback guidance for evaluation and pilot environments
- strengthen observability, API-contract discipline, and operator diagnostics without changing the architecture unnecessarily
- make connector and plugin adoption easier to evaluate through clearer examples and compatibility guidance

## Phase 0: Foundation
Status: complete

Key deliverables:
- Go module, Makefile, lint configuration, and CI
- universal inventory schema and connector interface
- Cobra CLI skeleton with the core commands
- public project documentation and contribution standards

Detail: [Phase 0 Archive](docs/roadmaps/phase-0.md)

## Phase 1: Discovery Engine MVP
Status: complete

Key deliverables:
- VMware vCenter VM, network, and storage discovery
- Proxmox VM, network, and storage discovery
- normalization into the universal schema
- state store wiring, CLI formatting, and integration coverage

Detail: [Phase 1 Archive](docs/roadmaps/phase-1.md)

## Phase 2: Cold Migration, Dashboard, And Veeam
Status: complete

Key deliverables:
- disk conversion and migration spec parsing
- cold migration orchestration, preflight checks, rollback, and network remapping
- Veeam backup job discovery and Hyper-V discovery
- React dashboard for inventory, migration workflows, and dependency views

Detail: [Phase 2 Archive](docs/roadmaps/phase-2.md)

## Phase 3: Lifecycle, Warm Migration, And Multi-Tenancy
Status: complete

Key deliverables:
- block replication and warm migration cutover
- cost modeling, policy engine, drift detection, and lifecycle views
- backup portability and lifecycle views
- KVM/libvirt and Nutanix AHV connectors
- multi-tenant administration and connector plugin hosting

Detail: [Phase 3 Archive](docs/roadmaps/phase-3.md)

## Phase 4: Scale, Extensibility, And Automation
Status: complete

Key deliverables:
- execution windows, approval requirements, wave planning, checkpoints, and resume support
- lifecycle remediation guidance and simulation flows
- tenant summary reporting and stronger operator diagnostics
- stronger release verification with coverage enforcement
- plugin lifecycle hardening

Detail: [Phase 4 Archive](docs/roadmaps/phase-4.md)

## Current Refinement Track

Focus areas:
- clearer assessment and supervised pilot product framing with explicit trust boundaries
- polished release bundles and install paths
- better upgrade and rollback guidance
- operator runbooks, demo assets, and evaluation materials
- stronger plugin author onboarding and compatibility guidance
- more consistent public docs and repo entrypoints

Expected outcomes:
- a careful ready for technical assessment release surface
- a tagged release flow that is easy to assess from source or packaged artifacts
- documentation that matches the current architecture and operator workflow
- contributor guidance that reinforces scope discipline and verification hygiene

Contribution opportunities:
- connector certification and compatibility validation
- richer operator examples and pilot references
- packaging and installation improvements across platforms
- observability, diagnostics, and release-engineering automation
- documentation cleanup where public wording still overstates maturity
