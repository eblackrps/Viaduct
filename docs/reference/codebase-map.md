# Codebase Map

This map is intended to help new contributors and operators find the parts of the repository that matter most.

## Root
- `cmd/viaduct/`: Cobra CLI entrypoints and shared CLI helpers
- `configs/`: sample migration specs, cost profiles, and policies
- `docs/`: operator, reference, and architecture documentation
- `examples/`: lab assets and plugin examples
- `internal/`: core implementation packages
- `tests/integration/`: end-to-end and integration coverage
- `web/`: React dashboard

## `internal/`
- `internal/api/`: REST API server and auth middleware
- `internal/connectors/`: built-in connectors and plugin host
- `internal/deps/`: dependency graph generation
- `internal/discovery/`: orchestrated discovery engine and diffing
- `internal/lifecycle/`: cost, policy, drift, remediation, and simulation logic
- `internal/migrate/`: spec parsing, planning, preflight, orchestration, warm migration, verification, and rollback
- `internal/models/`: universal schema and shared domain models
- `internal/store/`: in-memory and PostgreSQL persistence

## High-Leverage Files
- `AGENTS.md`: contributor rules, patterns, and gotchas
- `Makefile`: canonical build, test, release-gate, and package targets
- `.github/workflows/ci.yml`: CI and release bundle generation
- `scripts/package_release.go`: release bundle packager
- `internal/api/server.go`: primary backend integration point
- `internal/migrate/orchestrator.go`: cold migration state machine
- `internal/migrate/replication.go`: warm migration replication state
- `internal/connectors/plugin/host.go`: plugin lifecycle and safety rules

## Example And Demo Assets
- `examples/lab/`: local evaluation lab, config, and API payload examples
- `examples/plugin-example/`: minimal plugin reference implementation
