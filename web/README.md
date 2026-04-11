# Dashboard

This directory contains the React and Vite dashboard for Viaduct.

## Purpose

The dashboard is the operator UI for the current workspace-first workflow. Its first-run experience is: create workspace, discover, inspect, simulate, save plan, and export report.

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

For the packaged local operator path from the repo root:

```bash
make build
make web-build
./bin/viaduct start
```

Then open [http://127.0.0.1:8080](http://127.0.0.1:8080). The local runtime serves the built dashboard from `web/dist` when those assets are present and can generate the default local lab config automatically on a fresh checkout.

## Environment

- `VITE_VIADUCT_API_KEY`: tenant-scoped API key for development bootstrap or tenant-admin access
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: preferred service-account key for the normal operator flow

See [./.env.example](./.env.example).

The dashboard also supports runtime authentication bootstrap. When no environment key is set, the app either:
- uses the local single-user fallback when the default tenant is available without a key, or
- opens the bootstrap screen so the operator can provide a service-account or tenant key at runtime

The default storage is the browser session. Operators can explicitly opt into local storage with the remember option on the bootstrap screen for trusted workstations.

## Notes

- Vite 8 and the current React plugin require Node.js 20.19+ or a newer supported major.
- The Vite dev server is for local development only.
- The dashboard depends on the same backend state as the CLI and API; avoid frontend-only assumptions about migration or policy state.
- The default route is the pilot workspace flow in `web/src/features/workspaces/WorkspacePage.tsx`.
- The app shell groups pages into pilot, operate, govern, admin, and analysis sections so the workspace-first route stays prominent without hiding the rest of the operator surfaces.
