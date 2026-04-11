# Installation

This is the top-level installation entrypoint for Viaduct. Use it together with [QUICKSTART.md](QUICKSTART.md) if you want the fastest evaluation path, or see [docs/getting-started/installation.md](docs/getting-started/installation.md) for the deeper walkthrough.

## Requirements

- Go 1.24 or newer
- Node.js 20.19 or newer if you want to build the dashboard
- `make` for the standard workflow
- `golangci-lint` for local linting

## Install From Source

```bash
git clone https://github.com/eblackrps/Viaduct.git
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

Tagged bundles are attached to [GitHub Releases](https://github.com/eblackrps/Viaduct/releases), and the same layout can be generated locally through `make package-release-matrix`.

Release bundles produced by `make package-release-matrix` include:
- CLI binary
- built web assets
- docs and sample configs
- install scripts
- deployment reference assets
- checksums, a release manifest, and a dependency manifest

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

## Recommended Next Step

The cleanest evaluation path is still the local lab in [examples/lab](examples/lab). Continue with [QUICKSTART.md](QUICKSTART.md).

For packaged or persistent evaluation environments:
- use PostgreSQL instead of the in-memory store
- prefer service-account keys for normal operator access
- set `VIADUCT_ALLOWED_ORIGINS` if the dashboard is served from anything other than the default local Vite origins
- tune `VIADUCT_WORKSPACE_JOB_TIMEOUT` if discovery or planning jobs need a different server-side timeout budget
- keep the Vite dev server out of any shared or internet-facing environment
