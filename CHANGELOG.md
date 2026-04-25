# Changelog

All notable changes to Viaduct should be documented in this file.

This changelog tracks published releases and the major implementation milestones that shaped the current repository state.

## [Unreleased]

- No unreleased changes are documented yet.

## [3.2.1] - 2026-04-25

### Release, Website, And Deployment Surfaces

- aligned active release, install, Docker, Helm, website, and screenshot surfaces on the `v3.2.1` patch release
- documented PostgreSQL-backed state as the recommended persistent deployment path for Compose and Helm
- updated the public site install snippet and release links so they match the same GHCR, Docker Hub, and GitHub Release version

### Runtime Readiness And Safety

- added production-mode startup guards for memory-only state, missing authentication, missing lifecycle policies, and missing dashboard assets
- expanded `/readyz` so production deployments can see store, schema, policy, auth, connector circuit, and dashboard-asset status in one response
- added `VIADUCT_STATE_STORE_DSN` for container and Helm deployments that inject PostgreSQL connection details from environment or secrets

### Verification

- added public-site validation and published-image release acceptance checks for release owners
- strengthened release-surface drift detection so active docs and website files fail when stale semver image or release links remain

## [3.2.0] - 2026-04-24

### Trust, Diagnostics, And Observability

- added backend OpenTelemetry tracing across inbound HTTP requests, workspace jobs, discovery, migration orchestration, report/export generation, connector HTTP clients, and the PostgreSQL-backed persistence path that operators use during real assessments
- shipped a local Grafana + Tempo stack, source-controlled provisioning, and a validation helper so release owners can prove traces arrive from a real Viaduct runtime before tagging
- upgraded `viaduct doctor` and `viaduct status --runtime` to report config validity, store posture, auth posture, and concrete readiness degradation reasons instead of a simple reachable/not-reachable split

### Evaluator And Release Readiness

- strengthened the real runtime golden-path smoke so Viaduct now proves the full workspace-first evaluator flow from `viaduct start` through local session, workspace creation, discovery, graph, simulation, plan save, and report export
- added release-surface consistency checks, a current-release source of truth, CodeQL, and a Docker-backed observability smoke so version drift and trace regressions are caught before release
- refreshed the root docs, install guides, public site, Helm and compose samples, release references, and checked-in screenshots so the shipped operator story matches `v3.2.0`

## [3.1.1] - 2026-04-23

### Patch Release Follow-Up

- fixed a follow-up dashboard regression in `Content-Disposition` filename parsing so RFC 5987 `filename*=` parameters are matched literally and decoded correctly in the shipped client
- stabilized the new Get started auth-screen regression coverage by switching to unambiguous key-input queries and explicitly cleaning up test renders between runs so the web suite stays deterministic in CI
- aligned the current release-facing docs, install snippets, public site copy, Docker and Helm samples, and package metadata on `v3.1.1`

## [3.1.0] - 2026-04-22

### Get Started Experience

- replaced the runtime-credential-first auth screen with a simpler Get started flow that makes the local operator session the obvious primary path when it is available, keeps service account key sign-in as the default key-based option, and moves tenant key sign-in under advanced options
- rewrote the first-run copy, sign-in labels, and failed-auth messaging in plain English so operators can understand the local path, the key path, and the browser-session behavior at a glance
- refreshed the checked-in README and demo screenshots plus the supporting quickstart, lab, dashboard, and operator docs so the current release surface matches the shipped Get started experience

### Session Handling And Frontend Hardening

- stopped retaining browser-entered runtime keys client-side after successful session creation; the dashboard now clears the pasted key immediately, stores only the opaque session marker in browser storage, and relies on the server-backed session for subsequent requests
- fixed dashboard auth-session creation to keep sending `Content-Type: application/json` even when auth-header injection is skipped, preserved already-aborted external abort reasons before `fetch` begins, added RFC 5987 `filename*=` support for download names, and limited corrupt runtime-session cleanup to only the offending storage key
- expanded frontend regression coverage around already-aborted requests, auth-session header construction, corrupt storage fallback behavior, encoded download filenames, the new Get started layout, and raw-key clearing after sign-in

### Release Engineering And Tooling

- aligned the current release-facing version surfaces, Docker and Helm samples, package metadata, public site install snippets, and screenshot captions on `v3.1.0`
- hardened `.codex/setup.sh` so contributor bootstrap now fails fast below Go `1.25.9` and installs a configurable modern `golangci-lint` version that matches the repo-supported toolchain line

## [3.0.0] - 2026-04-21

### Release Model And Packaging

- shifted Viaduct to a Docker-canonical release model: the signed multi-arch GHCR image is now the primary release artifact, while native binary bundles remain published as an alternative path for operators who cannot run containers
- added a dedicated `image.yml` workflow that builds, signs, attests, scans, and publishes OCI images on merges to `main` and on release tags, while keeping native bundles as secondary tag assets
- documented the new v3 release cadence, deprecation policy, and roadmap so forward-looking platform work lives outside the changelog
- raised the supported source-build floor to Go `1.25.9+` so the Docker and release pipelines ship a Go toolchain line that clears the container vulnerability gate instead of baking known stdlib CVEs into the image

### Security And Runtime Hardening

- extended admin-key compatibility so `VIADUCT_ADMIN_KEY` accepts either plaintext or `sha256:<hex>` storage formats, always compares against the SHA-256 of the presented header in constant time, and warns operators to migrate off plaintext storage
- tightened malformed credential-hash decoding, forwarded-header rejection logging, durable audit retry handling, and session-revocation lifecycle coverage so strict local-runtime trust and shutdown behavior stay explicit
- hardened workspace enqueue accounting and release-workflow regression coverage around queue depth, artifact manifests, checksum reuse, and pinned signing identity

### Dashboard Polish And Accessibility

- standardized dashboard typography, radius, and neutral-color tokens so the operator console reads as one coherent application instead of page-local styling islands
- expanded focus-trap coverage, route-state accessibility checks, and visual-regression snapshots across the top operator routes, including mobile drawer restoration and keyboard-only navigation
- removed dead frontend exports, stale re-export shims, and unused package surface while keeping the dashboard request dedupe path and shared state primitives aligned with the current shell

## [2.7.0] - 2026-04-21

### Security And Session Hardening

- tightened dashboard-session revocation coordination with the session-manager lock and added concurrent lookup coverage around revocation and replay handling
- continued hardening credential-hash comparison and rotation-driven session invalidation, including additional malformed-hash handling and audit coverage
- hardened loopback request trust around malformed or zone-qualified forwarded client IPs and expanded local-runtime rejection auditing across more operator paths
- switched admin-route authentication onto the same hashed credential-compare path as tenant and service-account credentials, so `VIADUCT_ADMIN_KEY` now stores the persisted `sha256:<hex>` digest instead of the plaintext secret

### Store, Executor, And Runtime Reliability

- split PostgreSQL credential migration into a two-phase flow that seeds durable credential-hash rows first and clears legacy plaintext only after verifying the expected hash rows are present, with rollback coverage for missing preconditions
- made bounded workspace enqueue acknowledgements explicit and typed, removed the direct handoff race from the dispatch loop, and strengthened queued-work timeout coverage
- wired the dashboard auth-session sweeper into the server lifecycle with an explicit stop path so shutdown waits for the pruning goroutine instead of leaving it behind

### Dashboard And Release Engineering

- updated the focus trap so Escape handling, trigger restoration, and trap-root fallback behavior were more resilient on drawer and dialog teardown
- tightened dashboard request dedupe bookkeeping so identical opt-in GETs shared one fetch while still cleaning up the in-flight tracking map after requests settled
- hardened the release workflow around explicit expected bundle manifests, stronger signing-identity validation, and published-bundle checksum verification before the Docker image build reused release binaries
- deferred broader platform items discussed during the v2.7.0 cycle to [`docs/releases/roadmap.md`](docs/releases/roadmap.md) so the published release notes reflected only shipped work

## [2.6.0] - 2026-04-20

### Security Follow-Up Hardening

- normalized malformed credential-hash handling onto a fixed-cost comparison path so zero, malformed, and valid stored hashes all terminate in the same constant-time compare flow
- made dashboard-session revocation durable and atomic across the PostgreSQL revocation write plus the in-memory session cache, then revalidated credential-bound sessions against the current tenant or service-account hash on every lookup
- hardened trusted-forwarded-peer parsing by stripping IPv6 zones before direct peer evaluation while rejecting zoned or non-canonical forwarded addresses, and added explicit `AUDIT` loopback-rejection logs for rejected non-GET local-runtime requests
- kept loopback-only mode authoritative even when `VIADUCT_TRUSTED_PROXIES` is broadly configured, and tightened session invalidation so credential rotation immediately expires sessions bound to the old key material

### Runtime, Store, And Operator Platform

- added a distinct workspace enqueue deadline via `VIADUCT_WORKSPACE_ENQUEUE_TIMEOUT`, hardened worker handoff during shutdown, and covered executor cancellation with high-pressure enqueue stress tests
- upgraded PostgreSQL duplicate-credential preflight failures into actionable configuration errors that list the affected tenant IDs, point operators at `docs/operations/credential-migration.md`, and exit with sysexits-style code `78`
- taught the bundled store migration runner to honor non-transactional migration pragmas so concurrent index migrations stay safe and explicit
- tightened release-artifact verification by pinning `cosign verify-blob` to the exact workflow identity for the tagged release, signing and checksumming `zip` sidecars alongside the existing tarball assets, and adding `set -euo pipefail` to the chained release shell steps

### Dashboard And Console Reliability

- switched in-flight request bookkeeping to UUID-based tracking, made URL dedupe opt-in through `{ dedupe: true }`, preserved external abort reasons, and resolved relative API paths against the page `<base>` tag when the dashboard is hosted below the origin root
- hardened the mobile and dialog focus trap so empty traps keep focus on the container, removed triggers fall back to the trap root instead of `document.body`, and added Testing Library coverage for both edge cases

## [2.5.0] - 2026-04-19

### Upgrade Advisory

- the durable credential-hash uniqueness index shipped in v2.5.0, not v2.4.2; startup validation already blocked known duplicate hashes in v2.4.2, but v2.5.0 is the release that added the explicit concurrent unique-index migration artifact for the credential registry

### Security And Session Hardening

- normalized remote-address loopback checks so direct local runtime bootstrap accepts `127.0.0.1`, `::1`, and IPv4-mapped `::ffff:127.0.0.1` while rejecting unparseable or non-loopback peers
- stopped trusting forwarded client IP and scheme headers by default; `VIADUCT_TRUSTED_PROXIES` now gates `X-Forwarded-For` and `X-Forwarded-Proto`, loopback-bound runtimes stay on strict direct-peer trust, and dashboard session cookies only mark `Secure` when the effective request scheme is actually trusted as HTTPS
- tightened local-runtime CSRF protection so mutating bootstrap requests require a same-listener `Origin` or `Referer` instead of accepting headerless POSTs
- moved API credential comparisons onto fixed-size SHA-256 digests, capped remembered dashboard sessions to seven days unless `VIADUCT_LONG_SESSION_DAYS` is explicitly set, and added durable dashboard-session revocation for self sign-out plus admin-triggered revoke requests

### API, Store, And Runtime Reliability

- added a startup-applied credential-hash unique-index migration artifact for PostgreSQL, plus preflight duplicate detection that keeps the existing remediation guidance when legacy credentials still collide
- reordered PostgreSQL credential migration so durable hash rows are inserted and verified before plaintext API keys are cleared, with rollback coverage for mid-migration failures
- rebuilt the workspace executor around explicit enqueue acknowledgements, typed shutdown and tenant-share errors, per-tenant queue accounting, queue-depth metrics, and request-derived job contexts that still observe server shutdown cancellation
- exposed unauthenticated `/api/v1/ping` runtime smoke readiness, preserved the legacy `/api/v1/inventory` list shape alongside `/api/v2` pagination, and documented that loopback-only local runtime protections are TCP-only

### Dashboard, Tests, And Release Workflow

- restored focus to the original drawer trigger on mobile navigation close, added a container-focus fallback when a dialog has no focusable descendants, and covered both behaviors with Testing Library
- normalized dashboard request dedupe keys to the full request URL, added request-controller leak coverage around synchronous URL normalization failures, and strengthened runtime Playwright coverage for readiness plus legacy inventory compatibility
- changed the runtime smoke harness to synthesize a `/readyz` gate that waits for `/readyz`, `/api/v1/ping`, and an authenticated `/api/v2/inventory?per_page=1` check before browser tests begin
- hardened the release workflow so every `cosign sign-blob` result is immediately verified, `SHA256SUMS` is regenerated after sidecars exist, and the final checksum manifest is signed last

## [2.4.2] - 2026-04-18

### Upgrading From v2.4.1

- `viaduct serve-api` now defaults to loopback and refuses unauthenticated non-loopback listeners unless you configure credentials or pass the explicit dangerous override
- local operator bootstrap now requires a direct `127.0.0.1` browser request to a loopback-bound runtime; same-host reverse proxies and remote hostnames should use tenant or service-account credentials instead
- PostgreSQL credential upgrades now fail fast when legacy tenant or service-account API keys were reused across multiple identities, with a remediation message that tells operators to resolve duplicate keys before restart

### Runtime And Security Hardening

- hardened `serve-api`, local runtime bootstrap, and tenant auth so protected routes no longer inherit ambient default-tenant access and forwarded/proxied requests cannot masquerade as local loopback operator traffic
- migrated tenant and service-account credentials to hashed-at-rest storage with constant-time comparisons, persisted runtime-session digests instead of raw keys, and enforced global credential uniqueness across tenant keys and service-account keys
- replaced direct workspace-job goroutine fan-out with a bounded executor shared by enqueue and startup recovery, tied execution to server lifecycle cancellation, and added bounded-concurrency plus recovery coverage

### Runtime Contract And Operator UX

- aligned the shipped CLI/runtime, backend router, OpenAPI contract, and dashboard around the real same-origin operator path exposed by `viaduct start`, including live docs, runtime auth bootstrap, and browser/runtime smoke coverage against the actual Go runtime
- normalized the touched dashboard request query construction onto `URLSearchParams`, removed the last frontend assumptions about the retired `default-fallback` auth mode, and clarified auth bootstrap messaging for proxied versus direct loopback runtime access
- refreshed the release-facing README/demo screenshots and aligned the release notes, upgrade docs, install docs, lab guidance, and demo collateral around the `v2.4.2` release surface

### Release Workflow Follow-Through

- repaired the tag workflow recovery path so manual dispatch, YAML validation, bundle-sidecar bridging, and cosign sidecar publishing stay reproducible from source control

### Runtime And API Contract

- restored the documented local operator path so `viaduct start` serves the bundled dashboard, backend, and live Swagger docs together while exposing a loopback-only local operator bootstrap through `/api/v1/auth/session`
- aligned the packaged dashboard, API router, and OpenAPI contract around paginated `/api/v2/inventory`, `/api/v2/snapshots`, and `/api/v2/migrations` list routes while explicitly documenting `/api/v1` list responses as legacy compatibility shapes
- kept the workspace-first operator flow fully wired in the shipped backend and added browser coverage that boots both the seeded fixture server and the real `viaduct start` runtime so dashboard auth, workspace discovery, and the operator overview contract stay in sync

### Security And Reliability

- made `viaduct serve-api` bind to loopback by default and refuse unauthenticated remote listeners unless an operator configures credentials or passes an explicit dangerous override
- tightened local operator bootstrap to require direct loopback requests and an explicit auth-session handshake so protected routes no longer inherit ambient default-tenant access
- migrated tenant and service-account credentials to non-recoverable hashes in both stores, kept legacy plaintext PostgreSQL records authenticating through startup migration, and enforced global credential uniqueness across tenant keys and service-account keys
- added actionable upgrade guidance when legacy PostgreSQL credential migration finds reused tenant or service-account keys, and removed the last frontend/runtime references to the retired `default-fallback` auth mode
- replaced direct workspace-job goroutine fan-out with a bounded executor shared by fresh queueing and startup recovery, including lifecycle-aware shutdown and coverage for bounded concurrency and recovery requeue behavior
- normalized dashboard query construction onto `URLSearchParams` for the touched operator routes so filter and report parameters stay encoded consistently

## [2.4.1] - 2026-04-17

### Upgrading From v2.4.0

- no migration-spec, tenant-isolation, or published API-contract break is intended in this patch release
- local release owners should expect `make release-gate` to include connector certification plus dashboard lint, format, unit-test, and build verification
- the local packaging matrix now matches the shipped GitHub release assets across `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64`
- GitHub release notes and the public site’s latest-release surfaces now resolve from source-controlled release files rather than one-off release metadata

### Release Engineering And Maintenance

- aligned `make release-gate` with the repo’s public contract by adding connector certification coverage plus dashboard lint, format, unit-test, and build verification to the canonical local release-owner path
- aligned `make package-release-matrix`, `scripts/package_release`, and `.github/workflows/release.yml` around the shipped artifact matrix: `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64` tarballs plus `dist/SHA256SUMS`
- made the tag workflow source-controlled and reproducible end to end by publishing GitHub releases from versioned files under `docs/releases/` and by tagging GHCR images with both `vX.Y.Z` and `X.Y.Z` alongside `latest`
- refreshed the release toolchain surface with the `node:20.20-bookworm-slim` base image, current GitHub Actions dependencies, and the latest web-tooling maintenance updates already merged on `main`

### Public Site And Docs

- replaced the hardcoded public-site release badge with a dynamic latest-release badge and stable release-notes link so `site/` stays aligned with the current published release surface
- aligned the README, release docs, install docs, support matrix, and demo collateral around the same verification path, packaging matrix, and `v2.4.1` release surface

## [2.4.0] - 2026-04-17

### Upgrading From v2.3.0

- dashboard runtime auth now keeps only the remembered session identifier in `localStorage`; tenant and service-account API keys stay in tab memory and clear on tab close
- anonymous fallback now defaults to the viewer role unless `VIADUCT_ALLOW_ANONYMOUS_ADMIN=true` is explicitly set, so default-tenant bootstrap flows no longer inherit admin rights accidentally
- auth routes now enforce a stricter per-IP limiter and the API server prunes expired auth sessions in the background based on the configured TTLs
- operator health checks are now split between `/healthz` and `/readyz`, and Prometheus metrics are served behind admin authentication for packaged deployments

#### Security And API Hardening

- parameterized snapshot list filtering in `internal/store/postgres.go`, clamped paginated store queries, and added SQL-injection regression coverage in `internal/store/postgres_pagination_test.go`
- downgraded anonymous default-tenant fallback to viewer unless `VIADUCT_ALLOW_ANONYMOUS_ADMIN=true`, tightened tenant auth invariants in `internal/api/middleware.go`, and added a stricter auth-route limiter so principal, credential, and explicitly scoped tenant context mismatches are rejected with correlated warnings instead of silently drifting across tenants
- moved migration background execution in `internal/api/server.go` onto derived server-lifetime contexts, added panic recovery with structured logs, and started shutdown handling before listener startup so cancellation and server teardown stay bounded
- stopped swallowing workspace-job persistence failures in `internal/api/workspaces.go`, always record failed terminal state with `output_json.error`, and cap persisted job output to 1 MiB with an explicit `truncated` signal
- added an auth-session sweeper in `internal/api/auth_session.go` so expired dashboard sessions are pruned in the background and stop cleanly with API shutdown

#### Operator Console And UX Polish

- kept runtime dashboard auth storage limited to short-lived session identifiers in `web/src/runtimeAuth.ts`, moved operator-provided API keys to tab-memory only, and clear corrupted session markers with contextual warnings
- centralized request timeout controller creation in `web/src/api.ts`, threaded abort signals through `web/src/app/useOperatorOverview.ts`, and kept overview refresh cancellation explicit on unmount and refresh replacement
- unified sidebar focus treatment in `web/src/components/navigation/SidebarNav.tsx`, extracted focus trapping into `web/src/components/navigation/useFocusTrap.ts`, and kept the mobile drawer accessible with dialog semantics and focus restoration via `web/src/layouts/AppShell.tsx`
- added workspace filtering and a dedicated empty-filter recovery panel with a clear-filters action in `web/src/features/workspaces/WorkspacePage.tsx`
- surfaced live operator alerts through an explicit polite live region in `web/src/layouts/AppShell.tsx`

#### Observability, Packaging, And Release Workflows

- split operator health probes into `/healthz` and `/readyz`, moved metrics behind admin authentication, and updated packaged health checks plus example deployments in `internal/api/server.go`, `Dockerfile`, `examples/deploy/docker-compose.yml`, and `examples/deploy/kubernetes/deployment.yaml`
- added starter Grafana collateral in `docs/observability/` for the Prometheus metrics surface exposed by the API
- hardened CI in `.github/workflows/ci.yml` with fail-fast frontend a11y linting, `gosec`, and `trivy` filesystem scanning
- added `.github/workflows/release.yml` to build cross-platform bundles, publish GitHub releases from `v*` tags, emit CycloneDX SBOM output with `syft`, and sign release artifacts plus the published container image with keyless `cosign`

## [2.3.0] - 2026-04-16

### Upgrading From v2.2.0

- expect the dashboard to use the refreshed operator shell, updated navigation, and denser card layouts introduced in the v2.3.0 visual refresh
- if you rely on runtime dashboard authentication, verify the browser can keep the cookie-backed session path because the dashboard no longer depends on persisting plaintext API keys between reloads
- when consuming inventory, snapshot, or migration lists programmatically, prefer the paginated `/api/v2` endpoints introduced alongside the v2.3.0 operator hardening work

### Dashboard And Operator Experience

- rebuilt the dashboard visual system around calmer typography, standardized surfaces, clearer hierarchy, and reusable primitives for page headers, cards, notices, stats, and pagination
- refreshed the major operator screens, including the auth bootstrap, pilot workspace flow, inventory assessment, migration workflow, lifecycle views, policy surfaces, drift views, reports, and settings
- improved dense operational layouts so inventory and workspace review stay readable on laptop, desktop, tablet, and mobile breakpoints without depending on awkward overflow behavior

### Accessibility And Interaction Quality

- replaced the collapsed navigation slide-over with a keyboard-safe drawer that supports focus trapping, Escape dismissal, focus restoration, and explicit expanded-state signaling
- corrected the workspace workload-detail action so it now truthfully saves selection state instead of advertising a migration-plan action it did not perform
- added stronger toggle semantics, cleaner checkbox behavior, and explicit detail-panel reveal behavior when workloads are inspected on stacked layouts

### Release Surfaces And Collateral

- removed runtime dashboard font CDN dependencies so packaged and offline dashboard assets remain self-contained
- refreshed the public README, release notes, screenshot galleries, and demo collateral with current seeded-product captures instead of stale illustrative SVGs
- aligned the public site release badge, dashboard package metadata, and release docs around the `v2.3.0` release surface

## [2.2.0] - 2026-04-16

### Fixed

- fail fast on invalid lifecycle policy bundles during API startup while warning cleanly when the policy directory is absent
- downgrade anonymous default-tenant fallback to viewer unless `VIADUCT_ALLOW_ANONYMOUS_ADMIN=true` is explicitly set, add pre-auth IP rate limiting for auth flows, and tighten same-origin/CORS defaults for API-key deployments
- move dashboard runtime auth off browser-stored plaintext API keys onto cookie-backed sessions, log corrupted session payload cleanup, and add request timeout plus abort handling for operator overview refreshes
- stop swallowing API error-response and Swagger JSON write failures by logging request-correlated server-side errors
- add store-backed pagination totals for snapshot and migration history so large tenants no longer require loading entire result sets into memory

### Added

- frontend lint, format, unit-test, and Playwright CI coverage for the dashboard workspace and auth flows
- structured request logging, security headers, OpenAPI JSON generation checks, and database pool diagnostics for the API server
- versioned `/api/v2` paginated inventory, snapshot, and migration list endpoints while preserving legacy `/api/v1` response shapes for existing clients

## [2.1.0] - 2026-04-15

### Dashboard And UI Polish

- normalized all custom `rounded-[...]` border-radius values to standard Tailwind steps (`rounded-xl`, `rounded-2xl`) throughout the entire dashboard — removes the bubbly appearance and gives the operator console a crisper, more professional look
- replaced per-item description lines in the sidebar navigation with compact icon-and-label rows, reducing sidebar height by roughly 60% and making the nav feel like an operator tool rather than a feature brochure
- slimmed the sidebar brand panel by removing the "Default flow" and "Shared truth" info callouts and the "Operator path" section, leaving a clean brand mark and navigation
- simplified the TopBar to a compact header strip — removed the 5-column metric grid, removed the static "REST API + shared store" and "Tenant-scoped visibility" badges that carried no operator signal; metrics remain available on the Dashboard page
- fixed duplicate status badge labels in metric cards and `SignalRow` components — badges now show a meaningful status word ("Healthy", "Attention", "Critical") instead of repeating the adjacent label text
- removed the outer `.panel` wrapper that surrounded all page content in `AppShell`, eliminating an extra layer of nesting that was fighting with `PageHeader` and `SectionCard` panels on each page
- corrected page heading hierarchy — `PageHeader` titles are now `text-2xl` (the page's primary heading); the TopBar shows a compact navigation label at `text-base`
- assigned unique icons to every navigation route — `/workspaces` now uses `FolderKanban`, `/inventory` uses `Server`, `/lifecycle` uses `TrendingUp`, `/drift` uses `GitCompare`, `/graph` uses `Network`; no two routes share the same icon
- added `MobileSidebarDrawer` — a hamburger button and slide-in drawer are now available on viewports below the `xl` breakpoint (1280 px), replacing the previous stacked-sidebar behaviour on tablets and laptops
- replaced the bare error paragraph in `AppShell` with a new `ErrorBanner` component that includes an `AlertTriangle` icon, a `role="alert"` attribute, and an optional dismiss button

## [2.0.0] - 2026-04-11

### Installation, Startup, And First Run
- added `viaduct start`, `viaduct stop`, and `viaduct doctor` so the default local experience is now one WebUI-first runtime instead of a manual multi-step API bootstrap
- taught `viaduct start` to generate the default local lab config automatically when `~/.viaduct/config.yaml` is missing and to point it at the shipped KVM fixtures
- added recorded local runtime status reporting through `viaduct status --runtime`, including the WebUI URL, API URL, PID, and runtime log location
- updated the Unix and Windows install scripts to copy bundled docs, examples, and configs together and to generate a starter config for the installed lab path

### Dashboard, Site, And Product Surfaces
- aligned the dashboard runtime auth flow with the built-in local single-user fallback so the default local lab path no longer requires pasted browser credentials
- synchronized the dashboard, root docs, lab docs, troubleshooting guidance, deployment examples, public site, and release-facing screenshots around the new local startup model
- refreshed the public website and social-card copy to emphasize installation, startup, workspace progression, and controlled operator workflows more clearly

### Verification, Packaging, And Release Readiness
- extended automated CLI coverage with tests around local runtime paths and starter-config generation
- kept `make release-gate`, `make certification-test`, `make plugin-check`, `make contract-check`, and `make package-release-matrix` aligned with the same packaged product surface
- added `v2.0.0` release notes and synchronized visible version markers, screenshot labels, and package metadata around the new release

## [1.9.0] - 2026-04-11

### Install, Packaging, And Startup Flow
- taught `viaduct serve-api` to serve built dashboard assets from the repo build output, packaged bundles, and installed asset paths so the default operator path is now one same-origin process
- added an explicit `--web-dir` override plus `VIADUCT_WEB_DIR` support for non-standard packaged asset layouts
- aligned the Windows install script with the shared `share/viaduct/web` layout used on Unix-like installs

### Deployment And Operator Experience
- corrected container, Docker Compose, and Kubernetes command wiring so the shipped image starts the intended `viaduct serve-api` process cleanly
- moved the first-run and lab documentation to the WebUI-first path at `http://localhost:8080` while preserving the Vite dev-server flow for frontend development
- tightened troubleshooting, configuration, upgrade, and deployment guidance around the same packaged dashboard, CLI, and API behavior

### Release And Demo Surfaces
- refreshed release-facing screenshot labels and demo references for the `v1.9.0` product surface
- added `v1.9.0` release notes and synchronized the changelog, dashboard package metadata, and release-facing docs around the new version

## [1.8.0] - 2026-04-11

### Dashboard And Operator Experience
- refreshed the operator dashboard shell, page hierarchy, runtime auth recovery, inventory presentation, and workspace-first progression so the web UI reads more like a serious control plane
- added clearer workflow guidance, state markers, empty states, loading states, and recovery language around the workspace path from authentication through report export
- tightened the shared dashboard component system for headers, cards, badges, tables, and operator-facing status callouts

### Public Website And Documentation
- rewrote the public `site/` landing page, 404 surface, metadata, and social-card copy around discovery, dependency mapping, migration planning, supervised execution, and operator-visible reporting
- aligned root docs, quickstart language, release notes, demo assets, and deeper workflow guides with the refreshed dashboard and public-site terminology
- removed stale internal positioning language from tracked public docs so the repository, release surfaces, and website describe the same product clearly

## [1.7.0] - 2026-04-11

### Workspace Reliability And Operator Hardening
- added stricter validation for pilot workspace create, update, job, and report-export requests so invalid operator input fails early with field-level API errors
- added read-only workspace access for viewer principals while keeping workspace mutation and job execution operator-scoped
- added workspace deletion, restart recovery for queued or running workspace jobs, configurable workspace job timeouts, and richer exported report handoff detail

### Dashboard And Operator Experience
- switched runtime dashboard auth to session-scoped storage by default with an explicit remember option for trusted browsers
- added workspace creation toggles, persisted job history, retry actions, and clearer correlation-aware job states in the workspace-first flow

### Release Engineering, Docs, And Contract
- hardened the Windows release-gate helpers so race, coverage, and CLI smoke validation remain reproducible on Application Control-constrained operator workstations
- documented `VIADUCT_ALLOWED_ORIGINS` and `VIADUCT_WORKSPACE_JOB_TIMEOUT`
- updated the pilot workspace guide, quickstarts, installation guides, and OpenAPI contract to match the hardened workspace and auth behavior

## [1.6.0] - 2026-04-11

### Workspace-First Operator Flow
- added a first-class pilot workspace model that persists source connections, discovery snapshots, dependency graph output, target assumptions, readiness results, saved plans, approvals, notes, and exported reports
- added tenant-scoped API routes for listing, creating, updating, and exporting pilot workspace state without introducing a parallel product surface
- added persisted background jobs for workspace discovery, graph generation, simulation, and plan generation so the operator workflow can survive page refreshes and produce reproducible state

### Dashboard And Auth Bootstrap
- reworked the dashboard so the first operator experience is create workspace, discover, inspect, simulate, save plan, and export report
- added runtime dashboard authentication bootstrap using service-account or tenant keys instead of relying on build-time-only configuration
- strengthened loading, empty, retry, and request-correlation-aware error handling across the workspace flow

### Lab, Contract, And Release Surface
- added a deterministic `examples/lab` end-to-end smoke flow for workspace creation through report export
- updated the published OpenAPI contract, quickstart flow, lab assets, configuration guidance, and operator docs to match the new workspace APIs and runtime auth flow
- added v1.6.0 release-note material and release-facing screenshot assets for the workspace-first operator application

## [1.5.0] - 2026-04-08

### Early Product Hardening
- narrowed the public product story around VMware-exit migration assessment and supervised pilot planning with explicit v1 scope, reliability-path, trust-control, observability, validation, demo, and commercialization artifacts
- aligned repo entrypoint docs so the current product direction, support boundary, and operator guidance are easier to evaluate from the packaged and source workflows

### API And Dashboard Trust Surfaces
- hardened the API contract with structured JSON error responses, stabilized migration command acknowledgements, and updated OpenAPI coverage for the operator-facing routes
- improved dashboard-side error handling so settings and report workflows preserve request correlation and operator-facing failure detail instead of flattening backend errors into generic strings

### Documentation And Operator Readiness
- added presenter-ready demo assets, real-user validation templates, and commercialization decision guidance to support design-partner conversations and pilot packaging
- refreshed quickstart, configuration, troubleshooting, and multi-tenancy guidance so service accounts, trust controls, and the supported pilot workflow are documented more consistently

## [1.4.2] - 2026-04-08

### Release Reliability
- completed the dependency graph TypeScript fix so the D3 link endpoint handlers compile cleanly during `make release-gate`
- superseded the `v1.4.1` candidate tag before publishing a downloadable GitHub release

## [1.4.1] - 2026-04-08

### Release Reliability
- fixed the dashboard dependency graph typing so `make release-gate` can complete the web build and package the release bundle
- superseded the `v1.4.0` candidate tag before publishing a downloadable GitHub release

## [1.4.0] - 2026-04-08

### Dashboard Product Workflow
- reorganized the React dashboard around a clearer app shell, navigation model, and feature-oriented page structure
- turned migration planning into an operator workflow with intake, validation, saved-plan review, and execution-preparation states instead of a detached wizard
- improved inventory, dependency, and remediation surfaces so planning context stays connected to the broader operator view

### Operator Authentication And Configuration
- added dashboard support for service-account API keys alongside tenant API keys
- documented the new dashboard environment variable contract for local development and release packaging

### Public Web Presence
- added a standalone static `site/` for the public project surface
- added a GitHub Pages workflow to publish the site independently from the product dashboard build

## [1.3.0] - 2026-04-07

### Tenant Isolation And Operability
- added tenant-scoped permission enforcement and richer tenant introspection for service-account automation
- added store diagnostics, API build metadata, and operational metrics/reporting surfaces

### Backup And Plugin Ecosystem
- added backup continuity and backup-policy drift validation for post-migration portability checks
- added plugin manifest validation tooling and a release-facing plugin certification guide

### Release And Deployment Experience
- added OpenAPI contract checks to the release workflow and published the stable operator contract reference
- hardened deployment references for Docker Compose, systemd, and Kubernetes pilots

## [1.2.0] - 2026-04-07

### Tenant Security And Scale
- added tenant-scoped service accounts with viewer, operator, and admin roles for API authentication
- added role-gated tenant routes and a current-tenant introspection route without leaking API keys
- added tenant quotas for API request rate, snapshot count, and migration count

### Migration And API Correctness
- fixed current-inventory aggregation so the API no longer misses sources once snapshot history grows past twenty entries
- replaced brittle pending-approval summary detection with real migration-state decoding
- wired migration `credential_ref` resolution through the CLI config and API server connector-resolution paths
- added the `/api/v1/about` route for operator-visible build and compatibility metadata

### Plugin And Release Operability
- added optional plugin host-version compatibility markers in `plugin.json`
- added a machine-readable `dependency-manifest.json` to packaged release bundles
- expanded regression coverage for service-account auth, quota enforcement, plugin compatibility, packaging metadata, and summary correctness

## [1.1.0] - 2026-04-05

### Current Tagged Feature Set
- multi-platform discovery for VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup inventory
- dependency graph construction across workload, network, storage, and backup metadata
- declarative cold and warm migration orchestration with preflight checks, execution windows, approval gates, checkpoints, resume support, verification, and rollback
- lifecycle cost, policy, drift, remediation, and simulation workflows
- multi-tenancy, tenant-scoped API access, persistent state backends, and plugin hosting
- React dashboard, reproducible release packaging, install scripts, and a shared release gate

### Operability And Ecosystem
- added connector certification coverage and a tagged soak-test path for large-wave migration exercises
- added deployment reference assets for Docker Compose, systemd, and Kubernetes-based pilot environments
- added plugin manifest validation and config-aware plugin connection handling so plugin connectors receive the same auth and transport settings as built-in connectors
- added tenant-scoped audit exports, request correlation headers, API metrics, and basic tenant rate limiting to improve diagnostics without changing core workflows

### Maintenance
- refreshed the dashboard stack to React 19, Vite 8, and `@vitejs/plugin-react` 6 with a Node 20.19+ baseline
- grouped Dependabot updates more conservatively and ignored Docker base-image major jumps until they are evaluated intentionally
- aligned the web TypeScript configuration with Vite's bundler module resolution and deferred semver-major Tailwind CSS and TypeScript jumps until a dedicated migration pass

### Repository Professionalization
- aligned top-level docs, roadmap archives, examples, and community files with the implemented codebase
- added release-era install, quickstart, upgrade, release, support, and troubleshooting entrypoints
- improved directory onboarding for docs, configs, examples, API assets, tests, and the dashboard

## [1.0.0] - 2026-04-05

### Highlights
- shipped the first tagged Viaduct release with a release-gated CLI, API, dashboard, install scripts, packaged web assets, checksums, and release manifest generation
- delivered multi-platform discovery for VMware, Proxmox, Hyper-V, KVM, Nutanix, and Veeam-related backup inventory
- delivered dependency-aware migration planning, cold and warm migration workflows, execution windows, approval gates, checkpoints, resume support, verification, and rollback
- delivered lifecycle cost, policy, drift, remediation, and simulation workflows
- delivered tenant-scoped API access, persistent state backends, plugin hosting, contributor docs, operator runbooks, and example lab environments

## Historical Implementation Milestones

### Phase 4 Complete
- added execution windows, approval gates, wave planning, resume support, lifecycle remediation guidance, simulation flows, tenant summary reporting, and stronger release gating

### Phase 3 Complete
- added warm migration, lifecycle management, backup portability, KVM and Nutanix connectors, multi-tenancy, and plugin hosting

### Phase 2 Complete
- added cold migration orchestration, Veeam and Hyper-V discovery, the dependency graph, and the operator dashboard

### Phase 1 Complete
- added VMware and Proxmox discovery, normalization, state persistence, and discovery CLI workflows

### Phase 0 Complete
- created the repository foundation, universal schema, connector interfaces, CLI skeleton, CI, and project governance files
