# Troubleshooting

## `invalid output format`

Cause:
- the CLI `--output` flag was set to a value other than `table`, `json`, or `yaml`

Fix:
- rerun with a supported output format

## `missing tenant API key`

Cause:
- you called a tenant-scoped API route without `X-API-Key`
- or the default-tenant fallback is no longer active because custom tenants exist

Fix:
- provide a valid tenant API key
- verify tenant status in the admin tenant list

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
- `VITE_VIADUCT_API_KEY` is missing for tenant-protected routes

Fix:
- start `viaduct serve-api --port 8080`
- verify the dashboard `.env` file
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
- discover returned a nil result

Fix:
- validate the plugin against the checklist in [plugin-author-guide.md](plugin-author-guide.md)
- test the plugin through `internal/connectors/plugin/host.go` behavior first
