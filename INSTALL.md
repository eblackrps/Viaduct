# Installation

This is the top-level installation entrypoint for Viaduct. Use it with [QUICKSTART.md](QUICKSTART.md) for the fastest browser-first path, see [docs/getting-started/installation.md](docs/getting-started/installation.md) for the deeper walkthrough, and use [docs/releases/current.md](docs/releases/current.md) as the repo-local source of truth for the current release/install story.

## Requirements

- Go 1.26.0 or newer
- Node.js 20.19 or newer if you want to build the dashboard locally; CI and release packaging currently pin Node.js 20.20.x
- `make` for the standard workflow
- `golangci-lint` for local linting

## Primary Install From Docker

After the `v3.2.1` tag workflow publishes, Viaduct uses the signed OCI image as the primary packaged install path.

```bash
docker pull ghcr.io/eblackrps/viaduct:3.2.1
```

Verify the image before rollout:

```bash
cosign verify ghcr.io/eblackrps/viaduct:3.2.1 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.2.1' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

The primary signed registry is `ghcr.io/eblackrps/viaduct`. The Docker Hub mirror is `docker.io/emb079/viaduct:3.2.1` when repository Docker Hub secrets are configured.

GitHub Actions mirrors release tags plus `main` branch `:edge` and `:sha-*` image tags to Docker Hub whenever `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured for this repo or exposed through organization-level Actions secrets. For persistent deployments, follow the PostgreSQL-backed container guidance in [docs/operations/docker.md](docs/operations/docker.md), the Compose sample in [deploy/docker-compose.prod.yml](deploy/docker-compose.prod.yml), or the Helm chart defaults in [deploy/helm/viaduct](deploy/helm/viaduct).

## Build From Source

```bash
git clone https://github.com/eblackrps/Viaduct.git
cd viaduct
go mod tidy
make build
make web-build
./bin/viaduct version
```

Start the local operator runtime:

```bash
./bin/viaduct start
```

On a fresh source checkout, `viaduct start` generates the default local lab config when `~/.viaduct/config.yaml` is missing, serves the built dashboard and API together, and prints the WebUI URL. The browser-first flow starts on the Get started screen, where the default local lab path offers `Start local session` instead of requiring a pasted key.

The default local URL is [http://127.0.0.1:8080](http://127.0.0.1:8080).
The same runtime also serves live API docs at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

This is the contributor and local-lab path. For packaged environments, prefer the signed OCI image above.

If you need the browser-side evaluator smoke from the same checkout, run `make web-e2e-setup` once and then `make pilot-smoke`.

## Native Release Bundle Alternative

Tagged native bundles are attached to [GitHub Releases](https://github.com/eblackrps/Viaduct/releases) as an alternative path for operators who cannot run containers. The same layout can be generated locally through `make package-release-matrix`.

Release bundles produced by `make package-release-matrix` include:
- CLI binary
- built web assets
- docs and sample configs
- install scripts
- deployment reference assets
- checksums, a release manifest, and a dependency manifest

On POSIX systems:

```bash
PREFIX=/usr/local ./install.sh ./bin/viaduct ./web ./configs ./examples ./docs
```

On Windows PowerShell:

```powershell
.\install.ps1 -SourceBin .\bin\viaduct.exe -SourceWeb .\web -SourceConfigs .\configs -SourceExamples .\examples -SourceDocs .\docs -Prefix "$env:LOCALAPPDATA\\Viaduct"
```

Both install scripts copy the CLI, built dashboard assets, docs, configs, and examples into one predictable layout. They also create a starter config that points at the installed lab fixtures when no config already exists at the install location.

After install, start Viaduct with the config path printed by the installer. For example:

```bash
viaduct start --config /usr/local/etc/viaduct/config.yaml
```

On Windows, the equivalent installed command is typically:

```powershell
viaduct.exe start --config "$env:LOCALAPPDATA\Viaduct\etc\viaduct\config.yaml"
```

The default local URL remains [http://127.0.0.1:8080](http://127.0.0.1:8080).

## Verify The Install

```bash
viaduct version
viaduct --help
viaduct doctor
```

When built dashboard assets are present, Viaduct serves the dashboard shell at `/` and the API under `/api/v1/`.

## Recommended Next Step

The cleanest evaluation path is still the local lab in [examples/lab](examples/lab). Continue with [QUICKSTART.md](QUICKSTART.md).

For packaged or persistent evaluation environments:
- use PostgreSQL instead of the in-memory store
- prefer service account keys for normal operator access
- keep `VIADUCT_ALLOWED_ORIGINS` empty for same-origin deployments; set it only when the dashboard is served from a different origin than the API
- set `VIADUCT_WEB_DIR` only when you keep built dashboard assets in a non-standard location
- tune `VIADUCT_WORKSPACE_JOB_TIMEOUT` if discovery or planning jobs need a different server-side timeout budget
- tune `VIADUCT_WORKSPACE_JOB_CONCURRENCY` if a packaged deployment needs fewer or more concurrent workspace workers than the default bounded pool
- keep the dashboard runtime auth flow on its default server-backed session path unless you have a reason to pre-seed browser credentials in development
- keep the Vite dev server out of any shared or internet-facing environment
- use `viaduct serve-api` directly for service, container, or intentionally headless deployments, keeping its default loopback bind unless you have configured API credentials
- do not use `VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE=true` outside disposable break-glass scenarios
