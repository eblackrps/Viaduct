# Release

This document describes the current Docker-canonical packaging and release process for Viaduct.

## Canonical Commands
- `make release-gate`: canonical local release-owner verification for backend, CLI, dashboard lint/format/unit/build checks, certification coverage, soak coverage, packaging, and coverage enforcement
- `make package-release-matrix`: produce secondary native bundles in `dist/` for `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `windows/amd64`
- `make certification-test`: run connector certification fixtures
- `make soak-test`: run the tagged migration soak workflow
- `make plugin-check`: validate plugin manifest compatibility against the host version
- `make contract-check`: verify the published OpenAPI reference still covers the documented operator routes
- `make release-surface-check`: verify the current version, install docs, site copy, and deployment samples still agree
- `make web-e2e-setup`: install the dashboard dependencies plus the Playwright Chromium runtime for local browser smoke
- `make pilot-smoke`: run the root-level evaluator path (`tests/integration` workspace smoke plus the real `viaduct start` browser smoke)
- `make observability-up`: start the local Grafana + Tempo stack used for backend trace validation
- `make observability-validate`: confirm Grafana, Tempo, and a real Viaduct trace are reachable in a local telemetry run
- `go run ./scripts/openapi_generate`: regenerate the checked-in OpenAPI JSON used by `/api/v1/docs/swagger.json`
- `cd web && npm run lint`: enforce dashboard lint and accessibility rules
- `cd web && npm run test`: run dashboard unit tests
- `cd web && npm run e2e`: run dashboard end-to-end coverage
- `cd web && npm run screenshots:readme`: regenerate the README and demo screenshot assets from the seeded fixture runtime

On Windows, `make release-gate` still builds `bin/viaduct.exe`, but it validates the CLI smoke commands through a LocalAppData-staged `go run ./cmd/viaduct` helper because some operator workstations enforce Application Control policies that block freshly built unsigned binaries from direct execution or from `%TEMP%`. The Windows test helpers stage race and coverage artifacts under repo-local cache directories so the canonical gate remains reproducible on locked-down workstations.

`make release-gate` is the authoritative local check. The canonical tag workflow lives in `.github/workflows/image.yml`, which publishes the signed OCI image to GHCR, mirrors it to `docker.io/emb079/viaduct` when Docker Hub secrets are available, attaches SBOM attestations and provenance, runs the image scan, and publishes the secondary native bundles. `.github/workflows/release.yml` is now a failing guard only so old manual-dispatch habits cannot publish competing images, signatures, SBOMs, provenance, Docker Hub mirrors, or GitHub releases under a second workflow identity. CI adds browser end-to-end coverage, a Docker-backed observability smoke that validates Tempo trace ingestion, plus `gosec`, `trivy`, and `actionlint`; those checks are required for merges, but the browser and Docker-backed portions stay outside the local release gate because they depend on extra browser, Docker, or scanner setup.

## Release Checklist
1. Ensure the working tree is in the intended state and public docs are current.
2. Run `make release-gate`.
3. Inspect the generated bundles in `dist/`.
4. Verify `release-manifest.json`, `dependency-manifest.json`, the bundle-local `SHA256SUMS.txt`, and the release-asset `dist/SHA256SUMS`.
5. Smoke-test the packaged binary with `viaduct version`, `viaduct --help`, `viaduct doctor`, and the canonical local start flow (`viaduct start --config <installed-config> --detach --open-browser=false`) against the bundled dashboard assets when they are present.
   When the local environment has browser prerequisites installed, run `make pilot-smoke` as the high-signal evaluator path before tagging.
6. Confirm install docs, quickstarts, upgrade docs, rollback docs, deployment examples, and the pilot workspace guide still match the artifact layout and current auth behavior.
7. Run `make release-surface-check` so the current version, release notes, image tags, Helm/Compose samples, and public-facing docs agree before tagging.
8. Refresh the checked-in README and demo screenshots, then confirm the root README, install docs, quickstarts, and the `site/` landing page all lead with the Docker-first install and verification path for the current release.
9. Verify the plugin manifest check, OpenAPI contract check, and runtime Swagger UI (`/api/v1/docs`) remain aligned.
10. Confirm there are no open release PRs left hanging if the release is being published directly from `main`.
11. Confirm `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` exist as Actions secrets on the Viaduct repo or as inherited organization secrets if Docker Hub mirroring is expected. Secrets stored in another repository are not visible here.
12. Tag and publish only after the verification and smoke checks are clean. The tag workflow should publish the signed OCI image first, mirror the tag to Docker Hub when those secrets are present, attach SPDX plus CycloneDX attestations and provenance, run the image scan, and then attach the `make package-release-matrix` native bundles as alternative assets.
13. If Docker Hub secrets were added after a release tag already existed, backfill the mirror without retagging by running `gh workflow run image.yml --ref main -f mirror_release_tag=vX.Y.Z`.

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
Git tags keep the leading `v`, but native-bundle archive names use the numeric version label: `dist/viaduct_<version>_<goos>_<goarch>.tar.gz`.

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
