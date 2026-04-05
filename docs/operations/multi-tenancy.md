# Multi-Tenancy Guide

Viaduct's API and store layers are tenant-scoped. Tenant isolation should be treated as a release-blocking concern for any deployment.

## Tenant Authentication
- Tenant API routes use `X-API-Key`
- Admin routes use `X-Admin-Key`
- Tenant context is injected by middleware and propagated into store operations

## Key Routes
- `GET /api/v1/inventory`
- `GET /api/v1/snapshots`
- `GET /api/v1/migrations`
- `POST /api/v1/migrations`
- `GET /api/v1/audit`
- `GET /api/v1/reports/<name>`
- `GET /api/v1/summary`
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

## State Isolation
- snapshots are tenant-scoped
- migrations are keyed by `(tenant_id, id)` in PostgreSQL
- recovery points are tenant-scoped
- audit events and CSV/JSON reports are tenant-scoped
- tenant summaries are derived through tenant-aware store helpers

## Operational Recommendations
- Use PostgreSQL for any multi-tenant environment.
- Keep admin and tenant API keys separate.
- Keep request correlation IDs when integrating Viaduct behind reverse proxies or shared control planes.
- Validate tenant A/B isolation with integration tests or a staging environment before onboarding multiple customers or business units.
- Treat any cross-tenant leakage as critical severity.
