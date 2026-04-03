# Phase 1 Roadmap

Milestone target: end of June 2026

Phase 1 delivers the Discovery Engine MVP.

## Primary Outcomes
- discover VMware inventory from vCenter
- discover Proxmox inventory from Proxmox VE
- normalize workloads, network, and storage into the shared schema
- persist discovery state and expose useful CLI output formats
- add integration coverage and a final verification sweep

## Planned Workstreams
1. VMware VM discovery
2. VMware network and storage discovery
3. Proxmox VM discovery
4. Proxmox network and storage discovery
5. Normalization layer
6. State store wiring
7. CLI formatting
8. Output formatting
9. Integration tests
10. Verification sweep

## Definition of Done
- VMware and Proxmox connectors return normalized `DiscoveryResult` data
- discovery output is usable from the CLI
- integration tests cover representative connector flows
- the codebase is ready to begin migration orchestration work
