# Ship Readiness Record

This record summarizes the release-readiness work prepared for `v3.2.1`. It is intentionally limited to verified repository surfaces: release metadata, website copy, deployment defaults, runtime readiness, screenshots, and validation commands.

## Completed

- Active install, release, website, Compose, Helm, package metadata, and release-note surfaces are aligned on `v3.2.1`.
- `make release-surface-check` fails when active docs, website files, image tags, or release links drift from the dashboard package version.
- `make site-check` validates the static site source, local assets, Pages workflow directory, and active release version strings.
- `scripts/release_acceptance` can validate a published image by pulling it, verifying cosign identity, running it against PostgreSQL in production mode, and checking health/readiness/about plus tenant auth.
- Production mode refuses memory-only state, missing auth configuration, missing lifecycle policies, and missing dashboard assets before serving.
- `/readyz` reports store diagnostics, schema state, lifecycle policy loading, auth configuration, connector circuit state, dashboard assets, and production mode.
- The Docker image now includes built dashboard assets and default config/policy inputs needed by the packaged readiness path.

## Release Owner Checks

Run the release checklist in [RELEASE.md](../../RELEASE.md). The strongest local path remains:

```bash
make release-gate
```

When Docker, cosign, and network access are available, validate the published image after the tag workflow produces it:

```bash
go run ./scripts/release_acceptance \
  -image ghcr.io/eblackrps/viaduct:3.2.1 \
  -certificate-identity 'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.2.1'
```

After GitHub Pages deploys, verify the public site artifact:

```bash
go run ./scripts/site_validate -base-url https://viaducthq.com
```

## Boundaries

- Fixture-backed connector tests do not prove production-pilot coverage for every connector pair.
- The local KVM lab is the first-run and demo path, not evidence of live production certification.
- Production deployments should use PostgreSQL, hashed admin-key storage, service accounts for routine automation, same-origin CORS by default, and TLS at a reverse proxy or ingress.
