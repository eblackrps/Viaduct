# Pre-Session Checklist

Use this before every Step 7 validation session.

## Session Metadata

- Session date:
- Participant ID:
- Moderator:
- Note taker:
- Environment used:
- Data fidelity:
  - `real/pilot-like sanitized`
  - `internally prepared`
  - `fixture/synthetic`

## Participant Fit Check

- Participant is a real infrastructure-minded operator, approver, or technical reviewer.
- Participant has recent VMware exposure.
- Participant has recent migration-planning, approval, or execution context.
- Participant is not primarily a friendly insider, contributor, or generic startup interview volunteer.

## Environment Check

- `Settings` loads successfully.
- Intended tenant is visible.
- Auth mode is visible.
- VMware baseline is present.
- Proxmox baseline is present.
- The same three standardized workload candidates are present in `Inventory`.
- The saved first-wave plan is present in `Migrations`.
- The saved failed or late-stage pilot run is present in `Migrations`.
- `Reports` can export summary, migration, and audit outputs.
- The prepared CLI discovery command is visible and copyable.

## Scenario Lock

- Low-risk candidate is visible in the current UI as `ready` or low risk.
- Medium-risk candidate is visible in the current UI as `needs-review` or medium risk.
- High-risk candidate is visible in the current UI as blocked or high risk.
- Saved plan includes the low-risk and medium-risk candidates.
- Preflight results include both warnings and at least one blocking issue.
- Saved run state includes completed checkpoints followed by a later-phase issue.

## Moderator Guardrails

- Session goal is workflow trust, not roadmap ideation.
- No live destructive migration actions will be run.
- At most one neutral rescue prompt will be used per workflow.
- Discovery will be discussed or demonstrated only if it does not consume the session.
- KVM lab or fixture-backed data will not be represented as live pilot proof.

## Dry Run Check

- Moderator can complete the planned scenario in under 10 minutes without improvising.
- Note taker has the participant note template ready.
- Screen share or recording setup is working.
- Any environment issue discovered here is fixed before the participant joins.

## Go / No-Go

- `Go`
- `Reschedule`
- `Invalidate if needed`

## Issues Found Before Session

- 
- 
- 
