# Viaduct
> Open-source, evaluation-ready control plane for mixed virtualization discovery, planning, and supervised pilot workflows.

[![CI](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml/badge.svg)](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/eblackrps/viaduct)](https://github.com/eblackrps/viaduct/blob/main/LICENSE)

Viaduct helps operators understand mixed virtualization estates before they commit to a migration path. The repository combines a Go backend, REST API, CLI, React dashboard, and a small public site around one shared model for discovery, dependency-aware planning, migration readiness, saved pilot workspaces, and exported operator evidence.

## Current Status

Viaduct is evaluation-ready and under active development. The strongest current story is:
- mixed-estate discovery
- dependency-aware assessment
- readiness and simulation
- supervised pilot planning
- operator-visible reporting and export

The default first-run experience is the pilot workspace flow: create workspace, discover, inspect, simulate, save plan, and export report. Start in the local lab or a supervised pilot environment first. Do not assume unattended migration breadth across every connector pair.

## Why Viaduct

Many teams do not need more abstract migration talk. They need to know what exists, what depends on what, what should move first, and what evidence is good enough to approve a pilot. Viaduct is aimed at that assessment-to-pilot gap.

It is strongest when operators need:
- one normalized inventory across mixed platforms
- dependency context before planning a first wave
- a persisted assessment record instead of disconnected notes and screenshots
- a CLI, API, and dashboard that reflect the same state

## What You Can Evaluate Today

- Discovery: normalize inventory from VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup systems into the shared model.
- Dependency mapping: build workload, network, storage, and backup graph context before execution.
- Pilot workspace workflow: persist source connections, snapshots, graph outputs, target assumptions, readiness results, saved plans, approvals, notes, and exported reports in one operator record.
- Migration planning: use declarative specs, preflight checks, saved dry-run plans, execution windows, approval requirements, checkpoints, resume support, and rollback state.
- Lifecycle analysis: review drift, policy, cost, remediation guidance, and backup portability inputs in the same control plane.
- Multi-tenancy and extensibility: use tenant-scoped APIs, service accounts, PostgreSQL-backed state, and the community plugin host.

## Platform Coverage

| Platform / Integration | Status | Notes |
| --- | --- | --- |
| VMware vSphere | Implemented | vCenter discovery with VM and infrastructure metadata. |
| Proxmox VE | Implemented | REST-based discovery and a common target in current pilot examples. |
| Microsoft Hyper-V | Implemented | WinRM and PowerShell-backed inventory collection. |
| KVM / libvirt | Implemented | XML-backed fallback works out of the box; live libvirt support is available behind the `libvirt` build tag. |
| Nutanix AHV | Implemented | Prism Central v3 inventory collection. |
| Veeam Backup & Replication | Implemented | Backup discovery, restore-point correlation, and portability planning support. |
| Community plugins | Supported | gRPC plugin host with manifest and compatibility checks. |

## First Run

The cleanest path is the local lab in [examples/lab](examples/lab).

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
make build
./bin/viaduct version

export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

In another terminal:

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants

curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://localhost:8080/api/v1/service-accounts
```

Then start the dashboard in `web/`, sign in with `lab-operator-key`, and run the workspace-first flow.

Use these entrypoints next:
- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Detailed quickstart: [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md)
- Pilot workspace guide: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)

## Repository Surfaces

- `cmd/viaduct/`: CLI entrypoints.
- `internal/`: discovery, dependency mapping, migration orchestration, lifecycle analysis, API server, and state store packages.
- `web/`: React dashboard for operator workflows.
- `site/`: separate static public site for GitHub Pages.
- `examples/`: lab, deployment, and plugin evaluation assets.
- `docs/`: deeper reference, operations, and product-scope documentation.

## Verification

`make release-gate` is the canonical verification path. It keeps backend checks, web build validation, coverage enforcement, packaging, and the lab-oriented smoke flow aligned in one command.

Other high-signal commands:

```bash
make build
make certification-test
make soak-test
make contract-check
make package-release-matrix
```

## Documentation

- Overview and status: [README.md](README.md)
- Install: [INSTALL.md](INSTALL.md)
- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Upgrade: [UPGRADE.md](UPGRADE.md)
- Release process: [RELEASE.md](RELEASE.md)
- Support: [SUPPORT.md](SUPPORT.md)
- Security: [SECURITY.md](SECURITY.md)
- Documentation index: [docs/README.md](docs/README.md)
- API contract: [docs/reference/openapi.yaml](docs/reference/openapi.yaml)
- Support matrix: [docs/reference/support-matrix.md](docs/reference/support-matrix.md)

## Contributing

Contributions are welcome. Keep docs, examples, and public-facing assets aligned with operator-visible behavior. See [CONTRIBUTING.md](CONTRIBUTING.md) for workflow and verification expectations.

## Support And Security

- Usage, evaluation, and troubleshooting: [SUPPORT.md](SUPPORT.md)
- Security reporting: [SECURITY.md](SECURITY.md)
- Project change history: [CHANGELOG.md](CHANGELOG.md)
- Community expectations: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## License

Viaduct is licensed under the [Apache License 2.0](LICENSE).
