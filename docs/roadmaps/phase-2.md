# Phase 2 Roadmap

Milestone target: end of September 2026

Phase 2 expands Viaduct from discovery into cold migration orchestration and operator-facing UI.

## Primary Outcomes
- convert disks across major hypervisor formats
- parse declarative migration specs
- orchestrate cold migration flows with validation, rollback, and remapping
- integrate Veeam backup discovery and add Hyper-V discovery
- introduce the React dashboard and dependency visualization

## Planned Workstreams
1. Disk format conversion engine
2. Migration YAML spec parser
3. Cold migration orchestrator
4. Network remapping
5. Pre-flight checks
6. Rollback mechanism
7. Veeam backup job discovery
8. Hyper-V discovery connector
9. Inventory dashboard
10. Migration wizard
11. Dependency graph visualization
12. End-to-end migration testing

## Definition of Done
- a cold migration can be defined declaratively and executed through Viaduct
- disk conversion and rollback paths are verified
- operators can inspect inventory and migration plans through the dashboard
- backup and dependency context materially improves migration planning
