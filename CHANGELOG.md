# Changelog

All notable changes to Viaduct should be documented in this file.

This changelog tracks published releases and the major implementation milestones that shaped the current repository state.

## [1.5.0] - 2026-04-08

### Early Product Hardening
- narrowed the public product story around the VMware-exit assessment-to-pilot wedge with explicit beachhead, v1 scope, reliability-path, trust-control, observability, validation, demo, and commercialization artifacts
- aligned repo entrypoint docs so the current product direction, support boundary, and operator guidance are easier to evaluate from the packaged and source workflows

### API And Dashboard Trust Surfaces
- hardened the API contract with structured JSON error responses, stabilized migration command acknowledgements, and updated OpenAPI coverage for the operator-facing routes
- improved dashboard-side error handling so settings and report workflows preserve request correlation and support-grade failure detail instead of flattening backend errors into generic strings

### Documentation And Operator Readiness
- added presenter-ready demo assets, real-user validation templates, and commercialization decision guidance to support design-partner conversations and pilot packaging
- refreshed quickstart, configuration, troubleshooting, and multi-tenancy guidance so service accounts, trust controls, and the supported pilot workflow are documented more consistently

## [1.4.2] - 2026-04-08

### Release Reliability
- completed the dependency graph TypeScript fix so the D3 link endpoint handlers compile cleanly during `make release-gate`
- superseded the `v1.4.1` candidate tag before publishing a downloadable GitHub release

## [1.4.1] - 2026-04-08

### Release Reliability
- fixed the dashboard dependency graph typing so `make release-gate` can complete the web build and package the release bundle
- superseded the `v1.4.0` candidate tag before publishing a downloadable GitHub release

## [1.4.0] - 2026-04-08

### Dashboard Product Workflow
- reorganized the React dashboard around a clearer app shell, navigation model, and feature-oriented page structure
- turned migration planning into an operator workflow with intake, validation, saved-plan review, and execution-preparation states instead of a detached wizard
- improved inventory, dependency, and remediation surfaces so planning context stays connected to the broader operator view

### Operator Authentication And Configuration
- added dashboard support for service-account API keys alongside tenant API keys
- documented the new dashboard environment variable contract for local development and release packaging

### Public Web Presence
- added a standalone static `site/` for the public project surface
- added a GitHub Pages workflow to publish the site independently from the product dashboard build

## [1.3.0] - 2026-04-07

### Tenant Isolation And Operability
- added tenant-scoped permission enforcement and richer tenant introspection for service-account automation
- added store diagnostics, API build metadata, and operational metrics/reporting surfaces

### Backup And Plugin Ecosystem
- added backup continuity and backup-policy drift validation for post-migration portability checks
- added plugin manifest validation tooling and a release-facing plugin certification guide

### Release And Deployment Experience
- added OpenAPI contract checks to the release workflow and published the stable operator contract reference
- hardened deployment references for Docker Compose, systemd, and Kubernetes pilots

## [1.2.0] - 2026-04-07

### Tenant Security And Scale
- added tenant-scoped service accounts with viewer, operator, and admin roles for API authentication
- added role-gated tenant routes and a current-tenant introspection route without leaking API keys
- added tenant quotas for API request rate, snapshot count, and migration count

### Migration And API Correctness
- fixed current-inventory aggregation so the API no longer misses sources once snapshot history grows past twenty entries
- replaced brittle pending-approval summary detection with real migration-state decoding
- wired migration `credential_ref` resolution through the CLI config and API server connector-resolution paths
- added the `/api/v1/about` route for operator-visible build and compatibility metadata

### Plugin And Release Operability
- added optional plugin host-version compatibility markers in `plugin.json`
- added a machine-readable `dependency-manifest.json` to packaged release bundles
- expanded regression coverage for service-account auth, quota enforcement, plugin compatibility, packaging metadata, and summary correctness

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

## [1.0.0] - 2026-04-05

### Highlights
- shipped the first tagged Viaduct release with a release-gated CLI, API, dashboard, install scripts, packaged web assets, checksums, and release manifest generation
- delivered multi-platform discovery for VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup inventory
- delivered dependency-aware migration planning, cold and warm migration workflows, execution windows, approval gates, checkpoints, resume support, verification, and rollback
- delivered lifecycle cost, policy, drift, remediation, and simulation workflows
- delivered tenant-scoped API access, persistent state backends, plugin hosting, contributor docs, operator runbooks, and example lab environments

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
