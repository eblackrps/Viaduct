# Docker Operations

Viaduct v3.3.0 uses the signed GHCR OCI image as the primary packaged release artifact.

## Registries

- Primary signed registry: `ghcr.io/eblackrps/viaduct`
- Docker Hub mirror: `docker.io/emb079/viaduct`

GitHub Actions mirrors release tags plus `main` branch `:edge` and `:sha-*` image tags to Docker Hub whenever `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured for the Viaduct repo or inherited from an organization-level Actions secret.

If those secrets are added after a GitHub release tag already exists, release owners can backfill the Docker Hub semver tags from the current workflow definition without retagging the repo:

```bash
gh workflow run image.yml --ref main -f mirror_release_tag=v3.3.0
```

## Local Evaluation

For the local lab, run Compose from the repo root:

```bash
docker compose up -d --build
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). This path requires no tenant key, service-account key, admin key, or copied secret file. It starts PostgreSQL with a trust-only internal Docker connection, runs Viaduct in local-runtime mode, and uses the bundled KVM fixtures for the default assessment flow.

The root `compose.yaml` publishes only `127.0.0.1:${VIADUCT_PORT:-8080}` on the host. Do not reuse it for a remotely reachable shared environment.

Stop it with:

```bash
docker compose down
```

Use `docker compose down -v` only when you intentionally want to delete local state.

## Production Pull And Run

For shared or production deployments, use PostgreSQL and set `VIADUCT_ENVIRONMENT=production`. The Compose sample in `deploy/docker-compose.prod.yml` starts PostgreSQL and Viaduct together, reads the admin-key hash and database password from environment variables, and uses `/readyz` for the container health check.

```bash
docker pull ghcr.io/eblackrps/viaduct:3.3.0
mkdir -p config
cp configs/config.example.yaml config/config.yaml
# edit config/config.yaml before starting the service
export VIADUCT_ADMIN_KEY='sha256:<hex>'
export POSTGRES_PASSWORD='<database-password>'
docker compose -f deploy/docker-compose.prod.yml up -d
```

The Compose file mounts `./config` to `/etc/viaduct:ro`, so `config/config.yaml`
becomes `/etc/viaduct/config.yaml` inside the Viaduct container. Keep secrets in
environment variables or your deployment secret manager rather than committing
them to the copied config file.

`/healthz` reports that the HTTP process is alive. `/readyz` is the production
readiness endpoint and checks the store, schema state, policy loading, auth
configuration, connector circuit state, dashboard assets, and production mode.
Use `/readyz` for Compose, Kubernetes, and load balancer readiness checks.

For a single-container evaluation run outside Compose, omit `VIADUCT_ENVIRONMENT=production` and keep the listener bound to loopback unless credentials are configured:

```bash
docker run --rm \
  --read-only \
  --tmpfs /tmp \
  -v viaduct-state:/var/lib/viaduct \
  -v "$PWD/config:/etc/viaduct:ro" \
  -e VIADUCT_ADMIN_KEY='sha256:<hex>' \
  -p 127.0.0.1:8080:8080 \
  ghcr.io/eblackrps/viaduct:3.3.0 \
  serve-api --host 127.0.0.1 --config /etc/viaduct/config.yaml --port 8080
```

Writable fallback files such as audit retry logs live under `/var/lib/viaduct`. The container is designed to run with a read-only root filesystem plus `--tmpfs /tmp`.

`ghcr.io/eblackrps/viaduct:edge` is published from merges to `main` and is not for production use.

If GHCR access is restricted in your environment and the mirror tag has been published, you can pull the mirrored image instead:

```bash
docker pull docker.io/emb079/viaduct:3.3.0
```

Treat GHCR as the verification source even when you deploy from the Docker Hub mirror.

## Verify The Image Signature

```bash
cosign verify ghcr.io/eblackrps/viaduct:3.3.0 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.3.0' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

## Verify The SBOM Attestation

```bash
cosign verify-attestation --type spdx ghcr.io/eblackrps/viaduct:3.3.0 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.3.0' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

## Consume The SBOM

Download the SPDX or CycloneDX attestation payload from the GitHub Release or inspect the attestation directly after verification. Feed the attested SBOM into your registry scanner, admission controller, or software-supply-chain inventory tooling instead of rebuilding a private SBOM from the unpacked image.

## Optional Backend Observability

The production container and Helm defaults now expose opt-in OpenTelemetry environment variables so packaged environments can export traces without patching the image. The default remains safe and lightweight: if you leave observability disabled, Viaduct still runs normally with its built-in `/metrics` and readiness surfaces only.

## Production Safety Notes

- Set `VIADUCT_ENVIRONMENT=production` for persistent deployments.
- Configure PostgreSQL with `state_store_dsn` or `VIADUCT_STATE_STORE_DSN`; production mode refuses the in-memory store.
- Configure `VIADUCT_ADMIN_KEY` as `sha256:<hex>` and use service account keys for normal dashboard automation.
- Keep `VIADUCT_ALLOWED_ORIGINS` empty for same-origin deployments. Add only explicit trusted origins when the dashboard and API are served from different origins.
- Terminate TLS at a reverse proxy or ingress and configure trusted proxy CIDRs with `VIADUCT_TRUSTED_PROXIES` before relying on forwarded protocol headers.
- Do not use `VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE=true` in production mode; production startup ignores the dangerous override.

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
