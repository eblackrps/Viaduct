# Installation

Viaduct can be evaluated from the signed OCI image, from source, or from a native release bundle when containers are not an option. The OCI image is the canonical install surface for packaged environments.

## Canonical OCI Install

```bash
docker pull ghcr.io/eblackrps/viaduct:3.1.0
cosign verify ghcr.io/eblackrps/viaduct:3.1.0 \
  --certificate-identity \
  'https://github.com/eblackrps/Viaduct/.github/workflows/image.yml@refs/tags/v3.1.0' \
  --certificate-oidc-issuer \
  'https://token.actions.githubusercontent.com'
```

The signed canonical registry is `ghcr.io/eblackrps/viaduct`. The Docker Hub mirror is `docker.io/emb079/viaduct:3.1.0`.

GitHub Actions mirrors release tags plus `main` branch `:edge` and `:sha-*` tags to Docker Hub whenever `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured for this repo or inherited from an organization secret scope. For runtime flags, upgrade guidance, SBOM verification, the production compose sample, and the Helm chart defaults, continue with [../operations/docker.md](../operations/docker.md), [../../deploy/docker-compose.prod.yml](../../deploy/docker-compose.prod.yml), and [../../deploy/helm/viaduct](../../deploy/helm/viaduct).

## Source Build Alternative

### Prerequisites
- Go 1.25.9+
- Node.js 20.19+ for the dashboard (`20.20.x` is what CI and release packaging currently pin)
- `make` for the standard build flow
- `qemu-img` if you want live disk-conversion execution outside tests

### Build The CLI

```bash
go mod tidy
make build
make web-build
./bin/viaduct version
./bin/viaduct start
```

On a fresh source checkout, `viaduct start` creates the default local lab config when it is missing, points it at `examples/lab/kvm`, and serves the WebUI and API together at `http://127.0.0.1:8080`.
The browser-first flow starts on the Get started screen, where the default local lab path offers a direct loopback-only `Start local session` action instead of requiring a pasted key.
The same runtime also serves live operator API docs at `http://127.0.0.1:8080/api/v1/docs`.

## Native Release Bundle Alternative

Tagged native bundles are attached to [GitHub Releases](https://github.com/eblackrps/Viaduct/releases) as an alternative path. The same structure can also be generated locally through `make package-release-matrix`.

Generate a self-contained local bundle:

```bash
make package-release-matrix
```

This creates:
- `dist/viaduct_<version>_<goos>_<goarch>/`
- `dist/viaduct_<version>_<goos>_<goarch>.tar.gz`
- `dist/SHA256SUMS`

The canonical packaging matrix is:
- `linux/amd64`
- `linux/arm64`
- `darwin/arm64`
- `windows/amd64`

Git tags keep the leading `v`, but bundle names use the numeric release label. For example, `v3.1.0` publishes `dist/viaduct_3.1.0_linux_amd64.tar.gz`.

Each bundle includes:
- the Viaduct CLI binary
- built web assets
- install scripts
- docs, configs, and examples
- `release-manifest.json`
- `dependency-manifest.json`
- `SHA256SUMS`

## Local Container Build

Build the container image locally when you want to validate the Docker path from source:

```bash
make docker
```

The image packages the API binary plus built dashboard assets under `/opt/viaduct/web`. The default container command starts `viaduct serve-api --port 8080`, which serves the dashboard at `/` and the API under `/api/v1/`. `serve-api` binds to loopback by default and expects explicit API credentials before you expose it remotely.

## Install On Unix-Like Systems

From an unpacked release bundle:

```bash
PREFIX=/usr/local ./install.sh ./bin/viaduct ./web ./configs ./examples ./docs
```

This installs:
- `viaduct` to `$PREFIX/bin`
- web assets to `$PREFIX/share/viaduct/web`
- starter config to `$PREFIX/etc/viaduct/config.yaml`
- bundled examples and docs under `$PREFIX/share/viaduct`

## Install On Windows

From an unpacked release bundle:

```powershell
.\install.ps1 -SourceBin .\bin\viaduct.exe -SourceWeb .\web -SourceConfigs .\configs -SourceExamples .\examples -SourceDocs .\docs
```

The default target is a user-local directory under `%LOCALAPPDATA%\Programs\Viaduct`.

When the bundled web assets are present, the installer leaves you with one obvious next step:

```bash
viaduct start --config /your/prefix/etc/viaduct/config.yaml
```

The generated starter config points at the installed lab fixtures when no config already exists at that location.

## First-Run Files
- CLI config example: [`../../configs/config.example.yaml`](../../configs/config.example.yaml)
- Root env example: [`../../.env.example`](../../.env.example)
- Dashboard env example: [`../../web/.env.example`](../../web/.env.example)

## Verification
After install:

```bash
viaduct version
viaduct --help
viaduct doctor
```

If you installed only the CLI and not the dashboard assets, the API and migration/lifecycle backends still work; only the packaged static WebUI is absent.

For browser access in packaged environments:
- keep `VIADUCT_ALLOWED_ORIGINS` empty for same-origin deployments; set it only if the dashboard is served from a different trusted origin
- set `VIADUCT_WEB_DIR` only if the built dashboard assets live outside the standard packaged or installed paths
- prefer service account keys for normal operator access
- rely on the Get started session flow for packaged environments unless you intentionally pre-seed a development-only browser credential
- use `VIADUCT_WORKSPACE_JOB_TIMEOUT` if workspace jobs need a different server-side timeout budget
- use `VIADUCT_WORKSPACE_JOB_CONCURRENCY` if packaged workspace execution needs a different bounded worker count
- use `viaduct serve-api` directly only when you intentionally want the lower-level service command instead of the local `start` flow, and keep its default loopback bind unless API credentials are configured
- reserve `VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE=true` for disposable break-glass scenarios only
