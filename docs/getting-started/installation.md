# Installation

Viaduct can be evaluated from source or from a packaged release bundle.

## Source Build

### Prerequisites
- Go 1.24+
- Node.js 20.19+ for the dashboard
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
The same runtime also serves live operator API docs at `http://127.0.0.1:8080/api/v1/docs`.

## Packaged Release Bundle

Tagged release bundles are attached to [GitHub Releases](https://github.com/eblackrps/Viaduct/releases). The same structure can also be generated locally through `make package-release-matrix`.

Generate a self-contained local bundle:

```bash
make package-release-matrix
```

This creates:
- `dist/viaduct_<version>_<goos>_<goarch>/`
- `dist/viaduct_<version>_<goos>_<goarch>.zip`

Each bundle includes:
- the Viaduct CLI binary
- built web assets
- install scripts
- docs, configs, and examples
- `release-manifest.json`
- `dependency-manifest.json`
- `SHA256SUMS.txt`

## Container Build

Build the container image:

```bash
make docker
```

The image packages the API binary plus built dashboard assets under `/opt/viaduct/web`. The default container command starts `viaduct serve-api --port 8080`, which serves the dashboard at `/` and the API under `/api/v1/`.

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
- prefer service-account keys for normal operator access
- rely on the runtime auth bootstrap for packaged environments unless you intentionally pre-seed a development-only browser credential
- use `VIADUCT_WORKSPACE_JOB_TIMEOUT` if workspace jobs need a different server-side timeout budget
- use `viaduct serve-api` directly only when you intentionally want the lower-level service command instead of the local `start` flow
