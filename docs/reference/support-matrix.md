# Support Matrix

This matrix reflects what is implemented and what is currently validated in the repository. It is intentionally conservative.

## Toolchain
| Component | Expected Version | Notes |
| --- | --- | --- |
| Go | 1.25.9+ | CI and release docs align to the `go.mod` version. |
| Node.js | 20.19+ locally / 20.20.x in CI-release | Dashboard development supports Node.js 20.19+, while CI and release packaging pin Node.js 20.20.x. |
| `make` | Optional but recommended | Windows users can run the underlying commands directly if `make` is unavailable. |
| `qemu-img` | Optional | Needed for live disk conversion outside mocked or fixture-backed tests. |

## Runtime Validation
| Area | Validation Scope | Notes |
| --- | --- | --- |
| Release owner gate | CI build job and local release-gate coverage | `make release-gate` runs `go mod tidy`, backend build, vet, `golangci-lint`, race tests, certification fixtures, soak coverage, plugin and OpenAPI contract checks, CLI smoke checks, dashboard lint/format/unit/build verification, the coverage gate, and the package matrix. |
| Dashboard lint and unit tests | CI and local | `make release-gate`, `npm run lint`, `npm run format`, and `npm run test` enforce frontend quality and auth/workspace regressions. |
| Dashboard end-to-end tests | CI and local opt-in | `npm run e2e` exercises the login bootstrap, operator console, inventory, and migration workflow paths. It stays outside `make release-gate` because it depends on Playwright browser setup, but CI still enforces it on every pull request. |
| Security scans | CI and local opt-in | `gosec ./...` and `trivy fs --severity HIGH,CRITICAL .` run in CI; release owners can run the same commands locally when scanners are available. |
| Connector certification | Fixture-backed local and CI coverage | `make certification-test` validates KVM and Proxmox normalization against stable fixtures. |
| Migration soak | Tagged local and CI coverage | `make soak-test` exercises large-wave orchestration behavior without requiring external hypervisors. |
| API contract | Local and CI contract check | `make contract-check` verifies the published OpenAPI reference plus `/api/v1/docs/swagger.json` coverage for documented routes. |
| Plugin compatibility | Local and CI manifest validation | `make plugin-check` validates manifest protocol and host-version compatibility markers. |
| Release packaging | Local and CI packaging checks | `.github/workflows/image.yml` publishes the canonical signed multi-arch OCI image, while `make package-release-matrix` and the tag workflow attach native bundles for `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64` plus `dist/SHA256SUMS` as an alternative path. |
| Dashboard build | CI and local | `npm run build` is part of the release gate. |

## Connectors
| Connector | Status | Repository validation | Notes |
| --- | --- | --- | --- |
| VMware | Discovery implemented | Package-level tests and fixtures | vCenter discovery with VM and infrastructure metadata. |
| Proxmox | Discovery implemented | Package-level tests plus fixture-backed certification | REST-based inventory for VMs, containers, networks, storage, and nodes. |
| Hyper-V | Discovery implemented | Package-level tests and fixtures | WinRM / PowerShell inventory collection. |
| KVM | Discovery implemented | Package-level tests plus fixture-backed certification | XML-backed fallback plus libvirt build-tag implementation. |
| Nutanix | Discovery implemented | Package-level tests and fixtures | Prism Central v3 inventory collection. |
| Veeam | Backup discovery and portability implemented | Package-level tests and fixtures | Used for backup correlation and portability planning. |
| Community plugins | Supported | Host and manifest tests plus `make plugin-check` | gRPC plugin host with validation for health, platform ID, discovery results, and manifest compatibility. |

## Operational Notes
- Persistent deployments should use PostgreSQL rather than the in-memory store.
- Multi-tenant deployments should use service-account keys for normal operator access and reserve tenant keys for bootstrap or tenant-admin work.
- Any higher-risk migration usage should be exercised in a lab or supervised pilot environment first, especially for connector-specific runtime actions beyond discovery.
- The Vite dashboard dev server is for local development only and should not be exposed as a public or shared internet-facing surface.
