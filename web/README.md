# Dashboard

This directory contains the React and Vite dashboard for Viaduct.

## Purpose

The dashboard is the operator UI for the current assessment-to-pilot wedge. Its first-run experience is the workspace-first flow: create workspace, discover, inspect, simulate, save plan, and export report.

## Development

```bash
node --version
npm ci
npm run dev
```

For a fuller local loop with the API:

```bash
npm ci
npm run dev:full
```

## Build

```bash
npm ci
npm run build
```

Release bundles for the dashboard are produced through `make package-release-matrix`, and `make release-gate` is the canonical verification path.

## Environment

- `VITE_VIADUCT_API_KEY`: tenant-scoped API key for development bootstrap or tenant-admin access
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: preferred service-account key for the normal operator flow

See [./.env.example](./.env.example).

The dashboard also supports runtime authentication bootstrap. When no environment key is set, the app opens a bootstrap screen and stores the chosen service-account or tenant key locally in the browser until the operator signs out or replaces it.

## Notes

- Vite 8 and the current React plugin require Node.js 20.19+ or a newer supported major.
- The Vite dev server is for local development only.
- The dashboard depends on the same backend state as the CLI and API; avoid frontend-only assumptions about migration or policy state.
- The default route is the pilot workspace flow in `web/src/features/workspaces/WorkspacePage.tsx`.
