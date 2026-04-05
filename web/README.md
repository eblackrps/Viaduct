# Dashboard

This directory contains the React and Vite dashboard for Viaduct.

## Purpose
The dashboard surfaces inventory, migration workflows, dependency views, lifecycle analysis, remediation guidance, tenant summaries, and runbook-oriented status from the Viaduct API.

## Development

```bash
npm ci
npm run dev
```

For a full local loop with the API:

```bash
npm ci
npm run dev:full
```

## Build

```bash
npm ci
npm run build
```

Built assets are packaged into release bundles through `make package-release`.

## Environment
- `VITE_VIADUCT_API_KEY`: tenant-scoped API key used by the dashboard in development

See [./.env.example](./.env.example).

## Notes
- The Vite dev server is for local development only and should not be exposed as a public production surface.
- The dashboard depends on the same backend state as the CLI and API; avoid frontend-only assumptions about migration or policy state.
