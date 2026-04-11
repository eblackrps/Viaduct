# Support

Viaduct is maintained as an open-source project with best-effort community support.

## Start Here

- Project overview and status: [README.md](README.md)
- Install and first run: [INSTALL.md](INSTALL.md), [QUICKSTART.md](QUICKSTART.md)
- Full docs map: [docs/README.md](docs/README.md)
- Local runtime health: `viaduct status --runtime`, `viaduct doctor`
- Troubleshooting: [docs/reference/troubleshooting.md](docs/reference/troubleshooting.md)
- Support matrix: [docs/reference/support-matrix.md](docs/reference/support-matrix.md)

## How To Ask For Help

- Installation, usage, or evaluation questions: open a GitHub issue and describe what you are trying to do, your environment, and where you got blocked.
- Suspected defects: use the bug report template and include reproduction steps, expected behavior, actual behavior, version or commit, and relevant logs with secrets removed.
- Feature requests: describe the operator problem first, then the change you think would help.

## Useful Context To Include

- `viaduct version` output or commit SHA
- `request_id` from the UI or API when available
- `migration_id`, `workspace_id`, `job_id`, or `snapshot_id` when relevant
- OS and architecture
- Go and Node versions if you built from source
- connector or platform involved
- whether you are using the in-memory or PostgreSQL store
- whether the issue reproduces in `make release-gate`, `make certification-test`, or `make soak-test`
- relevant config snippets with secrets removed

## Expectations

- Support is best effort and prioritized by severity, reproducibility, and operator impact.
- Clear repro steps and correlated identifiers materially improve turnaround time.
- The fastest path to a good answer is usually a narrow question grounded in the current documented workflow.

## Security

Do not use public issues for security reports. Follow [SECURITY.md](SECURITY.md) instead.
