# Current Release

Viaduct `v3.2.1` is the current published release.

This page is the repo-local source of truth for the current release/install story. Use it to sanity-check the root docs, deployment samples, screenshots, and public site before the next version is prepared.

The only tag-publishing workflow is [`.github/workflows/image.yml`](../../.github/workflows/image.yml). [`.github/workflows/release.yml`](../../.github/workflows/release.yml) is a guard-only workflow and must not publish competing release assets.

## Primary Install

Use the signed GHCR OCI image as the primary packaged install surface:

```bash
docker pull ghcr.io/eblackrps/viaduct:3.2.1
cosign verify ghcr.io/eblackrps/viaduct:3.2.1 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.2.1' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

The Docker Hub mirror for the same release is `docker.io/emb079/viaduct:3.2.1` when the repository Docker Hub secrets are configured.

## Operator Path

- Local evaluation: run `viaduct start`, open `http://127.0.0.1:8080`, choose `Start local session`, create a workspace, discover, inspect, simulate, save a plan, and export a report.
- Shared or packaged deployments: prefer service account keys for the dashboard Get started flow. Tenant keys remain supported for setup and recovery.
- Runtime auth remains server-backed: the browser keeps only a session marker after sign-in.
- Observability validation: optionally run `make observability-up` and `make observability-validate` to confirm the backend trace path into Grafana + Tempo before a pilot or release review.

## Release References

- Versioned release note: [docs/releases/v3.2.1.md](v3.2.1.md)
- Changelog stream: [CHANGELOG.md](../../CHANGELOG.md)
- Install guide: [INSTALL.md](../../INSTALL.md)
- Quickstart: [QUICKSTART.md](../../QUICKSTART.md)
- Detailed quickstart: [docs/getting-started/quickstart.md](../getting-started/quickstart.md)
- Docker operations: [docs/operations/docker.md](../operations/docker.md)
