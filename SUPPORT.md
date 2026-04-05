# Support

Viaduct is maintained as an open source project with best-effort community support.

## Where To Start
- Read [README.md](README.md) for the current project status and capability summary.
- Use [INSTALL.md](INSTALL.md) and [QUICKSTART.md](QUICKSTART.md) for first-run setup.
- Use [docs/README.md](docs/README.md) for the full documentation map.
- Use [docs/reference/troubleshooting.md](docs/reference/troubleshooting.md) for common failure modes and recovery steps.

## How To Get Help
- Installation, usage, or evaluation questions: open a GitHub issue and describe what you are trying to do, what environment you are using, and where you got blocked.
- Suspected defects: use the bug report template and include reproduction steps, expected behavior, actual behavior, version or commit, and relevant logs with secrets removed.
- Feature requests: use the feature request template and describe the operator or contributor problem you are trying to solve.

## What To Include
- Viaduct version or commit SHA from `viaduct version`
- OS and architecture
- Go and Node versions if building from source
- connector or platform involved
- whether you are using the in-memory or PostgreSQL store
- whether the issue appears in `make release-gate`, `make certification-test`, or `make soak-test`
- relevant config snippets with secrets removed

## Security
Do not use public issues for security reports. Follow [SECURITY.md](SECURITY.md) instead.

## Support Expectations
- Support is best effort and prioritized by severity, reproducibility, and operator impact.
- Clear repro steps and logs materially improve turnaround time.
- Compatibility questions are easiest to answer when they reference the [docs/reference/support-matrix.md](docs/reference/support-matrix.md).
