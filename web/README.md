# Dashboard

This directory contains the React and Vite dashboard for Viaduct.

## Purpose

The dashboard is the UI for the current assessment workflow. Its first-run experience is: create an assessment, discover, inspect, simulate, save a plan, and export a report.

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
npm run screenshots:readme
```

From the repo root, `make web-install` installs dependencies, `make web-e2e-setup` adds the Playwright browser for the real-runtime smoke, and `make pilot-smoke` runs the root-level evaluator flow that pairs the assessment integration smoke with the browser runtime smoke.

## Build

```bash
npm ci
npm run build
```

The dashboard ships inside the primary packaged OCI image published by `.github/workflows/image.yml`. `make package-release-matrix` still produces native bundles as an alternative path, and `make release-gate` remains the release check command. CI adds Playwright end-to-end coverage plus `gosec` and `trivy` on top of that same source-controlled flow.

For the packaged local dashboard path from the repo root:

```bash
make build
make web-build
./bin/viaduct start
```

Then open [http://127.0.0.1:8080](http://127.0.0.1:8080). The local runtime serves the built dashboard from `web/dist` when those assets are present and can generate the default local lab config automatically on a fresh checkout.
The lower-level `viaduct serve-api` path also serves the dashboard, but it now binds to loopback by default and refuses unauthenticated remote exposure unless credentials are configured or the explicit dangerous override is enabled.

## Environment

- `VITE_VIADUCT_API_KEY`: tenant-scoped API key for advanced or tenant-admin development access
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: preferred service account key for the normal workflow
- `VITE_VIADUCT_API_TIMEOUT_MS`: request timeout in milliseconds for dashboard fetches

See [./.env.example](./.env.example).

The dashboard also supports a runtime Get started flow. When no environment key is set, the app either:
- offers a direct loopback-only `Start local session` path when the packaged runtime started through `viaduct start` is running against the default local lab path, or
- opens the Get started screen so the user can use a service account key, with the tenant key path kept under advanced options when needed

The runtime Get started flow keeps tenant or service account credentials in a server-backed session behind an `httpOnly` cookie. The browser stores only an opaque session marker, and local sessions do not use an API key at all. Non-persistent sessions use session storage for that marker, and the remember option persists only the marker in local storage on trusted workstations.
Tenant and service account keys are persisted by the backend as non-recoverable hashes; the raw key is only shown at create or rotate time.

Current README and release-facing dashboard screenshots can be regenerated with `npm run screenshots:readme`. That script builds the dashboard, boots the seeded Playwright fixture server, and captures the checked-in PNG assets used by the root README and demo collateral. Runtime compatibility against the actual `viaduct start` path is covered separately by `npm run e2e:runtime`.

## Notes

- Vite 8 and the current React plugin require Node.js 20.19+ or a newer supported major. CI and release packaging currently pin Node.js 20.20.x.
- `.codex/setup.sh` now enforces the same Node floor and can install the `web/` dependencies plus the Playwright Chromium runtime unless you opt out with `VIADUCT_SKIP_WEB_SETUP=1` or `VIADUCT_SKIP_PLAYWRIGHT_INSTALL=1`.
- The Vite dev server is for local development only.
- The dashboard depends on the same backend state as the CLI and API; avoid frontend-only assumptions about migration or policy state.
- The default route is the assessment workflow in `web/src/features/workspaces/WorkspacePage.tsx`.
- The app shell groups pages into Operate, Govern, Observe, and Admin sections so the assessment route stays prominent without hiding the rest of the dashboard.
- Live runtime API docs are served by the backend at `/api/v1/docs`.
