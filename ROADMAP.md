# Roadmap

Viaduct is being built in four phases so the project can move from foundation work to real multi-hypervisor migration outcomes without losing architectural discipline.

## Current Status
- Phase 0 is the active bootstrap milestone.
- The repository foundation, CLI skeleton, inventory schema, CI workflow, and connector stubs are in place.
- The next implementation milestone is the Discovery Engine MVP.

## Phase 0: Foundation
Target outcome: a public, buildable repository with contributor guidance and the minimum code scaffolding needed to begin platform work.

Key deliverables:
- Go module, Makefile, lint configuration, and CI
- Universal inventory schema and connector interface
- Cobra CLI skeleton with the core commands
- VMware and Proxmox connector stubs
- Public project documentation and contribution standards

Detail: [Phase 0 Roadmap](docs/roadmaps/phase-0.md)

## Phase 1: Discovery Engine MVP
Target milestone: end of June 2026

Key deliverables:
- VMware vCenter VM, network, and storage discovery
- Proxmox VM, network, and storage discovery
- Normalization into the universal schema
- State store wiring, CLI formatting, and integration coverage
- Verification sweep and project housekeeping

Detail: [Phase 1 Roadmap](docs/roadmaps/phase-1.md)

## Phase 2: Cold Migration, Dashboard, and Veeam
Target milestone: end of September 2026

Key deliverables:
- Disk conversion and migration spec parsing
- Cold migration orchestration, pre-flight checks, rollback, and network remapping
- Veeam backup job discovery and Hyper-V discovery
- React dashboard for inventory, migration workflows, and dependency views
- End-to-end migration validation

Detail: [Phase 2 Roadmap](docs/roadmaps/phase-2.md)

## Phase 3: Lifecycle, Warm Migration, and Multi-Tenancy
Target milestone: end of January 2027

Key deliverables:
- Block replication and warm migration cutover
- Cost modeling, policy engine, and drift detection
- Backup portability and lifecycle views
- KVM/libvirt and Nutanix AHV connectors
- MSP-oriented multi-tenancy and community connector SDK

Detail: [Phase 3 Roadmap](docs/roadmaps/phase-3.md)

## Guiding Principles
- Keep the universal schema in `internal/models/` as the system of record.
- Build connectors as isolated plugins behind the shared interface.
- Prefer reversible migration workflows and explicit verification.
- Treat mixed-hypervisor operations as a durable operating model, not a temporary bridge.
