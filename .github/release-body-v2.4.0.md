## Viaduct v2.4.0

### Upgrading From v2.3.0

- dashboard runtime auth now keeps only the remembered session identifier in `localStorage`; tenant and service-account API keys stay in tab memory and clear on tab close
- anonymous fallback now defaults to the viewer role unless `VIADUCT_ALLOW_ANONYMOUS_ADMIN=true` is explicitly set, so default-tenant bootstrap flows no longer inherit admin rights accidentally
- auth routes now enforce a stricter per-IP limiter and the API server prunes expired auth sessions in the background based on the configured TTLs
- operator health checks are now split between `/healthz` and `/readyz`, and Prometheus metrics are served behind admin authentication for packaged deployments

### Security And API Hardening

- parameterized snapshot list filtering in `internal/store/postgres.go`, clamped paginated store queries, and added SQL-injection regression coverage in `internal/store/postgres_pagination_test.go`
- downgraded anonymous default-tenant fallback to viewer unless `VIADUCT_ALLOW_ANONYMOUS_ADMIN=true`, tightened tenant auth invariants in `internal/api/middleware.go`, and added a stricter auth-route limiter so principal, credential, and explicitly scoped tenant context mismatches are rejected with correlated warnings instead of silently drifting across tenants
- moved migration background execution in `internal/api/server.go` onto derived server-lifetime contexts, added panic recovery with structured logs, and started shutdown handling before listener startup so cancellation and server teardown stay bounded
- stopped swallowing workspace-job persistence failures in `internal/api/workspaces.go`, always record failed terminal state with `output_json.error`, and cap persisted job output to 1 MiB with an explicit `truncated` signal
- added an auth-session sweeper in `internal/api/auth_session.go` so expired dashboard sessions are pruned in the background and stop cleanly with API shutdown

### Operator Console And UX Polish

- kept runtime dashboard auth storage limited to short-lived session identifiers in `web/src/runtimeAuth.ts`, moved operator-provided API keys to tab-memory only, and clear corrupted session markers with contextual warnings
- centralized request timeout controller creation in `web/src/api.ts`, threaded abort signals through `web/src/app/useOperatorOverview.ts`, and kept overview refresh cancellation explicit on unmount and refresh replacement
- unified sidebar focus treatment in `web/src/components/navigation/SidebarNav.tsx`, extracted focus trapping into `web/src/components/navigation/useFocusTrap.ts`, and kept the mobile drawer accessible with dialog semantics and focus restoration via `web/src/layouts/AppShell.tsx`
- added workspace filtering and a dedicated empty-filter recovery panel with a clear-filters action in `web/src/features/workspaces/WorkspacePage.tsx`
- surfaced live operator alerts through an explicit polite live region in `web/src/layouts/AppShell.tsx`

### Observability, Packaging, And Release Workflows

- split operator health probes into `/healthz` and `/readyz`, moved metrics behind admin authentication, and updated packaged health checks plus example deployments in `internal/api/server.go`, `Dockerfile`, `examples/deploy/docker-compose.yml`, and `examples/deploy/kubernetes/deployment.yaml`
- added starter Grafana collateral in `docs/observability/` for the Prometheus metrics surface exposed by the API
- hardened CI in `.github/workflows/ci.yml` with fail-fast frontend a11y linting, `gosec`, and `trivy` filesystem scanning
- added `.github/workflows/release.yml` to build cross-platform bundles, publish GitHub releases from `v*` tags, emit CycloneDX SBOM output with `syft`, and sign release artifacts plus the published container image with keyless `cosign`
