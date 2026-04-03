# Phase 0 Roadmap

Status: active

Phase 0 establishes the public repository foundation required to build the platform in the open.

## Goals
- create a buildable Go module and repository layout
- define the universal inventory schema and connector interface
- scaffold the CLI surface for discovery and migration workflows
- add CI, contributor guidance, and project governance files

## Deliverables
- `cmd/viaduct/` Cobra CLI skeleton
- `internal/models/` universal schema
- `internal/connectors/` interface, registry, and connector stubs
- CI workflow, lint config, and Makefile
- README, contributing guide, roadmap, and security/community files

## Exit Criteria
- repository builds cleanly
- tests and vet checks pass
- contributor expectations are documented
- the repo is ready for Phase 1 implementation work
