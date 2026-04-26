# Support Matrix

This matrix reflects what is implemented and what is currently validated in the repository. It is intentionally conservative.

The platform names and validation scope are mirrored in `docs/reference/support-matrix.json`; `make support-matrix-check` verifies that README and website connector claims stay aligned with this table.

## Toolchain
| Component | Expected Version | Notes |
| --- | --- | --- |
| Go | 1.26.0+ | CI and release docs align to the `go.mod` version; the release image currently builds with Go 1.26.2. |
| Node.js | 20.19+ locally / 20.20.x in CI-release | Dashboard development supports Node.js 20.19+, while CI and release packaging pin Node.js 20.20.x. |
| `make` | Optional but recommended | Windows users can run the underlying commands directly if `make` is unavailable. |
| `qemu-img` | Optional | Needed for live disk conversion outside mocked or fixture-backed tests. |

## Runtime Validation
| Area | Validation Scope | Notes |
| --- | --- | --- |
| Release owner gate | CI build job and local release-gate coverage | `make release-gate` runs `go mod tidy`, backend build, vet, `golangci-lint`, race tests, certification fixtures, soak coverage, plugin and OpenAPI contract checks, CLI smoke checks, dashboard lint/format/unit/build verification, the coverage gate, and the package matrix. |
| Dashboard lint and unit tests | CI and local | `make release-gate`, `npm run lint`, `npm run format`, and `npm run test` enforce frontend quality and auth/workspace regressions. |
| Dashboard end-to-end tests | CI and local opt-in | `npm run e2e` exercises the Get started sign-in flow, dashboard, inventory, and migration workflow paths. It stays outside `make release-gate` because it depends on Playwright browser setup, but CI still enforces it on every pull request. |
| Security scans | CI and local opt-in | `gosec ./...` and `trivy fs --severity HIGH,CRITICAL .` run in CI; release owners can run the same commands locally when scanners are available. |
| Connector certification | Fixture-backed local and CI coverage | `make certification-test` validates KVM and Proxmox normalization against stable fixtures. |
| Migration soak | Tagged local and CI coverage | `make soak-test` exercises large-wave orchestration behavior without requiring external hypervisors. |
| API contract | Local and CI contract check | `make contract-check` verifies the published OpenAPI reference plus `/api/v1/docs/swagger.json` coverage for documented routes. |
| Plugin compatibility | Local and CI manifest validation | `make plugin-check` validates manifest protocol and host-version compatibility markers. |
| Release packaging | Local and CI packaging checks | `.github/workflows/image.yml` publishes the primary signed multi-arch OCI image, while `make package-release-matrix` and the tag workflow attach native bundles for `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64` plus `dist/SHA256SUMS` as an alternative path. |
| Dashboard build | CI and local | `npm run build` is part of the release gate. |

## Connectors
| Connector | Implemented | Unit tested | Fixture tested | Live lab tested | Production pilot tested | Known limitations |
| --- | --- | --- | --- | --- | --- | --- |
| VMware | Discovery and mapping helpers | Yes | Yes | Not claimed | Not claimed | Runtime behavior depends on vCenter API access and fixture coverage is not a substitute for environment-specific pilot testing. |
| Proxmox | REST discovery and mapping helpers | Yes | Yes, including certification fixtures | Not claimed | Not claimed | Discovery is better covered than connector-specific migration execution. |
| Hyper-V | WinRM / PowerShell discovery | Yes | Yes | Not claimed | Not claimed | Requires operator-provided WinRM/PowerShell access in real environments. |
| KVM | XML fixture discovery and libvirt build-tag implementation | Yes | Yes, including certification fixtures | Local fixture lab only | Not claimed | Default builds use the portable XML fallback; live libvirt usage requires the `libvirt` build tag and host libraries. |
| Nutanix | Prism Central v3 discovery | Yes | Yes | Not claimed | Not claimed | Fixture-backed validation does not prove every Prism deployment shape. |
| Veeam | Backup discovery, restore-point correlation, and portability planning inputs | Yes | Yes | Not claimed | Not claimed | Backup correlation is name-based and case-insensitive because Veeam commonly exposes protected objects by display name. |
| Community plugins | gRPC plugin host, manifest compatibility, and sample plugin | Yes | Sample plugin and manifest checks | Not claimed | Not claimed | Plugins must report a non-empty platform ID and pass host/plugin compatibility checks before use. |

## Migration And Lifecycle Features
| Area | Implemented | Repository validation | Production claim |
| --- | --- | --- | --- |
| Declarative specs and workload selection | Yes | Unit and integration tests | Available for evaluation and pilot planning. |
| Preflight and dry-run planning | Yes | Unit and integration tests | Recommended before any supervised migration attempt. |
| Cold and warm orchestration primitives | Yes | Unit, integration, and soak tests with mocked or fixture-backed dependencies | Not claimed as production-proven across every connector pair. |
| Resume, checkpoints, and rollback state | Yes | Unit, integration, and soak tests | Treat as supervised pilot functionality until validated in the target environment. |
| Lifecycle policy, drift, cost, and remediation guidance | Yes | Unit and integration tests | Decision support; recommendations should be reviewed against local policy before action. |
| Dashboard workspace and reports | Yes | Unit, fixture E2E, and runtime smoke tests | Suitable for evaluation and pilot evidence workflows. |

## Operational Notes
- Persistent deployments should use PostgreSQL rather than the in-memory store.
- Multi-tenant deployments should use service account keys for normal operator access and reserve tenant keys for setup or tenant-admin work.
- Any higher-risk migration usage should be exercised in a lab or supervised pilot environment first, especially for connector-specific runtime actions beyond discovery.
- The Vite dashboard dev server is for local development only and should not be exposed as a public or shared internet-facing surface.
