# Early Product Auth, Role, And Auditability Model

This document defines the first credible trust-control model for Viaduct as an early product. It is intentionally narrow. The goal is to make tenant-scoped evaluation and supervised pilot use feel trustworthy and accountable without pretending Viaduct already has enterprise identity, compliance, or policy infrastructure.

## 1. Current State Summary

### What exists today
- `internal/api/middleware.go` already supports three credential paths:
  - `X-Admin-Key` for platform-level admin routes
  - `X-API-Key` for tenant credentials
  - `X-Service-Account-Key` for tenant-scoped service accounts
- `internal/models/tenant.go` already defines three tenant roles and six permissions:
  - roles: `viewer`, `operator`, `admin`
  - permissions: `inventory.read`, `reports.read`, `lifecycle.read`, `migration.manage`, `tenant.read`, `tenant.manage`
- `internal/api/server.go` already enforces role and permission boundaries per route instead of relying on frontend-only assumptions.
- `GET /api/v1/tenants/current` already exposes effective role, permissions, auth method, and service-account identity for the active caller.
- `internal/models/audit.go` and the store layer already persist tenant-scoped audit events with `id`, `tenant_id`, `actor`, `request_id`, `category`, `action`, `resource`, `outcome`, `message`, `details`, and `created_at`.
- `internal/api/reports.go` already exposes tenant-scoped audit history at `GET /api/v1/audit` and `GET /api/v1/reports/audit`.
- `internal/api/observability.go` already generates or forwards `X-Request-ID`, returns it in API responses, and logs request-scoped metadata.
- The dashboard already exposes some trust context:
  - `web/src/features/settings/SettingsPage.tsx` shows auth method, role, service-account name, and effective permissions.
  - `web/src/features/reports/ReportsPage.tsx` allows report and audit export through the current credential context.

### What prior phases likely improved
- Phase 3 appears to have introduced multi-tenancy, tenant-scoped persistence, service accounts, quotas, and role-aware routing.
- Phase 4 appears to have made migration execution resumable, approval-aware, and checkpoint-driven, which makes action attribution and audit history materially more important.
- Phase 5 appears to have hardened API contracts with structured errors, request IDs, and a more explicit operator contract in `docs/reference/openapi.yaml`.

### What is still weak, ambiguous, over-scoped, or risky
- Default-tenant fallback still exists for local/lab convenience. That is useful for demos but not a credible trust posture for real evaluation or pilot use.
- A tenant API key currently authenticates as tenant `admin`. That is acceptable for bootstrap and break-glass use, but weak for day-to-day operator attribution.
- Human attribution is only as good as the credential in use. If multiple humans share one tenant key, audit history collapses to `tenant:<tenant-id>`.
- Platform-admin attribution is also weak today. `AdminAuthMiddleware` authenticates a shared `X-Admin-Key`, and admin audit events currently record only `Actor: "admin"`.
- Audit coverage is incomplete for trust-sensitive actions. Today it is strongest for migration commands and service-account changes, but not yet consistent for report exports or authenticated authorization denials.
- The audit schema is intentionally small, but details are not yet normalized into a stable convention for migration commands, approval context, or export activity.
- The role model has an important implementation nuance that is easy to misread: explicit service-account permissions narrow access inside the service account's role ceiling. They do not override `RequireTenantRole` checks in `internal/api/middleware.go`.
- The dashboard exposes current identity context in Settings, but not yet the action attribution an operator expects in migration history and execution detail views.
- `GET /api/v1/migrations` and `GET /api/v1/migrations/{id}` are currently `migration.manage` routes. That means the `viewer` role cannot inspect migration history today, even though it can read audit history and reports.
- Audit durability depends on the configured store. The in-memory store is useful for development but not a credible source of truth for pilot-grade audit history.

### What should be preserved
- The current API-key-based early-product posture.
- The three-role model already present in code.
- Explicit service-account permissions as a narrowing mechanism inside the existing role boundary, not a separate custom RBAC system.
- Tenant-scoped routing, storage, reporting, and request-correlation behavior.
- The existing dashboard Settings surface as the place where current caller context is explained.

### The smallest credible next move
- Freeze one early-product trust model around the primitives Viaduct already has:
  - separate platform admin and tenant credentials
  - three tenant roles
  - service accounts as the preferred non-break-glass identity
  - audit events for all trust-sensitive mutations and exports
  - visible attribution in existing dashboard surfaces

## 2. Proposed Trust-Control Model

Viaduct should adopt the following early-product trust stance:

1. Keep authentication simple and explicit.
2. Treat the tenant boundary as the primary security boundary.
3. Use service accounts for named access whenever possible.
4. Keep the role model small enough to reason about during pilots.
5. Make state-changing actions and audit/report exports traceable through one tenant-scoped audit trail.
6. Show operators who performed important actions, through which credential path, and with which request ID.

This is not an SSO-first or enterprise IAM design. It is the minimum model that makes Viaduct feel like serious infrastructure software during evaluation and supervised pilot work.

## 3. Authentication Expectations For The Early Product Stage

### Credential types and intended use

| Credential | Current mechanism | Intended use in early product | Should be used for |
| --- | --- | --- | --- |
| Platform admin key | `X-Admin-Key` | Platform bootstrap and tenant administration | Creating and deleting tenants |
| Tenant key | `X-API-Key` | Tenant bootstrap and break-glass admin access | Initial setup, emergency admin actions, short-lived direct admin work |
| Service-account key | `X-Service-Account-Key` | Normal named access for operators and automation | Dashboard access, scripted operations, export jobs, migration workflows |

### Early-product authentication rules
- Viaduct v1 does not need SSO, OIDC, browser sessions, or SCIM.
- Viaduct does need explicit, named credentials with a clear intended use.
- Tenant API keys should exist, but they should not be the recommended default for routine operator activity.
- Dashboard and automation guidance should prefer service accounts over a shared tenant key.
- In the current web client, that means `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` should be the default documented credential path, while `VITE_VIADUCT_API_KEY` should be documented as bootstrap or break-glass usage.
- Default-tenant fallback should remain available for local lab usage but should be treated as non-production behavior in docs, demos, and pilot guidance.
- Pilots should run behind TLS termination or a trusted reverse proxy. Viaduct's early trust model assumes transport security is provided by the deployment environment.

### Early-product operator identity stance
- When Viaduct does not yet have human user accounts, the credible substitute is a named service account per operator or per automation context.
- Service-account metadata should carry an owner or purpose label so audit trails remain understandable.
- Shared credentials should be treated as temporary exceptions, not the default operating pattern.

## 4. Minimum Viable Role Model

Viaduct should keep the current three-role model.

| Role | Current repo status | Intended user | Trust boundary |
| --- | --- | --- | --- |
| `viewer` | Implemented | Read-only evaluator, stakeholder, support reviewer | Can inspect tenant state but cannot change migration or tenant state |
| `operator` | Implemented | Migration operator or platform engineer running planned work | Can plan, validate, execute, resume, roll back, and simulate |
| `admin` | Implemented | Tenant owner or platform lead | Can do all operator work plus tenant/service-account administration |

### Why this is enough for v1
- It matches the code already shipping in `internal/models/tenant.go`.
- It is easy to explain in a pilot.
- It avoids a custom-role UI or policy editor before the product has validated its workflow boundaries.
- It still allows narrower automation through explicit service-account permissions without changing the route model.

### What not to add yet
- Custom roles
- Per-resource ACLs
- Team/group mapping
- Approval-policy authoring by persona
- Fine-grained field-level or object-level permissions

### Clarifications that avoid scope confusion
- Explicit service-account permissions are not a privilege-escalation mechanism. They only matter after the route's role gate has already passed.
- Viaduct does not need a new `migration.read` permission in this step. The current route boundary remains intact for v1.
- Because the current route boundary remains intact, read-only migration oversight for `viewer` is limited to summaries, reports, and audit history, not `GET /api/v1/migrations`.
- Admin-key actions remain only weakly attributed in v1. Do not claim stronger proof of admin identity than the current shared-key model actually provides.

## 5. Permission Boundaries For Major Actions

### Roles and permissions matrix

| Action | Route or surface | Permission | Viewer | Operator | Admin | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Read inventory | `GET /api/v1/inventory` | `inventory.read` | Yes | Yes | Yes | Includes graph and snapshots |
| Read graph and snapshots | `GET /api/v1/graph`, `GET /api/v1/snapshots` | `inventory.read` | Yes | Yes | Yes | Read-only analysis path |
| Run preflight | `POST /api/v1/preflight` | `migration.manage` | No | Yes | Yes | Operator path starts here |
| Create migration plan | `POST /api/v1/migrations` | `migration.manage` | No | Yes | Yes | Creates persisted migration state |
| Inspect migration state | `GET /api/v1/migrations`, `GET /api/v1/migrations/{id}` | `migration.manage` | No | Yes | Yes | Current route is operator-scoped |
| Execute migration | `POST /api/v1/migrations/{id}/execute` | `migration.manage` | No | Yes | Yes | High-trust action, always auditable |
| Resume migration | `POST /api/v1/migrations/{id}/resume` | `migration.manage` | No | Yes | Yes | High-trust action, always auditable |
| Roll back migration | `POST /api/v1/migrations/{id}/rollback` | `migration.manage` | No | Yes | Yes | High-trust action, always auditable |
| Read cost/policy/drift/summary | `GET /api/v1/costs`, `GET /api/v1/policies`, `GET /api/v1/drift`, `GET /api/v1/summary` | `lifecycle.read` | Yes | Yes | Yes | Read-only lifecycle analysis |
| Read audit history | `GET /api/v1/audit` | `reports.read` | Yes | Yes | Yes | Viewer access is acceptable for tenant-local transparency |
| Export reports | `GET /api/v1/reports/*` | `reports.read` | Yes | Yes | Yes | Export action itself should be auditable |
| Read current tenant context | `GET /api/v1/tenants/current` | `tenant.read` | Yes | Yes | Yes | Powers Settings trust context |
| List service accounts | `GET /api/v1/service-accounts` | `tenant.manage` | No | No | Yes | Admin-only tenant administration |
| Create service account | `POST /api/v1/service-accounts` | `tenant.manage` | No | No | Yes | Admin-only |
| Rotate service-account key | `POST /api/v1/service-accounts/{id}/rotate` | `tenant.manage` | No | No | Yes | Admin-only |
| Create or delete tenant | `/api/v1/admin/tenants*` | Admin key only | No | No | No | Outside tenant role model |

### Operator vs admin distinction
- `operator` is the person who can move workloads.
- `admin` is the person who can change who is allowed to move workloads.
- The same human may hold both in a small pilot, but the actions are still different and should remain separately modeled.
- Tenant administration should not be required for routine migration operations.
- Migration execution should not imply service-account management authority.

## 6. Actions That Should Always Be Auditable

The early-product bar is not "audit everything." The bar is "audit every trust-sensitive change and every trust-sensitive export."

### Always auditable in v1

| Action class | Current status | v1 expectation |
| --- | --- | --- |
| Tenant create/delete | Already audited | Keep |
| Service-account create | Already audited | Keep |
| Service-account key rotate | Already audited | Keep |
| Migration plan creation | Already audited | Keep |
| Migration execute | Already audited for success and some failures | Normalize details and keep |
| Migration resume | Already audited for success and some failures | Normalize details and keep |
| Migration rollback | Already audited for success and failure | Keep |
| CSV or JSON export from `GET /api/v1/reports/{summary|migrations|audit}` | Not consistently audited | Add |
| Authenticated permission denial (`403`) on trust-sensitive routes | Not consistently audited | Add when principal is known |

### Useful but not required for v1
- Every inventory read
- Every dashboard page view
- Unauthenticated invalid credential attempts as tenant audit events
- Full audit search, retention, and filtering UI
- Immutable/WORM audit retention

### Why this line is practical
- It captures who changed state or extracted reportable data.
- It avoids flooding the audit log with low-value read noise.
- It keeps the implementation centered on the existing `AuditEvent` model.

## 7. Canonical Action Vocabulary And Repo Ownership

The model only becomes executable if audit action names and ownership points are fixed. Viaduct should use the following route-to-action mapping for v1.

| Route or handler | Current implementation point | Category | Action | Notes |
| --- | --- | --- | --- | --- |
| `POST /api/v1/admin/tenants` | `handleAdminTenants` in `internal/api/server.go` | `admin` | `create-tenant` | Already emitted |
| `DELETE /api/v1/admin/tenants/{id}` | `handleAdminTenantByID` in `internal/api/server.go` | `admin` | `delete-tenant` | Already emitted |
| `POST /api/v1/service-accounts` | `handleServiceAccounts` in `internal/api/tenant_admin.go` | `tenant` | `create-service-account` | Already emitted |
| `POST /api/v1/service-accounts/{id}/rotate` | `handleServiceAccountByID` in `internal/api/tenant_admin.go` | `tenant` | `rotate-service-account-key` | Already emitted |
| `POST /api/v1/migrations` | `handleMigrations` in `internal/api/server.go` | `migration` | `plan` | Already emitted |
| `POST /api/v1/migrations/{id}/execute` | `handleMigrationByID` in `internal/api/server.go` | `migration` | `execute` | Already emitted; details need normalization |
| `POST /api/v1/migrations/{id}/resume` | `handleMigrationByID` in `internal/api/server.go` | `migration` | `resume` | Already emitted; details need normalization |
| `POST /api/v1/migrations/{id}/rollback` | `handleMigrationByID` in `internal/api/server.go` | `migration` | `rollback` | Already emitted |
| `GET /api/v1/reports/summary` | `writeSummaryReport` in `internal/api/reports.go` | `report` | `export-summary-report` | Missing today |
| `GET /api/v1/reports/migrations` | `writeMigrationsReport` in `internal/api/reports.go` | `report` | `export-migrations-report` | Missing today |
| `GET /api/v1/reports/audit` | `writeAuditReport` in `internal/api/reports.go` | `report` | `export-audit-report` | Missing today |
| Authenticated `403` on tenant route | `RequireTenantRole` and `RequireTenantPermission` in `internal/api/middleware.go` | `authz` | `deny-role` or `deny-permission` | Missing today; requires backend refactor or dependency injection |

## 8. Minimum Viable Audit Log Structure

Viaduct should keep the current `internal/models/audit.go` schema as the base contract and standardize how it is used.

### Base event shape

```json
{
  "id": "evt-123",
  "tenant_id": "tenant-a",
  "actor": "service-account:ops-dashboard",
  "request_id": "req-123",
  "category": "migration",
  "action": "execute",
  "resource": "migration-42",
  "outcome": "success",
  "message": "migration execution started",
  "details": {
    "auth_method": "service-account",
    "role": "operator",
    "spec_name": "wave-1",
    "approved_by": "alice",
    "ticket": "CHG-1234"
  },
  "created_at": "2026-04-08T14:30:00Z"
}
```

### Field intent
- `id`: unique event ID
- `tenant_id`: tenant security boundary
- `actor`: the credential-level identity Viaduct can currently prove
- `request_id`: correlation handle for support and troubleshooting
- `category`: stable event family such as `admin`, `tenant`, `migration`, or `report`
- `action`: stable verb such as `create-tenant`, `execute`, or `export-audit-report`
- `resource`: affected resource identifier when one exists
- `outcome`: `success` or `failure`
- `message`: concise human-readable summary
- `details`: small machine-readable context map
- `created_at`: UTC event timestamp

### Actor conventions
- Use `admin` for platform-admin-key actions.
- Use `tenant:<tenant-id>` for tenant-key actions.
- Use `service-account:<service-account-id>` for service-account actions.
- When human user accounts do not yet exist, do not fake a richer identity in `actor`. Use `details` and service-account metadata for owner labels.

### Recommended detail keys
- `auth_method`
- `role`
- `service_account_name`
- `spec_name`
- `approved_by`
- `ticket`
- `report_name`
- `report_format`
- `route`
- `required_permission`
- `error_code`

### Sample audit events

#### Service-account creation

```json
{
  "category": "tenant",
  "action": "create-service-account",
  "resource": "sa-ops-dashboard",
  "outcome": "success",
  "message": "service account created",
  "details": {
    "role": "operator",
    "service_account_name": "ops-dashboard",
    "auth_method": "tenant-api-key"
  }
}
```

#### Audit report export

```json
{
  "category": "report",
  "action": "export-audit-report",
  "resource": "audit",
  "outcome": "success",
  "message": "audit report exported",
  "details": {
    "report_name": "audit",
    "report_format": "json",
    "auth_method": "service-account"
  }
}
```

#### Permission denial

```json
{
  "category": "authz",
  "action": "deny-permission",
  "resource": "service-accounts",
  "outcome": "failure",
  "message": "tenant principal cannot access \"tenant.manage\"",
  "details": {
    "route": "POST /api/v1/service-accounts",
    "required_permission": "tenant.manage",
    "auth_method": "service-account",
    "role": "operator"
  }
}
```

## 9. How Action Attribution Should Appear In The UI

Viaduct does not need a large security console for v1. It does need attribution in the places operators already use.

### Required UI placement

| Surface | Current status | v1 expectation |
| --- | --- | --- |
| Settings page | Already shows auth method, role, service-account name, and permissions | Keep as the canonical "who am I authenticated as" surface |
| Migration detail and history | Does not yet expose enough attribution | Do not invent new migration API fields first; source initial attribution from `GET /api/v1/audit` matched by `resource == migration_id` |
| Reports page | Can export audit data but does not show much attribution context | Add recent audit history from `GET /api/v1/audit` before creating any separate audit page |
| Error states | API errors already carry request ID | Keep request ID visible in operator-facing failures |

### Minimum migration attribution fields
- action
- actor
- auth method
- timestamp
- request ID
- `approved_by` when supplied
- ticket/change reference when supplied

### UI behavior guidance
- Prefer attribution summaries attached to the existing migration timeline or history rows.
- Do not invent human identities the backend does not know.
- If the actor is a service account, show the service-account name when available and keep the stable ID available for drill-down.
- If the action came from a tenant key, label it clearly as tenant credential usage so it is visible as weaker attribution.
- Do not create client-side synthetic audit records. The UI must render server-returned audit events and request IDs.
- In the current repo, the first viable attribution implementation should extend `web/src/types.ts` and `web/src/api.ts` for audit events, then render those events in `web/src/features/reports/ReportsPage.tsx` and `web/src/components/MigrationHistory.tsx`.

## 10. Security Assumptions And Limits For The Early Product Stage

### Assumptions
- Viaduct is deployed behind TLS termination or a trusted reverse proxy.
- API keys are provisioned and rotated out of band.
- Pilot-grade environments use PostgreSQL, not only the in-memory store.
- Tenant isolation is the main security boundary.
- Operators understand that service-account ownership is the current unit of identity, not a full human user directory.

### Limits
- No SSO or OIDC yet
- No MFA yet
- No session-based browser auth yet
- No custom roles or group mapping
- No immutable audit retention guarantee
- No dedicated secrets manager integration
- No strong human identity proof beyond named credentials
- No strong platform-admin attribution beyond request IDs and deployment logs

### Practical implication
- This model is good enough for early product evaluation and supervised pilot work.
- It is not the final identity and compliance architecture.
- The UI and docs should state that clearly instead of implying a fuller security posture than the product actually has.

## 11. What Is Required For v1 Versus What Can Wait

### Required for v1
- Separate platform admin and tenant credentials
- Service accounts as first-class named credentials
- The current `viewer` / `operator` / `admin` role model
- Explicit service-account permissions as a narrowing control
- Tenant-scoped audit persistence in PostgreSQL-backed environments
- Request IDs on API errors and in audit events
- Audit coverage for all state-changing tenant/admin actions
- Audit coverage for report and audit exports
- Visible current-caller context in the dashboard
- Visible attribution for migration commands in the dashboard, sourced from audit events rather than frontend-only inference

### Can wait until after v1
- OIDC or SSO
- SCIM or group sync
- MFA
- Custom roles
- Per-object permissions
- A new `migration.read` permission or wider read-only migration visibility
- Full audit search/filtering UI
- Immutable retention or external SIEM streaming
- Approval policies tied to organizational identity providers

## 12. Recommended Implementation And Validation Order

### Work package 1: Freeze the product contract in docs and examples
- Keep the current auth headers and three-role model.
- Update operator-facing docs so tenant keys are bootstrap or break-glass credentials, while dashboard and automation guidance prefer service accounts.
- Document `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` as the normal dashboard credential path.
- Treat the local operator session bootstrap as a direct loopback-only local-lab behavior in product messaging.

### Work package 2: Complete backend audit coverage
- Normalize migration audit details in `internal/api/server.go` so `plan`, `execute`, `resume`, and `rollback` use the same detail keys.
- Add server-side export audit events in `internal/api/reports.go`. Do not rely on the dashboard client for attribution.
- Add authenticated `403` audit events for tenant routes. Because `RequireTenantRole` and `RequireTenantPermission` are currently package-level middleware helpers without direct store access, do not bolt on ad hoc logging. Either:
  - refactor those checks into server-bound wrappers with access to `recordAuditEvent`, or
  - inject an audit recorder dependency explicitly.
- Keep unauthenticated `401` failures in request logs and metrics unless a tenant can be safely identified.

### Work package 3: Surface attribution in the existing UI
- Keep `web/src/features/settings/SettingsPage.tsx` as the canonical current-caller context view.
- Add an `AuditEvent` type and API call in `web/src/types.ts` and `web/src/api.ts`.
- Add recent trust-sensitive audit history to `web/src/features/reports/ReportsPage.tsx`.
- Add migration attribution in `web/src/components/MigrationHistory.tsx` by joining server-returned audit events on migration ID, not by inventing client-side attribution.
- Continue showing request IDs in structured API error states.

### Work package 4: Pilot-grade validation
- Validate that audit events survive restart with PostgreSQL.
- Validate that named service accounts produce understandable attribution in the UI and exported audit trail.
- Validate that shared tenant-key usage is visible as weaker attribution.
- Validate that admin-key actions remain clearly documented as weakly attributed.

## 13. Acceptance Criteria And Test Outlines

### Acceptance criteria
- Every trust-sensitive tenant or admin command emits an immediate server-side audit event with `tenant_id`, `actor`, `request_id`, `category`, `action`, `outcome`, and `created_at`.
- Every report export from `/api/v1/reports/*` emits one server-side audit event.
- Authenticated permission denials on trust-sensitive tenant routes emit one server-side audit event.
- `GET /api/v1/tenants/current` remains the source of truth for UI identity context.
- The dashboard shows current auth method, role, and effective permissions.
- Migration execution history shows action attribution without frontend-only inference.
- The first UI attribution pass reads from audit events, not from new ad hoc migration-history fields.
- Local operator session bootstrap is documented as lab behavior, not pilot guidance.

### Test outlines
- Auth middleware tests:
  - tenant key authenticates as tenant admin
  - service-account key authenticates with effective permissions
  - inactive or expired service account is rejected
  - direct loopback-only local operator bootstrap is rejected for proxied or forwarded requests
  - explicit service-account permissions do not bypass role-gated routes
- Authorization tests:
  - viewer cannot call migration-manage routes
  - operator cannot create or rotate service accounts
  - admin can perform tenant-manage routes
- Audit tests:
  - `handleMigrations` emits `migration:plan`
  - `handleMigrationByID` execute/resume/rollback emit canonical actions with request ID
  - `writeSummaryReport`, `writeMigrationsReport`, and `writeAuditReport` emit canonical export actions
  - authenticated `403` emits `authz:deny-role` or `authz:deny-permission`
  - audit events remain tenant-scoped in both memory and PostgreSQL-backed stores
- UI tests:
  - Settings page renders role, auth method, and service-account name from `GET /api/v1/tenants/current`
  - reports surface renders recent trust-sensitive audit events from `GET /api/v1/audit`
  - migration attribution renders actor, action, timestamp, and request ID using audit-event joins

## 14. Lightweight Manual Validation Runbook

1. Create a tenant with the admin key.
2. Create two service accounts:
   - one `operator`
   - one `viewer`
3. Verify `GET /api/v1/tenants/current` reflects the correct role, permissions, and auth method for each credential.
4. Run a migration plan with the operator credential and confirm one `migration:plan` audit event appears.
5. Execute the migration with `approved_by` and `ticket`, then confirm:
   - the command is accepted
   - the audit event includes request ID and approval context
6. Attempt a service-account creation with the operator credential and confirm:
   - the API returns `403`
   - an authenticated authorization-denial audit event is recorded once that gap is implemented
7. Export the audit report and confirm one export audit event appears.
8. Load the dashboard with the operator service-account credential and confirm:
   - Settings shows service-account auth, role, and permissions
   - reports surface shows recent trust-sensitive audit events
   - migration history/detail shows attribution sourced from audit events
   - request IDs are visible on operator-facing failures

## 15. Maintainer Notes

This model is intentionally conservative.

- It does not reset the architecture.
- It does not require replacing the current auth mechanism.
- It does not invent human-user identity before the rest of the product is ready.
- It does not pretend the current shared admin key has strong attribution.
- It does not pretend the current migration APIs already carry action history.
- It does make a clear promise: Viaduct must be able to show who did the important thing, through which credential path, and under which request ID.

That is the first trust bar the product needs to clear.
