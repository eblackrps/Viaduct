# Viaduct
> Open-source control plane for virtualization migration assessment, planning, and controlled operator execution.

[![CI](https://github.com/eblackrps/Viaduct/actions/workflows/ci.yml/badge.svg)](https://github.com/eblackrps/Viaduct/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/eblackrps/Viaduct)](https://github.com/eblackrps/Viaduct/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/eblackrps/Viaduct?display_name=tag)](https://github.com/eblackrps/Viaduct/releases)

Viaduct helps operators discover mixed virtualization estates, map dependencies, build migration plans, and manage controlled migration work from one shared backend model. The repository combines a Go backend, REST API, CLI, React dashboard, and standalone public site around the same persisted inventory, workspace, planning, and reporting surfaces.

Versioned release notes live in [docs/releases/README.md](docs/releases/README.md), and the published release stream is tracked in [CHANGELOG.md](CHANGELOG.md).

## Current Focus

Viaduct is built for operators who need:
- mixed-estate discovery and inventory normalization
- dependency-aware migration assessment
- readiness and planning discipline before cutover work starts
- controlled, reviewable operator workflows with exported evidence

The default first-run experience is the WebUI-first workspace flow: start Viaduct, create a workspace, discover, inspect, simulate, save a plan, and export a report. The local lab remains the fastest path from fresh clone to a working dashboard and API.

## Why Viaduct

Many teams do not need more abstract migration talk. They need to know what exists, what depends on what, what should move first, and what evidence is good enough to approve a pilot. Viaduct is aimed at that planning and handoff gap.

It is strongest when operators need:
- one normalized inventory across mixed platforms
- dependency context before sequencing migration work
- a persisted assessment record instead of disconnected notes and screenshots
- a CLI, API, and dashboard that reflect the same state

## What You Can Evaluate Today

- Discovery: normalize inventory from VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup systems into the shared model.
- Dependency mapping: build workload, network, storage, and backup graph context before execution.
- Pilot workspace workflow: persist source connections, snapshots, graph outputs, target assumptions, readiness results, saved plans, approvals, notes, and exported reports in one operator record.
- Migration planning: use declarative specs, preflight checks, saved dry-run plans, execution windows, approval requirements, checkpoints, resume support, and rollback state.
- Lifecycle analysis: review drift, policy, cost, remediation guidance, and backup portability inputs in the same control plane.
- Multi-tenancy and extensibility: use tenant-scoped APIs, service accounts, PostgreSQL-backed state, and the community plugin host.

## Screenshots

These are current seeded-product captures from the packaged operator shell. They reflect the `viaduct start` path, the runtime bootstrap flow, and the current workspace-first dashboard.

<p align="center">
  <img src="docs/operations/demo/screenshots/auth-bootstrap.png" alt="Viaduct dashboard runtime authentication bootstrap" width="48%" />
  <img src="docs/operations/demo/screenshots/pilot-workspace.png" alt="Viaduct pilot workspace overview with operator workflow progression" width="48%" />
</p>
<p align="center">
  <img src="docs/operations/demo/screenshots/inventory-assessment.png" alt="Viaduct inventory assessment with selected workload detail" width="48%" />
  <img src="docs/operations/demo/screenshots/migration-ops.png" alt="Viaduct migration operations workspace with plan and execution posture" width="48%" />
</p>

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

## Canonical Install

Viaduct `v3.0.0` treats the signed OCI image as the canonical production artifact.

```bash
docker pull ghcr.io/eblackrps/viaduct:3.0.0
cosign verify ghcr.io/eblackrps/viaduct:3.0.0 \
  --certificate-identity \
  'https://github.com/eblackrps/viaduct/.github/workflows/image.yml@refs/tags/v3.0.0' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

Detailed container guidance lives in [docs/operations/docker.md](docs/operations/docker.md). Native bundles remain available on GitHub Releases as an alternative path for environments that cannot run containers.

## First Run From Source

The cleanest path is the local lab in [examples/lab](examples/lab).

```bash
make build
make web-build
./bin/viaduct version
./bin/viaduct start
```

On a fresh source checkout, `viaduct start`:
- creates `~/.viaduct/config.yaml` automatically when it is missing
- points that config at the shipped `examples/lab/kvm` fixtures
- serves the built dashboard and API together at [http://127.0.0.1:8080](http://127.0.0.1:8080)
- opens the WebUI automatically on interactive local runs when practical

For the default local lab path, the bootstrap screen offers `Use local operator session` on direct `127.0.0.1` browser requests, so no pasted browser key is required. Tenant keys and service-account keys remain supported for multi-tenant, packaged, and pilot environments.

Use these companion commands when you need them:

```bash
./bin/viaduct status --runtime
./bin/viaduct doctor
./bin/viaduct stop
```

The same runtime also publishes live operator API docs at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs), backed by the checked-in contract in [docs/reference/openapi.yaml](docs/reference/openapi.yaml).

`viaduct serve-api` remains the lower-level API command for container, service, and intentionally headless deployments. It still serves the built dashboard automatically when assets are present in `web/dist`, a packaged `web/` directory, or an installed `share/viaduct/web` layout. It now binds to `127.0.0.1` by default and refuses unauthenticated non-loopback listeners unless you configure an admin, tenant, or service-account credential or pass the explicit dangerous override. If you prefer the Vite development server while changing frontend code, that flow still lives in [web/README.md](web/README.md).

If you serve the dashboard from a different browser origin, configure `VIADUCT_ALLOWED_ORIGINS` on the API so tenant-protected routes can be reached safely. The default same-origin local path on `http://127.0.0.1:8080` does not need that override.

Tenant and service-account API keys are persisted as non-recoverable hashes. Viaduct only reveals a raw key at tenant creation time or during an explicit service-account rotate flow.

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

`make release-gate` is the canonical local release-owner verification path. It runs backend build, vet, lint, race coverage, certification fixtures, soak coverage, plugin and OpenAPI contract checks, CLI smoke coverage, dashboard lint/format/unit/build verification, coverage enforcement, and the cross-platform package matrix in one command. CI adds Playwright end-to-end coverage plus `gosec` and `trivy` on top of the same source-controlled release path.

Other high-signal commands:

```bash
make build
make certification-test
make soak-test
make contract-check
make package-release-matrix
```

`make package-release-matrix` produces the secondary native-bundle matrix that release tags attach beneath the canonical OCI image publication path: `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64` archives plus `dist/SHA256SUMS`.

## Documentation

- Overview and status: [README.md](README.md)
- Install: [INSTALL.md](INSTALL.md)
- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Upgrade: [UPGRADE.md](UPGRADE.md)
- Release process: [RELEASE.md](RELEASE.md)
- Release notes: [docs/releases/README.md](docs/releases/README.md)
- Support: [SUPPORT.md](SUPPORT.md)
- Security: [SECURITY.md](SECURITY.md)
- Documentation index: [docs/README.md](docs/README.md)
- API contract: [docs/reference/openapi.yaml](docs/reference/openapi.yaml)
- Live API docs: `/api/v1/docs` on any running Viaduct server
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
