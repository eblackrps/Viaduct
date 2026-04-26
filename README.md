# Viaduct
> Open-source software for multi-platform inventory, dependency review, migration planning, readiness checks, and reports.

[![CI](https://github.com/eblackrps/Viaduct/actions/workflows/ci.yml/badge.svg)](https://github.com/eblackrps/Viaduct/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/eblackrps/Viaduct)](https://github.com/eblackrps/Viaduct/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/eblackrps/Viaduct?display_name=tag)](https://github.com/eblackrps/Viaduct/releases)

Viaduct helps teams inspect multi-platform environments, map dependencies, build migration plans, and keep reviewable records before a pilot. The repository combines a Go backend, REST API, CLI, React dashboard, and standalone public site around the same saved inventory, assessment, planning, and reporting data.

Versioned release notes live in [docs/releases/README.md](docs/releases/README.md), the current install reference lives in [docs/releases/current.md](docs/releases/current.md), and the published release stream is tracked in [CHANGELOG.md](CHANGELOG.md).

## Current Focus

Viaduct is built for operators who need:
- multi-platform inventory collection
- review dependencies before planning
- readiness checks and planning steps before cutover work starts
- save plans and export reports

The default first-run experience is the dashboard assessment workflow: start Viaduct, create an assessment, discover, inspect, simulate, save a plan, and export a report. The local lab remains the fastest path from fresh clone to a working dashboard and API.

## Why Viaduct

Many teams do not need more abstract migration talk. They need to know what exists, what depends on what, what should move first, and what review material is good enough to approve a pilot. Viaduct is aimed at that planning and handoff gap.

It is strongest when operators need:
- one normalized inventory across mixed platforms
- dependency context before sequencing migration work
- a saved assessment instead of disconnected notes and screenshots
- a CLI, API, and dashboard that reflect the same state

## What You Can Evaluate Today

- Discovery: normalize inventory from VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup systems into the shared model.
- Dependency mapping: build workload, network, storage, and backup graph context before execution.
- Assessment workflow: save source connections, snapshots, graph outputs, target assumptions, readiness results, plans, approvals, notes, and exported reports together.
- Migration planning: use declarative specs, preflight checks, saved dry-run plans, execution windows, approval requirements, checkpoints, resume support, and rollback state.
- Lifecycle analysis: review drift, policy, cost, remediation guidance, and backup portability inputs against the same stored data.
- Diagnostics and observability: validate local runtime readiness with `viaduct doctor` and `viaduct status --runtime`, then optionally prove backend trace flow with the bundled Grafana + Tempo path.
- Multi-tenancy and extensibility: use tenant-scoped APIs, service accounts, PostgreSQL-backed state, and the community plugin host.

## Screenshots

These are current seeded-product captures from the packaged dashboard. They reflect the `viaduct start` path, the Get started sign-in screen, and the current assessment dashboard across inventory, graph, migration, and reporting pages.

<p align="center">
  <img src="docs/operations/demo/screenshots/auth-bootstrap.png" alt="Viaduct dashboard Get started screen with local and key-based session options" width="32%" />
  <img src="docs/operations/demo/screenshots/pilot-workspace.png" alt="Viaduct assessment overview with workflow progress" width="32%" />
  <img src="docs/operations/demo/screenshots/inventory-assessment.png" alt="Viaduct inventory assessment with selected workload detail" width="32%" />
</p>
<p align="center">
  <img src="docs/operations/demo/screenshots/dependency-graph.png" alt="Viaduct dependency graph view showing workload relationships" width="32%" />
  <img src="docs/operations/demo/screenshots/migration-ops.png" alt="Viaduct migration operations page with plan and execution status" width="32%" />
  <img src="docs/operations/demo/screenshots/reports-history.png" alt="Viaduct reports and history view with exported review reports" width="32%" />
</p>

## Platform Coverage

Detailed validation status, including fixture-backed versus live-lab claims, is maintained in [docs/reference/support-matrix.md](docs/reference/support-matrix.md).

| Platform / Integration | Status | Notes |
| --- | --- | --- |
| VMware vSphere | Implemented | vCenter discovery with VM and infrastructure metadata. |
| Proxmox VE | Implemented | REST-based discovery and a common target in current pilot examples. |
| Microsoft Hyper-V | Implemented | WinRM and PowerShell-backed inventory collection. |
| KVM / libvirt | Implemented | XML-backed fallback works out of the box; live libvirt support is available behind the `libvirt` build tag. |
| Nutanix AHV | Implemented | Prism Central v3 inventory collection. |
| Veeam Backup & Replication | Implemented | Backup discovery, restore-point correlation, and portability planning support. |
| Community plugins | Implemented | gRPC plugin host with manifest and compatibility checks. |

## Primary Docker Install

Viaduct v3.2.1 uses the signed GHCR OCI image as the primary packaged artifact.

```bash
docker pull ghcr.io/eblackrps/viaduct:3.2.1
cosign verify ghcr.io/eblackrps/viaduct:3.2.1 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.2.1' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

The primary signed registry is `ghcr.io/eblackrps/viaduct`. The Docker Hub mirror is `docker.io/emb079/viaduct:3.2.1` when repository Docker Hub secrets are configured.

GitHub Actions is configured to mirror release tags plus `main` branch `:edge` and `:sha-*` image tags to Docker Hub whenever `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured as Actions secrets for this repo or as inherited organization secrets. Detailed container guidance lives in [docs/operations/docker.md](docs/operations/docker.md). The Compose and Helm samples in [deploy/docker-compose.prod.yml](deploy/docker-compose.prod.yml) and [deploy/helm/viaduct](deploy/helm/viaduct) use PostgreSQL-backed state for persistent deployments. Native bundles remain available on GitHub Releases as an alternative path for environments that cannot run containers.

For the production Compose sample, create `config/config.yaml` from `configs/config.example.yaml` before starting the service; the container mounts that directory at `/etc/viaduct:ro`.

## Source Build And Local Lab

The cleanest contributor and offline evaluation path is still the local lab in [examples/lab](examples/lab). Docker is the primary packaged install path, but the local assessment workflow remains fastest from a fresh clone.

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
- opens the dashboard automatically on interactive local runs when practical

For the default local lab path, the Get started screen offers `Start local session` on direct `127.0.0.1` browser requests, so no pasted browser key is required. If you need a shared or packaged setup, open `Use a key instead`; service account keys are the normal path, and tenant keys remain available under the advanced option.

Use these companion commands when you need them:

```bash
./bin/viaduct status --runtime
./bin/viaduct doctor
./bin/viaduct stop
```

The same runtime also publishes live API docs at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs), backed by the checked-in contract in [docs/reference/openapi.yaml](docs/reference/openapi.yaml).

`viaduct serve-api` remains the lower-level API command for container, service, and intentionally headless deployments. It still serves the built dashboard automatically when assets are present in `web/dist`, a packaged `web/` directory, or an installed `share/viaduct/web` layout. It now binds to `127.0.0.1` by default and refuses unauthenticated non-loopback listeners unless you configure an admin, tenant, or service account credential or pass the explicit dangerous override. If you prefer the Vite development server while changing frontend code, that flow still lives in [web/README.md](web/README.md).

If you serve the dashboard from a different browser origin, configure `VIADUCT_ALLOWED_ORIGINS` on the API so tenant-protected routes can be reached safely. The default same-origin local path on `http://127.0.0.1:8080` does not need that override.

Tenant and service account API keys are persisted as non-recoverable hashes. Viaduct only reveals a raw key at tenant creation time or during an explicit service account rotate flow.

Use these entrypoints next:
- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Detailed quickstart: [docs/getting-started/quickstart.md](docs/getting-started/quickstart.md)
- Assessment guide: [docs/operations/pilot-workspace-flow.md](docs/operations/pilot-workspace-flow.md)

## Repository Surfaces

- `cmd/viaduct/`: CLI entrypoints.
- `internal/`: discovery, dependency mapping, migration orchestration, lifecycle analysis, API server, and state store packages.
- `web/`: React dashboard for assessment workflows.
- `site/`: separate static public site for GitHub Pages.
- `examples/`: lab, deployment, and plugin evaluation assets.
- `docs/`: deeper reference, operations, and product-scope documentation.

## Verification

`make release-gate` is the release check command. It runs backend build, vet, lint, race coverage, certification fixtures, soak coverage, plugin and OpenAPI contract checks, release-surface consistency checks, CLI smoke coverage, dashboard lint/format/unit/build verification, coverage enforcement, and the cross-platform package matrix in one command. CI adds Playwright end-to-end coverage, a Docker-backed observability smoke for Grafana + Tempo trace ingestion, CodeQL, `gosec`, and `trivy` on top of the same source-controlled release path.

Other high-signal commands:

```bash
make build
make observability-up
make web-e2e-setup
make pilot-smoke
make observability-validate
make certification-test
make soak-test
make contract-check
make release-surface-check
make package-release-matrix
```

`make package-release-matrix` produces the secondary native-bundle matrix that release tags attach beneath the primary OCI image publication path: `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64` archives plus `dist/SHA256SUMS`.

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
- Backend observability: [docs/operations/observability.md](docs/operations/observability.md)

## Contributing

Contributions are welcome. Keep docs, examples, and public-facing assets aligned with visible behavior. See [CONTRIBUTING.md](CONTRIBUTING.md) for workflow and verification expectations.

## Support And Security

- Usage, evaluation, and troubleshooting: [SUPPORT.md](SUPPORT.md)
- Security reporting: [SECURITY.md](SECURITY.md)
- Project change history: [CHANGELOG.md](CHANGELOG.md)
- Community expectations: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## License

Viaduct is licensed under the [Apache License 2.0](LICENSE).
