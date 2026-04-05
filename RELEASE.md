# Release

This document describes the stable packaging and release process for Viaduct.

## Canonical Commands
- `make release-gate`: full verification flow for backend, CLI, dashboard, coverage, and packaging
- `make package-release`: produce a release bundle in `dist/`

## Release Checklist
1. Ensure the working tree is in the intended state and public docs are current.
2. Run `make release-gate`.
3. Inspect the generated bundle in `dist/`.
4. Verify `release-manifest.json` and `SHA256SUMS.txt`.
5. Smoke-test the packaged binary with `viaduct version` and `viaduct --help`.
6. Confirm install docs, upgrade docs, and rollback docs still match the artifact layout.
7. Tag and publish only after the verification and smoke checks are clean.

## Bundle Contents
The release bundle should include:
- CLI binary
- built web assets
- install scripts
- docs
- sample configs
- examples
- manifest and checksums

## Release Notes Guidance
- summarize operator-visible changes
- document compatibility, migration, or upgrade concerns
- call out any connector-specific caveats
- update [CHANGELOG.md](CHANGELOG.md) with notable release information

## Rollback
If a build is not fit to publish, treat it as a release engineering failure and keep working through `make release-gate` until the issues are resolved. If a published release needs to be withdrawn, use [docs/operations/rollback.md](docs/operations/rollback.md) to guide downgrade communication and operator recovery steps.
