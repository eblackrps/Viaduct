# Viaduct
Hypervisor-agnostic workload migration and lifecycle management platform.
## Project Identity
- Repo: github.com/eblackrps/viaduct
- License: Apache 2.0
- Language: Go 1.24+ (core engine, CLI), Python 3.12+ (SDK), TypeScript/React (dashboard)
- Author: Eric Black (eblackrps)
## Architecture
Four layers, each a separate Go package under internal/:
1. Discovery Engine (internal/discovery/) - Implemented in Phase 1. Connects to hypervisor APIs, runs discovery concurrently, aggregates cross-platform results, and provides inventory diffing utilities.
2. Dependency Mapper (internal/deps/) - Implemented in Phase 2 for dashboard graphing. Builds a normalized node/edge graph from inventory, storage, network, and backup metadata.
3. Migration Orchestrator (internal/migrate/) - Implemented through Phase 3. Provides declarative spec parsing, workload matching, pre-flight validation, cold and warm migration orchestration, replication tracking, cutover coordination, boot verification, disk conversion, network remapping, and rollback state management.
4. Lifecycle Manager (internal/lifecycle/) - Implemented in Phase 3. Handles drift detection, cost modeling, policy evaluation/simulation, and backup portability planning inputs.
## Connector Plugin Model
Each connector lives in internal/connectors/<platform>/ and implements the Connector interface in internal/connectors/connector.go.
Connectors: vmware/ (govmomi), proxmox/ (REST), hyperv/ (WinRM/PowerShell), kvm/ (libvirt), nutanix/ (Prism v3), veeam/ (VBR REST).
- Connector implementations now follow a common pattern: a connector entry file (`<platform>.go`), mapper helpers (`mapper.go`), infrastructure discovery helpers (`infra.go` where needed), and focused tests per behavior area.
- REST connectors should keep HTTP transport/auth logic in a dedicated client file (`client.go`) so mapping logic stays deterministic and easy to test.
- Connector test fixtures live in `internal/connectors/<platform>/testdata/` and should store API-shaped payloads rather than ad hoc snippets when possible.
- Backup-oriented connectors such as `internal/connectors/veeam/` may expose domain-specific discovery methods instead of the core VM discovery interface when they enrich the universal inventory rather than replace it.
- The KVM connector supports a real libvirt-backed implementation behind the `libvirt` build tag; the default build keeps an XML-backed fallback so the repo remains portable on machines without libvirt development tooling.
- Community plugins are loaded through `internal/connectors/plugin/`, which exposes a gRPC-based plugin host and a sample plugin under `examples/plugin-example/`.
## Code Standards
- Run golangci-lint before committing. Config in .golangci.yml.
- All exported functions require doc comments.
- Wrap errors: fmt.Errorf("context: %w", err). Never swallow errors.
- No panic() in library code.
- context.Context as first param for any I/O function.
- Table-driven tests. Name pattern: TestFunction_Scenario_Expected.
- No global mutable state. Pass dependencies explicitly.
- Prefer small mapping helpers for platform-specific type translation so schema changes stay isolated to the connector package.
- Keep discovery results normalized in `internal/models/`; CLI rendering and storage layers should not invent parallel schemas.
## CLI (cobra)
Root command: viaduct. Subcommands: discover, plan, migrate, status, rollback, version.
Each subcommand in its own file under cmd/viaduct/.
## Build Commands
make build - Build binary to bin/viaduct
make test - go test ./... -v -race
make lint - golangci-lint run ./...
make all - lint + test + build
make certification-test - Run fixture-backed connector certification checks
make soak-test - Run tagged migration soak coverage
make release-gate - Full release verification: tidy, build, vet, lint, race tests, soak coverage, binary checks, web build, coverage, and packaging
make package-release-matrix - Build the CLI, build the dashboard, and create release bundles in dist/
## Commit Style
Conventional commits: type(scope): description
Types: feat, fix, docs, test, refactor, ci, chore
Scopes: cli, discovery, vmware, proxmox, hyperv, kvm, nutanix, migrate, deps, lifecycle, store, dashboard, proto, ci
## Constraints
- Never hardcode credentials. Use env vars or config files in .gitignore.
- Never commit .env, secrets, or API keys.
- All discovery-phase API calls are read-only.
- Universal schema in internal/models/ is the single source of truth.
- YAML migration specs are declarative, never imperative.
## Migration Spec
- Migration specs live in YAML and are parsed by `internal/migrate/spec.go`.
- Source and target sections define address, platform, and credential references.
- Workload selectors support glob patterns, regex via `regex:<expr>`, tag filters, folder matching, power-state filters, and exclude patterns.
- Overrides belong with selectors and carry target host, target storage, network mappings, storage mappings, and optional dependency hints for wave planning.
- Phase 4 execution controls live under `options.window`, `options.approval`, and `options.waves`. Keep them declarative so plans, pre-flight checks, API flows, and resume logic all consume the same spec.
- Example specs are kept in `configs/example-migration.yaml` and `configs/example-migration-minimal.yaml`.
## Veeam Integration
- `internal/connectors/veeam/` discovers jobs, restore points, and repositories through the VBR REST API.
- Backup correlation is name-based and case-insensitive because Veeam commonly exposes protected objects by display name rather than inventory UUID.
- Backup models live in `internal/models/backup.go` and are consumed by dependency graph and future lifecycle features.
## Dashboard
- The Go API server lives in `internal/api/server.go` and exposes inventory, snapshot, migration, pre-flight, health, graph, cost, policy, drift, remediation, simulation, audit, reporting, metrics, and tenant administration routes over `net/http`.
- The React dashboard lives in `web/` and is built with Vite, TypeScript, Tailwind CSS, Recharts, and D3.
- Core components are split by function: `InventoryTable`, `PlatformSummary`, `MigrationWizard`, `MigrationProgress`, `DependencyGraph`, `CostComparison`, `PolicyDashboard`, `DriftTimeline`, and `MigrationHistory`.
- Use the hidden CLI command `viaduct serve-api` for local API serving, and `make serve` to start the dashboard dev workflow from `web/`.
## Testing Patterns
- Use mock connectors in `internal/discovery/` tests to validate orchestration and aggregation behavior without external systems.
- Use `httptest.Server` for REST API connector tests such as Proxmox so authentication, routing, and payload parsing are exercised end to end.
- Store reusable API fixtures under package-local `testdata/` directories and keep them close to the connector they validate.
- Keep race-safe tests enabled by default. If future tests need real hypervisors, gate them behind an explicit build tag rather than weakening the default suite.
- Migration tests should swap the disk converter through `Orchestrator.SetDiskConverter` so end-to-end flows remain hermetic and do not require `qemu-img`.
- REST API dashboards and backup connectors should prefer realistic fixtures and graph-level assertions over brittle snapshot string comparisons.
## Phase 2 Patterns
- Disk conversion is centralized in `internal/migrate/diskconv.go` and shells out to `qemu-img`; integration coverage must stay behind the `integration` build tag.
- Pre-flight checks are composable, report pass/warn/fail status, and should continue collecting actionable results instead of aborting on the first issue.
- Rollback persists recovery metadata to the shared store and treats missing artifacts as idempotent cleanup rather than hard failure.
- API connectors should isolate HTTP transport in `client.go`, while dashboard graph tests should validate node/edge semantics rather than rendering internals.
## Phase 3 Patterns
- Warm migration state should flow through `internal/migrate/replication.go` and `internal/migrate/cutover.go`; replication checkpoints belong in the shared store so retries and cutovers remain idempotent.
- Lifecycle rules, cost profiles, and drift baselines should stay data-driven through YAML configs and normalized inventory snapshots rather than embedding policy logic into the CLI or dashboard.
- Tenant-aware code must use the helper functions in `internal/store/store.go` to derive the active tenant from context and keep persistence isolated per tenant.
- Plugin integrations should validate health and platform identity on load before exposing a connector instance to the rest of the system.
## Post-1.0 Guidance
- Treat tenant isolation as a release blocker. Migration IDs, recovery points, and inventory reads must remain scoped by tenant even when identifiers collide across tenants.
- `internal/api/server.go` should expose only the latest snapshot per source/platform when building current inventory views; merging every historical snapshot will duplicate VMs and distort lifecycle outputs.
- Warm migration resume logic must persist state even when the triggering context is canceled. If cancellation is the failure mode, persist through a tenant-scoped background context before returning.
- Rollback is only successful when cleanup and source restoration both complete. If rollback returns actionable errors, the persisted migration state must remain failed rather than rolled back.
- Connector-provided boot verification must honor caller timeouts. Never let a platform-specific verifier run without a bounded context.
- Plugin shutdown paths should tolerate already-dead plugin processes, but plugin load/connect/discover flows must reject empty platform IDs, nil results, and explicitly unsuccessful responses.
- Backup portability execution should surface partial-create or verification failures as errors while still returning enough result data for cleanup.
- Keep release gating reproducible. CI and local release verification should run through `make release-gate` so backend, CLI, frontend, and coverage checks stay aligned.
## Phase 4 Guidance
- Treat execution planning as persisted state, not transient UI decoration. Migration plans, checkpoints, approval state, and scheduling windows should stay in `internal/migrate/` so resume flows and API consumers see the same truth.
- Resume support must skip already-completed phases. Do not rerun export or conversion work if checkpoints show those phases completed successfully.
- Approval gating is not a failure state. Block execution cleanly, keep the migration in plan state, and surface actionable diagnostics rather than marking the run failed.
- Lifecycle remediation should combine policy and cost inputs before suggesting placement changes. If a cheaper target would violate enforce-level policy, suppress that recommendation.
- Policy waivers must be explicit and expiring. Never introduce permanent silent exceptions in policy evaluation paths.
- Tenant summary and reporting endpoints must remain tenant-scoped and derive their data through store helpers rather than ad hoc cross-tenant queries.
- The dashboard should surface runbooks, checkpoints, and remediation guidance with the same API payloads used by automation. Avoid frontend-only inference for execution or policy state.
- `make release-gate` is the canonical Phase 4 release check and now includes a coverage threshold gate through `scripts/coverage_gate.go`.
## Release Era Guidance
- Keep `go.mod`, `.golangci.yml`, CI, and public docs aligned on the supported Go version. Release drift between those files is a release blocker.
- `make package-release-matrix` is the canonical packaging path. Release bundles should include the CLI binary, built web assets, install scripts, docs, configs, examples, deployment references, a manifest, and checksums.
- Public docs are part of the product surface now. If an API route, CLI flag, config field, or operator workflow changes, update the relevant docs in the same change.
- The top-level docs (`README.md`, `INSTALL.md`, `QUICKSTART.md`, `UPGRADE.md`, `RELEASE.md`, `SUPPORT.md`, `SECURITY.md`, `CHANGELOG.md`) are the public entrypoints. Keep them aligned with the detailed guides under `docs/`.
- `docs/roadmaps/` is a historical archive. Completed phases should be described in past tense and not framed as the active state of the project.
- Contributor-facing directories such as `docs/`, `configs/`, `examples/`, `api/`, `tests/`, and `web/` should have a local `README` when their purpose would otherwise be unclear to a new contributor.
- Example configs and lab assets must stay runnable. Prefer real, parseable examples over pseudo-config snippets.
- The local KVM lab under `examples/lab/` is the default first-run and demo path; keep it fast, offline-friendly, and stable.
- Reference deployment assets under `examples/deploy/` should stay simple, parseable, and obviously scoped to lab and pilot use rather than implied production hardening.
- Keep operator guidance honest about maturity. If a workflow is backend-only or automation-oriented rather than a first-class CLI command, document it that way instead of implying a polished surface that does not exist.
- Release verification should exercise packaging as well as builds and tests. If packaging breaks, the release is not ready even if `go build` passes.
- Plugin executable launches now expect a `plugin.json` sidecar manifest with name, platform, version, and protocol version metadata. Keep the manifest aligned with the plugin’s reported platform.
- Tenant-scoped audit, reporting, and request-correlation behavior are part of the operator surface. Changes to those routes should update both the API docs and troubleshooting guidance.
- Treat install, upgrade, and rollback documentation as operational code. Broken runbooks are production bugs.
## Current State
- Discovery is implemented for VMware, Proxmox, Hyper-V, KVM, and Nutanix, with Veeam available for backup and restore-point enrichment plus portability planning.
- `internal/migrate/` includes declarative spec parsing, workload matching, pre-flight checks, execution windows, approval gates, wave planning, resumable checkpoints, disk conversion, cold and warm migration orchestration, replication progress tracking, boot verification, cutover coordination, and rollback support.
- `internal/lifecycle/` provides cost modeling, policy evaluation/simulation, waiver-aware remediation guidance, and drift detection backed by sample cost profiles and policy definitions under `configs/`.
- `internal/store/` persists discovery snapshots, migration state, recovery points, and tenant metadata in both in-memory and PostgreSQL backends.
- The Go API server and React dashboard are both present, and `web/` builds into production static assets with inventory, migration, history, dependency-graph, tenant summary, runbook, and lifecycle remediation views.
- Integration coverage includes discovery, cold migration, scheduled-window pre-flight gating, migration resume after interruption, warm migration resume, cutover rollback on boot failure, tenant isolation under concurrent access, lifecycle recommendation/simulation flows, plugin crash handling, and backup portability workflows.
