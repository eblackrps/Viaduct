# Installation

Viaduct can be evaluated from source or from a packaged release bundle.

## Source Build

### Prerequisites
- Go 1.24+
- Node.js 20+ for the dashboard
- `make` for the standard build flow
- `qemu-img` if you want live disk-conversion execution outside tests

### Build The CLI

```bash
go mod tidy
make build
./bin/viaduct version
```

### Build The Dashboard

```bash
cd web
npm ci
npm run build
```

## Packaged Release Bundle

Generate a self-contained local bundle:

```bash
make package-release
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
- `SHA256SUMS.txt`

## Container Build

Build the container image:

```bash
make docker
```

The image packages the API binary plus built dashboard assets under `/opt/viaduct/web`. The default entrypoint starts `viaduct serve-api --port 8080`.

## Install On Unix-Like Systems

From an unpacked release bundle:

```bash
PREFIX=/usr/local ./install.sh ./bin/viaduct ./web
```

This installs:
- `viaduct` to `$PREFIX/bin`
- web assets to `$PREFIX/share/viaduct/web`

## Install On Windows

From an unpacked release bundle:

```powershell
.\install.ps1 -SourceBin .\bin\viaduct.exe -SourceWeb .\web
```

The default target is a user-local directory under `%LOCALAPPDATA%\Programs\Viaduct`.

## First-Run Files
- CLI config example: [`../../configs/config.example.yaml`](../../configs/config.example.yaml)
- Root env example: [`../../.env.example`](../../.env.example)
- Dashboard env example: [`../../web/.env.example`](../../web/.env.example)

## Verification
After install:

```bash
viaduct version
viaduct --help
```

If you installed only the CLI and not the dashboard assets, the API and migration/lifecycle backends still work; only the packaged static web assets are absent.
