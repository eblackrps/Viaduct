# Backup Portability Guide

Viaduct includes Veeam discovery and backup portability planning so workload migrations do not silently lose restore coverage.

## What Is Covered
- backup job discovery
- restore point discovery
- repository discovery
- portable backup job template generation
- repository mapping
- post-create verification runs
- rollback cleanup for created jobs

## Current Operating Model

The Veeam portability flow is implemented in `internal/connectors/veeam/portability.go` and is currently intended for backend, integration, and automation use. There is not yet a dedicated top-level CLI command for backup job portability.

## Portability Planning Inputs
- source VM identity
- target VM identity
- repository mappings
- target-side repository availability

The portability planner rewrites protected object names toward the target workload and warns when the requested target repository is missing.

## Execution Behavior
- create target backup jobs
- launch verification runs for each created job
- collect created job IDs and verification state
- return actionable errors if any job create or verification step fails

## Rollback Behavior

On migration rollback, Viaduct can remove portable backup jobs it created during the portability run. Partial cleanup failures are surfaced as explicit errors and should be treated as operator follow-up work.

## Recommended Operator Process
1. Discover source backups and repositories.
2. Define repository mappings for the target environment.
3. Run portability planning in a lab or pilot scope first.
4. Verify created jobs and restore-point continuity after migration.
5. Capture exceptions or repository mismatches as change-management notes.

## Recovery Expectations
- Missing repositories should be treated as blocking portability warnings.
- Failed verification runs should block acceptance of the migrated backup posture.
- Rollback cleanup errors must be reviewed before calling the environment stable again.
