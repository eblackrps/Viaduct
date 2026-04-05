# Changelog

All notable changes to Viaduct should be documented in this file.

This changelog tracks published releases and the major implementation milestones that shaped the current repository state.

## [1.0.0] - 2026-04-05

### Highlights
- shipped the first tagged Viaduct release with a release-gated CLI, API, dashboard, install scripts, packaged web assets, checksums, and release manifest generation
- delivered multi-platform discovery for VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup inventory
- delivered dependency-aware migration planning, cold and warm migration workflows, execution windows, approval gates, checkpoints, resume support, verification, and rollback
- delivered lifecycle cost, policy, drift, remediation, and simulation workflows
- delivered tenant-scoped API access, persistent state backends, plugin hosting, contributor docs, operator runbooks, and example lab environments

## Unreleased

- no unreleased changes currently tracked

## [1.1.0] - 2026-04-05

### Current Stable Surface
- multi-platform discovery for VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup inventory
- dependency graph construction across workload, network, storage, and backup metadata
- declarative cold and warm migration orchestration with preflight checks, execution windows, approval gates, checkpoints, resume support, verification, and rollback
- lifecycle cost, policy, drift, remediation, and simulation workflows
- multi-tenancy, tenant-scoped API access, persistent state backends, and plugin hosting
- React dashboard, reproducible release packaging, install scripts, and a shared release gate

### Operability And Ecosystem
- added connector certification coverage and a tagged soak-test path for large-wave migration exercises
- added deployment reference assets for Docker Compose, systemd, and Kubernetes-based pilot environments
- added plugin manifest validation and config-aware plugin connection handling so plugin connectors receive the same auth and transport settings as built-in connectors
- added tenant-scoped audit exports, request correlation headers, API metrics, and basic tenant rate limiting to improve diagnostics without changing core workflows

### Maintenance
- refreshed the dashboard stack to React 19, Vite 8, and `@vitejs/plugin-react` 6 with a Node 20.19+ baseline
- grouped Dependabot updates more conservatively and ignored Docker base-image major jumps until they are evaluated intentionally
- aligned the web TypeScript configuration with Vite's bundler module resolution and deferred semver-major Tailwind CSS and TypeScript jumps until a dedicated migration pass

### Repository Professionalization
- aligned top-level docs, roadmap archives, examples, and community files with the implemented codebase
- added release-era install, quickstart, upgrade, release, support, and troubleshooting entrypoints
- improved directory onboarding for docs, configs, examples, API assets, tests, and the dashboard

## Historical Implementation Milestones

### Phase 4 Complete
- added execution windows, approval gates, wave planning, resume support, lifecycle remediation guidance, simulation flows, tenant summary reporting, and stronger release gating

### Phase 3 Complete
- added warm migration, lifecycle management, backup portability, KVM and Nutanix connectors, multi-tenancy, and plugin hosting

### Phase 2 Complete
- added cold migration orchestration, Veeam and Hyper-V discovery, the dependency graph, and the operator dashboard

### Phase 1 Complete
- added VMware and Proxmox discovery, normalization, state persistence, and discovery CLI workflows

### Phase 0 Complete
- created the repository foundation, universal schema, connector interfaces, CLI skeleton, CI, and project governance files
