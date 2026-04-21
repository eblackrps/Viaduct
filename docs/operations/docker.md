# Docker Operations

Viaduct `v3.0.0` treats the signed OCI image as the canonical release artifact.

## Pull And Run

```bash
docker pull ghcr.io/eblackrps/viaduct:3.0.0
docker run --rm \
  --read-only \
  --tmpfs /tmp \
  -v viaduct-state:/var/lib/viaduct \
  -v "$PWD/config:/etc/viaduct:ro" \
  -e VIADUCT_ADMIN_KEY='sha256:<hex>' \
  -p 8080:8080 \
  ghcr.io/eblackrps/viaduct:3.0.0 \
  serve-api --host 0.0.0.0 --config /etc/viaduct/config.yaml --port 8080
```

Writable state must live under `/var/lib/viaduct`. The container is designed to run with a read-only root filesystem plus `--tmpfs /tmp`.

`ghcr.io/eblackrps/viaduct:edge` is published from merges to `main` and is not for production use.

## Verify The Image Signature

```bash
cosign verify ghcr.io/eblackrps/viaduct:3.0.0 \
  --certificate-identity \
  'https://github.com/eblackrps/viaduct/.github/workflows/image.yml@refs/tags/v3.0.0' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

## Verify The SBOM Attestation

```bash
cosign verify-attestation --type spdx ghcr.io/eblackrps/viaduct:3.0.0 \
  --certificate-identity \
  'https://github.com/eblackrps/viaduct/.github/workflows/image.yml@refs/tags/v3.0.0' \
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
