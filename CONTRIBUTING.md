# Contributing

## Before You Start

Viaduct's public-facing materials are part of the product surface. If your change affects operator behavior, update the relevant docs, examples, and release-facing assets in the same change.

## Reporting Bugs

Open a GitHub issue with:
- a clear summary
- your environment
- reproduction steps
- expected behavior
- actual behavior

If logs or screenshots help, include them with secrets removed.

## Requesting Features

Open a GitHub issue that explains the operator or contributor problem first. Narrow, workflow-grounded requests are easier to evaluate than solution-only requests.

## Development Setup

- Go 1.24 or newer
- Node.js 20.19 or newer for the dashboard
- `make` for the standard workflow
- `golangci-lint`

Typical setup:

```bash
go mod tidy
make build
```

For a fuller local check:

```bash
make release-gate
```

## Workflow Expectations

1. Fork the repository.
2. Create a focused branch.
3. Make the smallest coherent set of changes.
4. Update docs, examples, sample configs, and public assets when operator-visible behavior changes.
5. Run the verification commands for the surfaces you touched.
6. Submit a pull request with scope, rationale, compatibility impact, and validation notes.

## Verification Expectations

- Run `go vet ./...`, `make build`, and `cd web && npm run build` for the relevant surfaces.
- Run `go test ./... -v -race` when your workstation can execute the default Go race path directly.
- Prefer `make release-gate` as the canonical verification path for release, packaging, dashboard, migration, tenant, plugin, doc, or public-asset changes. On Windows workstations with Application Control constraints, rely on the repo's helper-backed `make release-gate` path rather than ad hoc command substitutions.
- Run `make release-gate` before asking for review when the change affects release, packaging, dashboard, migrations, tenants, plugins, docs, or public assets.
- Add regression coverage for bug fixes.

## Compatibility And Scope

- Preserve API, state-store, and plugin compatibility where possible.
- Keep tenant boundaries strict and explicit.
- Prefer extending existing packages over introducing parallel abstractions.
- Keep the public maturity language honest. Use ready for technical assessment or more cautious wording unless stronger proof exists.

## Helpful References

- Architecture overview: [docs/architecture.md](docs/architecture.md)
- Codebase map: [docs/reference/codebase-map.md](docs/reference/codebase-map.md)
- Configuration guide: [docs/reference/configuration.md](docs/reference/configuration.md)
- Plugin author guide: [docs/reference/plugin-author-guide.md](docs/reference/plugin-author-guide.md)
- Support and contributor help: [SUPPORT.md](SUPPORT.md)
- Security process: [SECURITY.md](SECURITY.md)
