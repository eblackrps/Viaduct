# Release

This document describes the current packaging and release process for Viaduct.

## Canonical Commands
- `make release-gate`: canonical local release-owner verification for backend, CLI, dashboard lint/format/unit/build checks, certification coverage, soak coverage, packaging, and coverage enforcement
- `make package-release-matrix`: produce release bundles in `dist/` for `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64`
- `make certification-test`: run connector certification fixtures
- `make soak-test`: run the tagged migration soak workflow
- `make plugin-check`: validate plugin manifest compatibility against the host version
- `make contract-check`: verify the published OpenAPI reference still covers the documented operator routes
- `go run ./scripts/openapi_generate`: regenerate the checked-in OpenAPI JSON used by `/api/v1/docs/swagger.json`
- `cd web && npm run lint`: enforce dashboard lint and accessibility rules
- `cd web && npm run test`: run dashboard unit tests
- `cd web && npm run e2e`: run dashboard end-to-end coverage
- `cd web && npm run screenshots:readme`: regenerate the README and demo screenshot assets from the seeded fixture runtime

On Windows, `make release-gate` still builds `bin/viaduct.exe`, but it validates the CLI smoke commands through a LocalAppData-staged `go run ./cmd/viaduct` helper because some operator workstations enforce Application Control policies that block freshly built unsigned binaries from direct execution or from `%TEMP%`. The Windows test helpers stage race and coverage artifacts under repo-local cache directories so the canonical gate remains reproducible on locked-down workstations.

`make release-gate` is the authoritative local check, and the tag workflow in `.github/workflows/release.yml` reuses the same packaging path through `make package-release-matrix`. CI adds browser end-to-end coverage plus `gosec` and `trivy`; those checks are required for merges, but they stay outside the local release gate because they depend on extra browser or scanner setup.

## Release Checklist
1. Ensure the working tree is in the intended state and public docs are current.
2. Run `make release-gate`.
3. Inspect the generated bundles in `dist/`.
4. Verify `release-manifest.json`, `dependency-manifest.json`, the bundle-local `SHA256SUMS.txt`, and the release-asset `dist/SHA256SUMS`.
5. Smoke-test the packaged binary with `viaduct version`, `viaduct --help`, `viaduct doctor`, and the canonical local start flow (`viaduct start --config <installed-config> --detach --open-browser=false`) against the bundled dashboard assets when they are present.
6. Confirm install docs, quickstarts, upgrade docs, rollback docs, deployment examples, and the pilot workspace guide still match the artifact layout and current auth behavior.
7. Refresh the checked-in README and demo screenshots, then confirm the root README embeds, release notes entry, changelog entry, and the `site/` latest-release badge and release-notes link resolve to the current release.
8. Verify the plugin manifest check, OpenAPI contract check, and runtime Swagger UI (`/api/v1/docs`) remain aligned.
9. Confirm there are no open release PRs left hanging if the release is being published directly from `main`.
10. Tag and publish only after the verification and smoke checks are clean. The tag workflow should publish the `make package-release-matrix` tarballs, `dist/SHA256SUMS`, the CycloneDX SBOM, signatures, certificates, and the container tags `vX.Y.Z`, `X.Y.Z`, and `latest`.

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
Git tags keep the leading `v`, but bundle archive names use the numeric version label: `dist/viaduct_<version>_<goos>_<goarch>.tar.gz`.

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
- keep the versioned note under [docs/releases/](docs/releases/README.md) as the source-controlled GitHub release body for that tag

## Rollback
If a build is not fit to publish, treat it as a release engineering failure and keep working through `make release-gate` until the issues are resolved. If a published release needs to be withdrawn, use [docs/operations/rollback.md](docs/operations/rollback.md) to guide downgrade communication and operator recovery steps.
