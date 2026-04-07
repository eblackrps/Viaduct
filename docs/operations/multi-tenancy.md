# Multi-Tenancy Guide

Viaduct's API and store layers are tenant-scoped. Tenant isolation should be treated as a release-blocking concern for any deployment.

## Tenant Authentication
- Tenant API routes use `X-API-Key`
- Tenant service accounts can use `X-Service-Account-Key`
- Admin routes use `X-Admin-Key`
- Tenant context is injected by middleware and propagated into store operations

## Tenant Roles
- `viewer`: read-only access to inventory, reports, summaries, and lifecycle views
- `operator`: viewer access plus preflight, simulation, planning, execution, resume, and rollback routes
- `admin`: full tenant-scoped access including service-account administration

## Service Accounts
- `GET /api/v1/tenants/current` returns the effective tenant context, role, quotas, and auth method without leaking API keys
- `GET /api/v1/service-accounts` lists existing tenant service accounts
- `POST /api/v1/service-accounts` creates a tenant service account
- `POST /api/v1/service-accounts/<service-account-id>/rotate` rotates a service-account API key
- Service-account list responses intentionally redact API keys; create and rotate responses return the new key once

## Quotas
- Tenants can define `requests_per_minute`, `max_snapshots`, and `max_migrations`
- request quotas feed the API rate limiter
- snapshot and migration quotas are enforced by the shared store, so CLI, API, and background workflows see the same limit behavior
- tenant summaries expose remaining snapshot and migration quota headroom when configured

## Key Routes
- `GET /api/v1/inventory`
- `GET /api/v1/snapshots`
- `GET /api/v1/migrations`
- `POST /api/v1/migrations`
- `GET /api/v1/audit`
- `GET /api/v1/reports/<name>`
- `GET /api/v1/summary`
- `GET /api/v1/tenants/current`
- `GET /api/v1/service-accounts`
- `POST /api/v1/service-accounts`
- `POST /api/v1/service-accounts/<service-account-id>/rotate`
- `POST /api/v1/admin/tenants`
- `DELETE /api/v1/admin/tenants/<tenant-id>`

## Create A Tenant

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: <admin-key>" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants
```

## Default Tenant Behavior
- The built-in `default` tenant exists automatically.
- Default-tenant fallback only applies when there are no active custom tenants and the default tenant has no API key.
- Production operators should configure explicit tenant keys instead of relying on fallback behavior.
- Production operators should prefer service accounts for automation instead of sharing the tenant-wide API key.

## State Isolation
- snapshots are tenant-scoped
- migrations are keyed by `(tenant_id, id)` in PostgreSQL
- recovery points are tenant-scoped
- audit events and CSV/JSON reports are tenant-scoped
- tenant summaries are derived through tenant-aware store helpers

## Operational Recommendations
- Use PostgreSQL for any multi-tenant environment.
- Keep admin and tenant API keys separate.
- Use tenant service accounts for automation and rotate them on a schedule.
- Set tenant quotas deliberately so noisy environments do not crowd out shared control-plane capacity.
- Keep request correlation IDs when integrating Viaduct behind reverse proxies or shared control planes.
- Validate tenant A/B isolation with integration tests or a staging environment before onboarding multiple customers or business units.
- Treat any cross-tenant leakage as critical severity.
