# Demo State Checklist

Use this with [demo-scenario-brief.md](demo-scenario-brief.md) before any serious Viaduct demo.

## Environment

- Persistent store is enabled.
- The dashboard is already running and responsive.
- A named tenant is loaded, for example `vmware-exit-demo`.
- A named dashboard credential is configured, preferably a service account.
- Unrelated browser tabs, notifications, and terminal noise are closed.

## Inventory Baseline

- VMware source baseline is present.
- Proxmox target baseline is present.
- Baseline timestamps look recent and believable.
- Non-zero status counts are visible for `ready`, `needs review`, and `blocked`.
- Discovery context is visible without extra clicks.

## Standardized Workloads

- `web-portal-01` is visible as a ready / low-risk candidate.
- `orders-app-02` is visible as a needs-review / medium-risk candidate.
- `sql-legacy-01` is visible as blocked or high risk.

## Saved Plan And Preflight

- A saved first-wave plan exists for `web-portal-01` and `orders-app-02`.
- Preflight shows exactly 1 visible blocker.
- Preflight shows at least 2 visible warnings.
- Wave output or runbook output is visible.

## Saved Pilot State

- A saved failed run exists.
- The run phase is `failed`.
- Earlier checkpoints are completed through `configure`.
- `verify` is visibly failed.
- Per-workload state is visible.
- One clear per-workload failure reason is visible and plausible.

## Reports

- Summary export works.
- Migrations export works.
- Audit export works.
- Migration history is populated.
- Discovery snapshots are populated.

## Presenter Readiness

- You can move through the full 3-minute flow without improvising.
- You can move through the full 15-minute flow without improvising.
- You know the exact click path between `Inventory`, `Migrations`, `Reports`, and `Settings`.
- You can answer whether the data is sanitized real, internally prepared, or fixture-backed.
- You are ready to say early that discovery is CLI-first and execution is framed as supervised pilot control.

## Fallbacks

- Backup browser window or screenshots are ready for each main page.
- One migration ID is easy to reference.
- One export file is already downloaded in case export latency appears.
- The CLI discovery command is visible even if you do not run it.
