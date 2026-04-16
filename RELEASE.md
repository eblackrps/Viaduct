# Release

This document describes the current packaging and release process for Viaduct.

## Canonical Commands
- `make release-gate`: full verification flow for backend, CLI, dashboard, soak coverage, certification coverage, packaging, and coverage enforcement
- `make package-release-matrix`: produce release bundles in `dist/` for the supported packaging targets
- `make certification-test`: run connector certification fixtures
- `make soak-test`: run the tagged migration soak workflow
- `make plugin-check`: validate plugin manifest compatibility against the host version
- `make contract-check`: verify the published OpenAPI reference still covers the documented operator routes
- `go run ./scripts/openapi_generate`: regenerate the checked-in OpenAPI JSON used by `/api/v1/docs/swagger.json`
- `cd web && npm run lint`: enforce dashboard lint and accessibility rules
- `cd web && npm run test`: run dashboard unit tests
- `cd web && npm run e2e`: run dashboard end-to-end coverage

On Windows, `make release-gate` still builds `bin/viaduct.exe`, but it validates the CLI smoke commands through a LocalAppData-staged `go run ./cmd/viaduct` helper because some operator workstations enforce Application Control policies that block freshly built unsigned binaries from direct execution or from `%TEMP%`. The Windows test helpers stage race and coverage artifacts under repo-local cache directories so the canonical gate remains reproducible on locked-down workstations.

## Release Checklist
1. Ensure the working tree is in the intended state and public docs are current.
2. Run `make release-gate`.
3. Inspect the generated bundles in `dist/`.
4. Verify `release-manifest.json`, `dependency-manifest.json`, and `SHA256SUMS.txt`.
5. Smoke-test the packaged binary with `viaduct version`, `viaduct --help`, `viaduct doctor`, and the canonical local start flow (`viaduct start --config <installed-config> --detach --open-browser=false`) against the bundled dashboard assets when they are present.
6. Confirm install docs, quickstarts, upgrade docs, rollback docs, deployment examples, and the pilot workspace guide still match the artifact layout and current auth behavior.
7. Confirm the root README screenshot embeds, release notes entry, changelog entry, and the `site/` release badge all point at the same release.
8. Verify the plugin manifest check, OpenAPI contract check, and runtime Swagger UI (`/api/v1/docs`) remain aligned.
9. Confirm there are no open release PRs left hanging if the release is being published directly from `main`.
10. Tag and publish only after the verification and smoke checks are clean, and attach the generated release bundles plus checksums to the GitHub release.

## Bundle Contents
The release bundle should include:
- CLI binary
- built web assets
- install scripts
- docs
- sample configs
- examples
- release manifest, dependency manifest, and checksums
- deployment reference assets

The standalone public site under [`site/`](site/README.md) is published through GitHub Pages and is not bundled into the tagged release artifacts.

## Release Notes Guidance
- summarize operator-visible changes
- document compatibility, migration, or upgrade concerns
- call out any connector-specific caveats
- include the workspace-first operator flow, runtime-auth behavior, and local startup changes when they are part of the release
- include current screenshot assets when the dashboard experience changed materially
- note whether `/api/v1` remained backward compatible and whether any new `/api/v2` routes were added for newer clients
- include any new security headers, authentication defaults, or environment variables that operators need to understand before rollout
- use absolute GitHub URLs in the published GitHub release body when relative asset links would be ambiguous
- update [CHANGELOG.md](CHANGELOG.md) with notable release information

## Rollback
If a build is not fit to publish, treat it as a release engineering failure and keep working through `make release-gate` until the issues are resolved. If a published release needs to be withdrawn, use [docs/operations/rollback.md](docs/operations/rollback.md) to guide downgrade communication and operator recovery steps.
