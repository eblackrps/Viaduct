# Support Matrix

This matrix reflects what is implemented and what is currently validated in the repository. It is intentionally conservative.

## Toolchain
| Component | Expected Version | Notes |
| --- | --- | --- |
| Go | 1.24+ | CI and release docs align to the `go.mod` version. |
| Node.js | 20.19+ | Required for dashboard development and local web builds. |
| `make` | Optional but recommended | Windows users can run the underlying commands directly if `make` is unavailable. |
| `qemu-img` | Optional | Needed for live disk conversion outside mocked or fixture-backed tests. |

## Runtime Validation
| Area | Validation Scope | Notes |
| --- | --- | --- |
| Backend build and tests | CI and local release-gate coverage | Includes build, vet, lint, race tests, coverage, and packaging. |
| Connector certification | Fixture-backed local and CI coverage | `make certification-test` validates KVM and Proxmox normalization against stable fixtures. |
| Migration soak | Tagged local and CI coverage | `make soak-test` exercises large-wave orchestration behavior without requiring external hypervisors. |
| API contract | Local and CI contract check | `make contract-check` verifies the published OpenAPI reference still covers the documented routes. |
| Plugin compatibility | Local and CI manifest validation | `make plugin-check` validates manifest protocol and host-version compatibility markers. |
| CLI packaging | Local and CI packaging checks | Release bundles are ZIP-based and include docs, configs, and web assets. |
| Dashboard build | CI and local | `npm run build` is part of the release gate. |

## Connectors
| Connector | Status | Notes |
| --- | --- | --- |
| VMware | Discovery implemented | vCenter discovery with VM and infrastructure metadata. |
| Proxmox | Discovery implemented | REST-based inventory for VMs, containers, networks, storage, and nodes. |
| Hyper-V | Discovery implemented | WinRM / PowerShell inventory collection. |
| KVM | Discovery implemented | XML-backed fallback plus libvirt build-tag implementation. |
| Nutanix | Discovery implemented | Prism Central v3 inventory collection. |
| Veeam | Backup discovery and portability implemented | Used for backup correlation and portability planning. |
| Community plugins | Supported | gRPC plugin host with validation for health, platform ID, and discovery results. |

## Operational Notes
- Persistent deployments should use PostgreSQL rather than the in-memory store.
- Multi-tenant deployments should use service-account keys for normal operator access and reserve tenant keys for bootstrap or tenant-admin work.
- Any higher-risk migration usage should be exercised in a lab or supervised pilot environment first, especially for connector-specific runtime actions beyond discovery.
- The Vite dashboard dev server is for local development only and should not be exposed as a public or shared internet-facing surface.
