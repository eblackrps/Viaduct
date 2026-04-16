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

Quality checks for dashboard work:

```bash
npm run lint
npm run format
npm run test
npm run e2e
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
- `VITE_VIADUCT_API_TIMEOUT_MS`: request timeout in milliseconds for dashboard fetches

See [./.env.example](./.env.example).

The dashboard also supports runtime authentication bootstrap. When no environment key is set, the app either:
- uses the local single-user fallback when the default tenant is available without a key, or
- opens the bootstrap screen so the operator can provide a service-account or tenant key at runtime

The runtime bootstrap path now keeps the actual API credential in a server-backed session and `httpOnly` cookie. The browser stores only an opaque session marker. Non-persistent sessions use session storage for that marker, and the remember option persists only the marker in local storage on trusted workstations.

## Notes

- Vite 8 and the current React plugin require Node.js 20.19+ or a newer supported major.
- The Vite dev server is for local development only.
- The dashboard depends on the same backend state as the CLI and API; avoid frontend-only assumptions about migration or policy state.
- The default route is the pilot workspace flow in `web/src/features/workspaces/WorkspacePage.tsx`.
- The app shell groups pages into pilot, operate, govern, admin, and analysis sections so the workspace-first route stays prominent without hiding the rest of the operator surfaces.
- Live runtime API docs are served by the backend at `/api/v1/docs`.
