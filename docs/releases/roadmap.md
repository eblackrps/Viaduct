# Viaduct v3.x Roadmap

This roadmap tracks planned platform work that was discussed during the v2.7.0 hardening cycle but did not ship in that release. The changelog only records released work; this document holds the forward-looking plan.

## v3.1 Candidates

- OpenTelemetry tracing for HTTP handlers, workspace jobs, connector calls, and store access
- outbound tenant webhooks with signing, retries, and delivery diagnostics
- workspace templates with YAML-defined scaffolds and dashboard previews

## v3.2 Candidates

- scheduled migrations with timezone-aware maintenance windows and persisted scheduler state
- migration dry-run impact reports with persisted comparisons across replans
- approval workflows with auditable N-of-M gate semantics

## v3.3 Candidates

- MFA and WebAuthn enrollment with recovery-code support for dashboard login
- session fingerprint drift auditing and optional re-auth policy
- snapshot retention policies with dry-run planning and background pruning
- incident break-glass elevation with bounded TTL and auditable justification

## Deferred Platform Features

- dashboard-native retention, templates, approvals, and dry-run UX remain roadmap items until implemented
- roadmap prioritization is issue-driven and may shift between minor releases as hardening and operator feedback come in
