# Lab Screenshots

These screenshots are grounded in the current `examples/lab` seed data. They are meant to help evaluators understand what the local workspace flow and exported report should look like at a glance.
They mirror the current dashboard labels, the seeded lab workspace defaults, and the current report/export behavior.
They are also suitable for the root `README.md`, release notes, and lightweight evaluator packets.

## Files

- [Lab workspace flow](lab-workspace-flow.svg)
- [Lab report export](lab-report-export.svg)

## Preview

![Lab workspace flow](lab-workspace-flow.svg)

![Lab report export](lab-report-export.svg)

## Notes

- `lab-workspace-flow.svg` uses the actual fixture names from `examples/lab/kvm/`, the workspace defaults from `examples/lab/pilot-workspace-create.json`, and the local single-user startup path exposed by `viaduct start`.
- `lab-report-export.svg` reflects the current markdown export structure and the blocked-readiness result produced by the deterministic lab simulation flow.
