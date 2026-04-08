# Roadmap

Viaduct has completed its first four implementation phases and is now in the release and ecosystem-launch stage. This roadmap distinguishes completed implementation milestones from the current release-readiness and ecosystem work so contributors can see what is stable, what is still being hardened, and where new work adds the most leverage.

## Current Status
- Phase 0 through Phase 4 are complete in the repository.
- The current focus is early-product hardening: packaging, operator experience, release verification, supportability, and design-partner readiness.
- Short-term work should favor compatibility, contract clarity, observability, certification, installability, and contributor leverage over speculative rewrites or broad new surface area.

## Current Priorities
- narrow the public product story around assessment, planning, and supervised pilot workflows
- deepen operational confidence with live-environment certification, soak validation, and reproducible release gating
- strengthen install, upgrade, deployment, and rollback guidance so packaged adoption is low-friction
- improve observability, API-contract discipline, and operator diagnostics without breaking current workflows
- strengthen connector and plugin adoption with compatibility rules, reference examples, and contributor documentation

## Phase 0: Foundation
Status: complete

Key deliverables:
- Go module, Makefile, lint configuration, and CI
- Universal inventory schema and connector interface
- Cobra CLI skeleton with the core commands
- Public project documentation and contribution standards

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
- MSP-oriented multi-tenancy and connector plugin hosting

Detail: [Phase 3 Archive](docs/roadmaps/phase-3.md)

## Phase 4: Scale, Extensibility, And Automation
Status: complete

Key deliverables:
- execution windows, approval gates, wave planning, checkpoints, and resume support
- lifecycle remediation guidance and simulation flows
- tenant summary reporting and stronger operator diagnostics
- stronger release gating with coverage enforcement
- plugin lifecycle hardening

Detail: [Phase 4 Archive](docs/roadmaps/phase-4.md)

## Release And Ecosystem Launch
Current focus:
- a clearer early-product wedge with explicit trust boundaries
- polished release bundles and install paths
- upgrade and rollback guidance
- operator runbooks and reference environments
- plugin author onboarding and validation guidance
- clearer support and compatibility expectations

Expected outcomes:
- a credible early product centered on assessment, planning, and supervised pilot use
- a clean stable release flow that is easy to evaluate from source or packaged artifacts
- documentation that matches the current architecture, API surface, and workflows
- contributor guidance that reinforces release-gate discipline and compatibility rules

Contribution opportunities:
- connector certification and compatibility validation
- richer operator examples and reference environments
- packaging and installation improvements across platforms
- observability, diagnostics, and release-engineering automation
- documentation clarity where code and operator workflow still feel too implicit

## Guiding Principles
- Keep the universal schema in `internal/models/` as the system of record.
- Build connectors as isolated plugins behind the shared interface.
- Prefer reversible migration workflows and explicit verification.
- Treat mixed-hypervisor operations as a durable operating model, not a temporary bridge.
