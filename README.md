# Viaduct
> Hypervisor-agnostic workload migration and lifecycle management.

[![CI](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml/badge.svg)](https://github.com/eblackrps/viaduct/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/eblackrps/viaduct)](https://github.com/eblackrps/viaduct/blob/main/LICENSE)

## Why Viaduct
Broadcom's VMware licensing changes have forced infrastructure teams to revisit assumptions they expected to keep for years. Industry reporting cited in the Viaduct project plan shows 86% of organizations actively reducing their VMware footprint, while most migration tooling still pushes workloads into a single destination platform. That leaves teams choosing between vendor lock-in on the way in or vendor lock-in on the way out. Viaduct exists as the neutral bridge: an open source control plane for discovering, planning, migrating, and operating workloads across mixed hypervisor estates.

## Project Status
Viaduct is in active bootstrap and early implementation. The repository is ready for open development, CI-backed contributions, and Phase 1 discovery work.

## What It Does
- Discovery Engine: Connects to hypervisor APIs and normalizes workload inventory into a common schema.
- Dependency Mapper: Correlates workloads with surrounding network, storage, DNS, and backup relationships.
- Migration Orchestrator: Executes declarative workload migrations with reversible workflows.
- Lifecycle Manager: Keeps mixed-platform environments healthy with drift, policy, and operational insights.

## Quick Start
```bash
make build

./bin/viaduct discover --source https://vcenter.example.com --type vmware
./bin/viaduct discover --source https://proxmox.example.com:8006 --type proxmox
```

## Supported Platforms
| Platform | Status |
| --- | --- |
| VMware vSphere | In Progress |
| Proxmox VE | In Progress |
| Microsoft Hyper-V | Planned |
| KVM/libvirt | Planned |
| Nutanix AHV | Planned |

## Roadmap
- Phase 0: Bootstrap the project foundation, schema, CLI, CI, and contributor workflow.
- Phase 1: Ship the Discovery Engine MVP with VMware and Proxmox inventory normalization.
- Phase 2: Add cold migration orchestration, dashboard workflows, and Veeam integration.
- Phase 3: Expand into lifecycle management, warm migration, and multi-tenancy.

See [ROADMAP.md](ROADMAP.md) for the public roadmap and the phase documents under [docs/roadmaps](docs/roadmaps).

## Project Docs
- [Architecture Overview](docs/architecture.md)
- [Roadmap](ROADMAP.md)
- [Security Policy](SECURITY.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Contributing Guide](CONTRIBUTING.md)

## Contributing
Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, workflow, and issue guidance.

## License
Viaduct is licensed under the [Apache License 2.0](LICENSE).
