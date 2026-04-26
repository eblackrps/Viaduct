# Documentation Index

This directory holds the deeper Viaduct documentation set. The repo-root docs are the public entrypoints; `docs/` carries the more detailed reference, workflows, and scope material behind them.

## Getting Started

- [Installation](getting-started/installation.md)
- [Quickstart](getting-started/quickstart.md)
- [Docker Operations](operations/docker.md)

For v3.2.1, the signed GHCR OCI image is the main packaged install path. The Docker Hub mirror at `docker.io/emb079/viaduct` is published from the same workflow when the required Actions secrets are configured. The current release/install reference lives in [releases/current.md](releases/current.md). The default local contributor path still starts with `viaduct start`, opens the dashboard at `http://127.0.0.1:8080`, and uses the shipped lab fixtures when no local config exists yet.

## Workflows

- [Assessment Workflow](operations/pilot-workspace-flow.md)
- [Migration Operations](operations/migration-operations.md)
- [Backup Portability](operations/backup-portability.md)
- [Multi-Tenancy](operations/multi-tenancy.md)
- [Auth, Role, And Auditability Model](operations/auth-role-audit-model.md)
- [Backend Observability](operations/observability.md)
- [Observability Requirements](operations/observability-requirements.md)
- [Upgrade](operations/upgrade.md)
- [Rollback](operations/rollback.md)

## Demo And Validation

- [Demo Runbook](operations/demo-runbook.md)
- [Demo Presenter Kit](operations/demo/README.md)
- [Validation Kit Templates](operations/validation/README.md)
- [Real User Validation Plan](operations/real-user-validation-plan.md)
- [Ship Readiness Plan](operations/ship-readiness-plan.md)

## Reference And Scope

- [Architecture Overview](architecture.md)
- [Configuration Reference](reference/configuration.md)
- [Support Matrix](reference/support-matrix.md)
- [Troubleshooting](reference/troubleshooting.md)
- [OpenAPI Reference](reference/openapi.yaml)
- Runtime Swagger UI (`/api/v1/docs` on any running Viaduct server)
- [Plugin Author Guide](reference/plugin-author-guide.md)
- [Plugin Certification Guide](reference/plugin-certification.md)
- [Codebase Map](reference/codebase-map.md)
- [Early Product Focus](early-product-focus.md)
- [Initial Use Case Analysis](initial-use-case-analysis.md)
- [V1 Scope Definition](v1-scope.md)
- [Backend Contract Hardening](backend-contract-hardening.md)

## Release And History

- [Release Notes](releases/README.md)
- [Roadmap Archive Index](roadmaps/README.md)

If you are starting from the repository landing page, also see [../README.md](../README.md), [../INSTALL.md](../INSTALL.md), [../QUICKSTART.md](../QUICKSTART.md), [../UPGRADE.md](../UPGRADE.md), and [../RELEASE.md](../RELEASE.md).
