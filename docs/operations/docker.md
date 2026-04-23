# Docker Operations

Viaduct `v3.1.1` treats the signed OCI image as the canonical release artifact.

## Registries

- Canonical signed registry: `ghcr.io/eblackrps/viaduct`
- Docker Hub mirror: `docker.io/emb079/viaduct`

GitHub Actions mirrors release tags plus `main` branch `:edge` and `:sha-*` image tags to Docker Hub whenever `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured for the Viaduct repo or inherited from an organization-level Actions secret.

If those secrets are added after a GitHub release tag already exists, release owners can backfill the Docker Hub semver tags from the current workflow definition without retagging the repo:

```bash
gh workflow run image.yml --ref main -f mirror_release_tag=v3.1.1
```

## Pull And Run

```bash
docker pull ghcr.io/eblackrps/viaduct:3.1.1
docker run --rm \
  --read-only \
  --tmpfs /tmp \
  -v viaduct-state:/var/lib/viaduct \
  -v "$PWD/config:/etc/viaduct:ro" \
  -e VIADUCT_ADMIN_KEY='sha256:<hex>' \
  -p 8080:8080 \
  ghcr.io/eblackrps/viaduct:3.1.1 \
  serve-api --host 0.0.0.0 --config /etc/viaduct/config.yaml --port 8080
```

Writable state must live under `/var/lib/viaduct`. The container is designed to run with a read-only root filesystem plus `--tmpfs /tmp`.

`ghcr.io/eblackrps/viaduct:edge` is published from merges to `main` and is not for production use.

If GHCR access is restricted in your environment, you can pull the mirrored image instead:

```bash
docker pull docker.io/emb079/viaduct:3.1.1
```

Treat GHCR as the verification source even when you deploy from the Docker Hub mirror.

## Verify The Image Signature

```bash
cosign verify ghcr.io/eblackrps/viaduct:3.1.1 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.1.1' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

## Verify The SBOM Attestation

```bash
cosign verify-attestation --type spdx ghcr.io/eblackrps/viaduct:3.1.1 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.1.1' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

## Consume The SBOM

Download the SPDX or CycloneDX attestation payload from the GitHub Release or inspect the attestation directly after verification. Feed the attested SBOM into your registry scanner, admission controller, or software-supply-chain inventory tooling instead of rebuilding a private SBOM from the unpacked image.

## Upgrade Guidance

1. Pull the new immutable semver tag.
2. Verify the cosign signature and SBOM attestation.
3. Replace the running container with the new tag while preserving the mounted state volume and config mount.
4. Keep `:latest` for evaluation only; pin semver tags in production.
5. If you rely on the Docker Hub mirror, confirm the Viaduct repo still has `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` configured before the release tag is cut.

## Deployment References

- Compose sample: [../../deploy/docker-compose.prod.yml](../../deploy/docker-compose.prod.yml)
- Helm chart: [../../deploy/helm/viaduct](../../deploy/helm/viaduct)
- Release cadence and `:edge` policy: [../releases/cadence.md](../releases/cadence.md)
