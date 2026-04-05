# Installation

This document is the top-level installation entrypoint for Viaduct. For a deeper walkthrough, see [docs/getting-started/installation.md](docs/getting-started/installation.md).

## Requirements
- Go 1.24 or newer
- Node.js 20.19 or newer if you want to build the dashboard
- `make` for the standard workflow
- `golangci-lint` for local linting

## Install From Source

```bash
git clone https://github.com/eblackrps/viaduct.git
cd viaduct
go mod tidy
make build
./bin/viaduct version
```

Build the dashboard if you need the web UI:

```bash
cd web
npm ci
npm run build
```

## Install From A Release Bundle

Release bundles produced by `make package-release-matrix` include:
- the CLI binary
- built web assets
- docs and sample configs
- install scripts
- checksums and a release manifest

On POSIX systems:

```bash
PREFIX=/usr/local ./install.sh ./bin/viaduct ./web
```

On Windows PowerShell:

```powershell
.\install.ps1 -SourceBin .\bin\viaduct.exe -SourceWeb .\web -Prefix "$env:LOCALAPPDATA\\Viaduct"
```

## Verify The Install

```bash
viaduct version
viaduct --help
```

The fastest no-infrastructure evaluation path is the local KVM lab in [examples/lab](examples/lab). Continue with [QUICKSTART.md](QUICKSTART.md).

Reference deployment assets for Docker Compose, systemd, and Kubernetes live in [examples/deploy](examples/deploy).
