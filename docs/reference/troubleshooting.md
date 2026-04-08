# Troubleshooting

## `invalid output format`

Cause:
- the CLI `--output` flag was set to a value other than `table`, `json`, or `yaml`

Fix:
- rerun with a supported output format

## `missing tenant API key`

Cause:
- you called a tenant-scoped API route without either `X-API-Key` or `X-Service-Account-Key`
- or the default-tenant fallback is no longer active because custom tenants exist

Fix:
- provide a valid tenant API key or service-account key
- verify tenant status in the admin tenant list
- prefer a service-account key for normal dashboard or automation use if you are not intentionally using tenant-admin access

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
- `viaduct serve-api` is not running
- Vite is not proxying to `localhost:8080`
- neither `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` nor `VITE_VIADUCT_API_KEY` is configured for tenant-protected routes

Fix:
- start `viaduct serve-api --port 8080`
- verify the dashboard `.env` file
- prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` for normal dashboard use
- confirm API health at `/api/v1/health`

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
