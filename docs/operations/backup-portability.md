# Backup Portability Guide

Viaduct includes Veeam discovery and backup portability planning so workload migrations do not silently lose restore coverage.

## What Is Covered
- backup job discovery
- restore point discovery
- repository discovery
- repository compatibility reporting
- portable backup job template generation
- multi-VM portability planning
- repository mapping
- post-create verification runs
- post-migration continuity validation
- backup policy drift detection
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

## Continuity Validation

After portability execution, Viaduct can validate:
- whether the recreated jobs exist on the target side
- whether repository mappings stayed compatible
- whether schedule, retention, enabled state, or protected-VM membership drifted
- whether restore points for the migrated VM have resumed

The continuity report is intended for API, automation, and reporting flows. It is especially useful when migration cutover and backup operations are owned by different teams.

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
- Failed verification runs should block acceptance of the migrated backup status.
- Post-migration policy drift should be treated as a follow-up incident until the recreated jobs match the intended protection policy.
- Rollback cleanup errors must be reviewed before calling the environment stable again.
