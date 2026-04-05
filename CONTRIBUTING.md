# Contributing

## How To Report Bugs
Open a GitHub issue with a clear summary, the environment you were using, reproduction steps, expected behavior, and the actual behavior you observed. If logs or screenshots help clarify the problem, include them with any secrets removed. Use the bug-report template when possible so maintainers get the minimum triage context up front.

## How To Request Features
Open a GitHub issue describing the problem you are trying to solve, why existing behavior is not enough, and what a successful outcome would look like. Feature requests with operational context are easier to prioritize than solution-only requests.

## Development Setup
- Go 1.24 or newer
- Node.js 20 or newer for the dashboard
- `make` (preferred, optional on Windows if you run the underlying commands directly)
- `golangci-lint`

Clone the repository, install dependencies with `go mod tidy`, and use the Make targets for the standard workflow. The quickest release-candidate validation path is:

```bash
go mod tidy
make release-gate
```

If you use Codex for repo tasks, the bootstrap script in `.codex/setup.sh` installs the expected linter version in supported environments.

## Workflow
1. Fork the repository.
2. Create a feature branch for your work.
3. Make your changes with focused commits and update docs, examples, and sample configs when public behavior changes.
4. Run `go test ./... -v -race`, `go vet ./...`, `make build`, and `cd web && npm run build` for the surfaces you touched.
5. Run `make release-gate` before asking for review when your change affects release, packaging, dashboard, migrations, tenant behavior, plugins, or docs.
6. Submit a pull request with context about the problem, solution, compatibility impact, and verification.

## Compatibility Expectations
- Preserve API, state-store, and plugin compatibility where possible.
- Document any migration requirement or operator-facing behavior change in the PR and relevant docs.
- Keep tenant boundaries strict and explicit in tests and implementation.
- Add regression coverage for every bug fix.

## Documentation And Examples
- Top-level docs such as `README.md`, `INSTALL.md`, `QUICKSTART.md`, `UPGRADE.md`, `RELEASE.md`, `SUPPORT.md`, and `SECURITY.md` are part of the public product surface.
- Keep detailed docs under `docs/` aligned with the public entrypoint docs at the repo root.
- Historical phase documents under `docs/roadmaps/` are archives. Update them only to clarify shipped outcomes, not to describe current active work.
- Example configs and lab assets should remain runnable and parseable. Prefer realistic examples over pseudo-configuration.

## Testing Expectations
- Table-driven Go tests are the default.
- Integration tests live under `tests/integration/`.
- Release and packaging changes should include validation for examples, install flows, or generated artifacts where practical.
- If you touch dashboard code, run `cd web && npm run build`.

## Codebase Orientation
- Architecture overview: [docs/architecture.md](docs/architecture.md)
- Codebase map: [docs/reference/codebase-map.md](docs/reference/codebase-map.md)
- Configuration guide: [docs/reference/configuration.md](docs/reference/configuration.md)
- Plugin author guide: [docs/reference/plugin-author-guide.md](docs/reference/plugin-author-guide.md)

## Support
For usage questions, evaluation help, or contribution process questions, start with [SUPPORT.md](SUPPORT.md).

## Security
If you believe you have found a security issue, follow the process in [SECURITY.md](SECURITY.md) instead of opening a public bug report.
