# Viaduct
> Hypervisor-agnostic workload migration and lifecycle management.

[![CI](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml/badge.svg)](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/eblackrps/viaduct)](https://github.com/eblackrps/viaduct/blob/main/LICENSE)

Viaduct is an open source control plane for discovering, migrating, and operating workloads across mixed virtualization environments. It gives operators a shared backend, CLI, API, and dashboard for inventory normalization, dependency-aware planning, cold and warm migration workflows, lifecycle analysis, backup portability, and tenant-scoped operations without forcing a single-hypervisor end state.

## Why Viaduct
Broadcom's VMware licensing changes forced many teams into urgent platform decisions, but most migration tooling still assumes a one-time move into a single destination. Viaduct is built for operators who need a durable mixed-platform operating model: discover what exists, understand the blast radius, move workloads safely, preserve backup coverage, and keep managing cost, policy, and drift after cutover.

## Project Status
Viaduct is ready for broad evaluation, design-partner pilots, and community contribution. The repository includes multi-platform discovery, dependency graphing, declarative migration orchestration, warm-migration primitives, lifecycle remediation, backup portability, multi-tenancy with service accounts and quota controls, plugin hosting, a web dashboard, a standalone public site, reproducible release packaging, and a shared release gate for CI and local verification.

The current early-product wedge is VMware-exit mixed-estate discovery and migration readiness assessment with approval-ready pilot planning. Discovery, planning, operator visibility, and packaged evaluation are the strongest current surfaces. The default dashboard path is now a first-class pilot workspace that keeps source connections, snapshots, graph output, readiness results, saved plans, approvals, notes, and report exports in one operator-owned record. Live execution paths should still be validated in a lab or pilot environment before they are treated as routine production automation.

## Supported Capabilities
- Discovery engine: Collects normalized inventory from VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup systems into a universal schema.
- Dependency mapping: Builds graph views across workloads, networks, storage, and backup relationships to support safer migration planning.
- Migration orchestration: Supports declarative planning, preflight validation, cold and warm migration flows, execution windows, approval gates, checkpoints, resume support, and rollback.
- Lifecycle analysis: Evaluates cost, policy, and drift, then turns those signals into remediation guidance and simulation output.
- Pilot workspace workflow: Ties together assessment state, source connections, discovery baselines, readiness simulation, saved plans, operator notes, approvals, and exported reports.
- Multi-tenancy and extensibility: Provides tenant-scoped API access, service-account and role-based access controls, persistent state backends, and a gRPC-based plugin host for community connectors.
- Operator surfaces: Exposes the same core workflows through a CLI, REST API, and React dashboard.
- Operability: Ships request correlation, tenant-scoped audit and reporting routes, Prometheus-style metrics, an OpenAPI reference, deployment examples, and packaged release metadata.

## Supported Connectors And Integrations
| Platform / Integration | Status | Notes |
| --- | --- | --- |
| VMware vSphere | Implemented | vCenter discovery with VM and infrastructure metadata. |
| Proxmox VE | Implemented | REST-based inventory discovery; commonly used as a target in migration planning examples. |
| Microsoft Hyper-V | Implemented | WinRM / PowerShell-based inventory collection. |
| KVM / libvirt | Implemented | XML-backed fallback works out of the box; live libvirt support is available with the `libvirt` build tag. |
| Nutanix AHV | Implemented | Prism Central v3 inventory collection. |
| Veeam Backup & Replication | Implemented | Backup discovery, restore-point correlation, and portability planning support. |
| Community plugins | Supported | gRPC plugin host with validation for health, platform identity, and normalized discovery results. |

## Architecture Summary
- Discovery engine: `internal/discovery/` collects and normalizes inventory from built-in and plugin connectors.
- Dependency mapper: `internal/deps/` builds graph relationships across workload, network, storage, and backup data.
- Migration orchestrator: `internal/migrate/` handles spec parsing, planning, preflight, execution, checkpoints, resume, verification, and rollback.
- Lifecycle manager: `internal/lifecycle/` handles drift, cost, policy, simulation, and remediation workflows.

See [docs/architecture.md](docs/architecture.md) for the detailed architecture view and [docs/reference/support-matrix.md](docs/reference/support-matrix.md) for validation scope, runtime expectations, and connector notes.

## Quick Start
```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
make build
./bin/viaduct version

./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
./bin/viaduct serve-api --port 8080
curl http://localhost:8080/api/v1/about

cd web
npm ci
npm run dev
```

The local KVM lab under [examples/lab](examples/lab) gives you a first-run workflow without needing a live hypervisor. Start with [QUICKSTART.md](QUICKSTART.md) for the top-level guide or [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md) for the detailed walkthrough.

For the workspace-first dashboard flow, bootstrap the lab tenant and service account from `examples/lab/`, then authenticate through the dashboard's runtime bootstrap screen. The full operator path is documented in [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md).

## Build And Test
```bash
go mod tidy
make build
go test ./... -v -race -count=1
make certification-test
make soak-test
make plugin-check
make contract-check
go vet ./...
golangci-lint run ./...
cd web && npm run build
make release-gate
```

`make release-gate` is the canonical local verification path. It runs the backend checks, CLI smoke checks, soak coverage, dashboard build, coverage reporting, and release packaging in the same flow CI uses.

## Installation And Operations
- Install: [INSTALL.md](INSTALL.md)
- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Upgrade: [UPGRADE.md](UPGRADE.md)
- Release process: [RELEASE.md](RELEASE.md)
- Pilot workspace flow: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)
- Configuration reference: [docs/reference/configuration.md](docs/reference/configuration.md)
- Migration operations: [docs/operations/migration-operations.md](docs/operations/migration-operations.md)
- Auth, role, and auditability model: [docs/operations/auth-role-audit-model.md](docs/operations/auth-role-audit-model.md)
- Demo runbook: [docs/operations/demo-runbook.md](docs/operations/demo-runbook.md)
- Observability requirements: [docs/operations/observability-requirements.md](docs/operations/observability-requirements.md)
- Real user validation plan: [docs/operations/real-user-validation-plan.md](docs/operations/real-user-validation-plan.md)
- Backup portability: [docs/operations/backup-portability.md](docs/operations/backup-portability.md)
- Multi-tenancy: [docs/operations/multi-tenancy.md](docs/operations/multi-tenancy.md)
- Troubleshooting: [docs/reference/troubleshooting.md](docs/reference/troubleshooting.md)
- API contract: [docs/reference/openapi.yaml](docs/reference/openapi.yaml)
- Deployment examples: [examples/deploy/README.md](examples/deploy/README.md)

## Documentation Index
- Repository docs index: [docs/README.md](docs/README.md)
- Commercialization options: [docs/commercialization-options.md](docs/commercialization-options.md)
- Early product focus: [docs/early-product-focus.md](docs/early-product-focus.md)
- Beachhead use case analysis: [docs/beachhead-use-case.md](docs/beachhead-use-case.md)
- V1 scope definition: [docs/v1-scope.md](docs/v1-scope.md)
- Backend contract hardening: [docs/backend-contract-hardening.md](docs/backend-contract-hardening.md)
- Auth, role, and auditability model: [docs/operations/auth-role-audit-model.md](docs/operations/auth-role-audit-model.md)
- Demo runbook: [docs/operations/demo-runbook.md](docs/operations/demo-runbook.md)
- Pilot workspace flow: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)
- Observability requirements: [docs/operations/observability-requirements.md](docs/operations/observability-requirements.md)
- Primary reliability path: [docs/operations/primary-reliability-path.md](docs/operations/primary-reliability-path.md)
- Real user validation plan: [docs/operations/real-user-validation-plan.md](docs/operations/real-user-validation-plan.md)
- Release drafts: [docs/releases/README.md](docs/releases/README.md)
- Public site source: [site/README.md](site/README.md)
- Architecture overview: [docs/architecture.md](docs/architecture.md)
- Support matrix: [docs/reference/support-matrix.md](docs/reference/support-matrix.md)
- Plugin author guide: [docs/reference/plugin-author-guide.md](docs/reference/plugin-author-guide.md)
- Plugin certification guide: [docs/reference/plugin-certification.md](docs/reference/plugin-certification.md)
- Codebase map: [docs/reference/codebase-map.md](docs/reference/codebase-map.md)
- Historical phase roadmaps: [docs/roadmaps/README.md](docs/roadmaps/README.md)

## Extensibility
- Plugin and connector author guide: [docs/reference/plugin-author-guide.md](docs/reference/plugin-author-guide.md)
- Plugin certification checklist: [docs/reference/plugin-certification.md](docs/reference/plugin-certification.md)
- Example plugin: [examples/plugin-example/README.md](examples/plugin-example/README.md)
- Deployment examples: [examples/deploy/README.md](examples/deploy/README.md)
- Example and lab assets: [examples/README.md](examples/README.md)
- Sample configs and policies: [configs/README.md](configs/README.md)

## Roadmap
- Phase 0: Foundation and project scaffolding completed.
- Phase 1: Discovery engine MVP completed.
- Phase 2: Cold migration, dashboard, and Veeam integration completed.
- Phase 3: Warm migration, lifecycle management, and multi-tenancy completed.
- Phase 4: Scale, extensibility, and automation foundation completed.
- Release and ecosystem launch: packaging, installability, operator runbooks, ecosystem guidance, adoption readiness, and early-product hardening continue as the current refinement track after the first tagged release.

See [ROADMAP.md](ROADMAP.md) for the public roadmap, [docs/early-product-focus.md](docs/early-product-focus.md) for the current product wedge and trust boundaries, [docs/beachhead-use-case.md](docs/beachhead-use-case.md) for the wedge decision behind that framing, [docs/v1-scope.md](docs/v1-scope.md) for the frozen first-release support promise, [docs/backend-contract-hardening.md](docs/backend-contract-hardening.md) for the contract-stabilization work needed to support that promise, and [docs/roadmaps/README.md](docs/roadmaps/README.md) for the archived phase documents.

## Contributing
Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, compatibility expectations, testing, documentation expectations, and release-gate workflow guidance.

## Support And Security
- Support and usage guidance: [SUPPORT.md](SUPPORT.md)
- Security reporting policy: [SECURITY.md](SECURITY.md)
- Community expectations: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- Project change history: [CHANGELOG.md](CHANGELOG.md)

## License
Viaduct is licensed under the [Apache License 2.0](LICENSE).
