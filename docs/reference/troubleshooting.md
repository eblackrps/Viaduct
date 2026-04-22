# Troubleshooting

## `invalid output format`

Cause:
- the CLI `--output` flag was set to a value other than `table`, `json`, or `yaml`

Fix:
- rerun with a supported output format

## `missing tenant API key`

Cause:
- you called a tenant-scoped API route without either `X-API-Key` or `X-Service-Account-Key`
- or the local operator session was not started for this browser

Fix:
- provide a valid tenant API key or service account key
- on the default local lab path, use `Start local session` from the Get started screen and keep the request direct to `127.0.0.1`
- verify tenant status in the admin tenant list
- prefer a service account key for normal dashboard or automation use if you are not intentionally using tenant-admin access

## `migration requires approval before execution`

Cause:
- the migration spec includes an approval gate and it has not been satisfied

Fix:
- execute or resume through the API with an approval payload containing `approved_by`

## `migration window opens at ...` or `migration window closed at ...`

Cause:
- the spec execution window does not permit the current time

Fix:
- update the migration spec or execute during the allowed window

## Dashboard Cannot Reach The API

Cause:
- `viaduct start` or `viaduct serve-api` is not running
- the built dashboard assets are missing from the default serve path
- Vite is not proxying to `localhost:8080` when you are using the dev server
- no active dashboard session or pre-seeded dashboard key is available for tenant-protected routes

Fix:
- if you are deploying with Docker, confirm the container is running with `docker ps` and inspect logs with `docker logs`
- start `viaduct start`
- open `http://127.0.0.1:8080` for the default same-origin operator path
- set `VIADUCT_WEB_DIR` only if the built dashboard assets live outside the standard packaged or installed paths
- use the Get started screen or prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` for local development
- confirm API health at `/api/v1/health`

## Browser Reports `origin not allowed`

Cause:
- the dashboard is running from an origin that the API CORS allowlist does not trust

Fix:
- set `VIADUCT_ALLOWED_ORIGINS` to a comma-separated list of trusted dashboard origins
- restart `viaduct serve-api`
- keep the value narrow instead of using a wildcard
- skip this override for the default same-origin path on `http://127.0.0.1:8080`

## `viaduct start` Says The WebUI Assets Are Missing

Cause:
- the dashboard has not been built yet in a source checkout
- `VIADUCT_WEB_DIR` points to the wrong directory
- the packaged or installed web assets are incomplete

Fix:
- run `make web-build` before `viaduct start` in a source checkout
- use `viaduct doctor` to confirm the resolved dashboard asset path
- set `VIADUCT_WEB_DIR` only when the built dashboard assets live outside the standard packaged or installed paths
- rebuild or redeploy the container image if `/opt/viaduct/web` is missing in a packaged container environment
- reinstall from a complete native release bundle if the packaged `web/` layout is missing

## Workspace Job Fails With `context deadline exceeded`

Cause:
- the server-side workspace job timeout elapsed before discovery, graph generation, simulation, or planning completed

Fix:
- increase `VIADUCT_WORKSPACE_JOB_TIMEOUT`
- retry the job from the workspace job history after adjusting the timeout
- use PostgreSQL for persistent evaluation environments so retry and recovery state survive restarts

## Dashboard Sign-In Disappears After Closing The Browser

Cause:
- the Get started flow uses a non-persistent browser session by default
- the browser session marker was stored only in session storage, so it was intentionally cleared when the browser closed

Fix:
- sign in again
- use the Keep me signed in option if you intentionally want the browser to keep a local session marker across restarts
- prefer service account keys over tenant keys for remembered browser sessions
- review `VIADUCT_AUTH_SESSION_TTL` and `VIADUCT_AUTH_REMEMBER_TTL` if the server-side session window is too short for your environment

## KVM Fixture Discovery Returns No VMs

Cause:
- the `--source` path does not point at XML fixture files

Fix:
- use `examples/lab/kvm`
- ensure the directory contains `*.xml` files

## PostgreSQL Store Startup Fails

Cause:
- the DSN is invalid
- the database is unreachable
- credentials or SSL settings are wrong

Fix:
- verify `state_store_dsn`
- test database connectivity separately
- restart Viaduct after correcting the DSN

## Plugin Load Failures

Cause:
- plugin health returned an unexpected status
- platform ID is empty
- plugin manifest platform or protocol version does not match what Viaduct expects
- discover returned a nil result

Fix:
- validate the plugin against the checklist in [plugin-author-guide.md](plugin-author-guide.md)
- test the plugin through `internal/connectors/plugin/host.go` behavior first
- keep `plugin.json` next to the executable and ensure `protocol_version` is `v1`

## `tenant rate limit exceeded`

Cause:
- a tenant is sending more requests than the configured in-process rate limiter allows during the current window

Fix:
- retry after the `Retry-After` interval
- reduce dashboard polling or automation burst size
- use request correlation IDs and logs to identify the noisiest callers
