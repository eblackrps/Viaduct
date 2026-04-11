# Configuration Reference

Viaduct uses a small CLI config file, explicit API headers, and a few environment variables for credentials and dashboard access.

## CLI Config File

Default path:

```text
~/.viaduct/config.yaml
```

Example:

```yaml
username: ""
password: ""
insecure: false
state_store_dsn: "postgres://viaduct:change-me@localhost:5432/viaduct?sslmode=disable"

credentials:
  source/vcenter:
    username: "administrator@vsphere.local"
    password: "replace-me"
    insecure: true
  target/proxmox:
    username: "root@pam"
    password: "replace-me"
    insecure: true

sources:
  vmware:
    address: "https://vcenter.example.local"
    username: "administrator@vsphere.local"
    password: "replace-me"
    insecure: true
  kvm:
    address: "examples/lab/kvm"

plugins:
  example: "grpc://127.0.0.1:50071"
```

Fields:
- `username`: default username used when a source-specific value is absent
- `password`: default password used when a source-specific value is absent
- `insecure`: default TLS skip-verify behavior
- `state_store_dsn`: PostgreSQL DSN for persistent state; when empty, Viaduct uses the in-memory store
- `credentials`: reusable auth and transport blocks keyed by migration-spec `credential_ref`
- `sources`: keyed by source address or platform name, mapped to shared connector config
- `plugins`: optional platform-to-plugin map for external connector processes or already-running gRPC plugin endpoints

## CLI Environment Variables
- `VIADUCT_USERNAME`: overrides config file username for CLI connector auth
- `VIADUCT_PASSWORD`: overrides config file password for CLI connector auth
- `VIADUCT_ADMIN_KEY`: admin API key used by the REST server for tenant administration
- `VIADUCT_PLUGIN_ADDR`: plugin listener address used by community connector plugins
- `VIADUCT_ALLOWED_ORIGINS`: comma-separated browser origins allowed to call the API from another origin; defaults to the local Vite origins on ports `5173` and `4173`
- `VIADUCT_WEB_DIR`: override path for built dashboard assets when they are not in `web/dist`, `web/`, or the installed `share/viaduct/web` layout
- `VIADUCT_WORKSPACE_JOB_TIMEOUT`: per-job server-side timeout for pilot workspace discovery, graph, simulation, and plan generation; defaults to `2m`

## Dashboard Environment Variables
- `VITE_VIADUCT_API_KEY`: tenant API key injected into dashboard requests
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: scoped service-account key injected into dashboard requests; when set, the dashboard prefers this header over `VITE_VIADUCT_API_KEY`

The dashboard reads this through Vite. See [`../../web/.env.example`](../../web/.env.example).

The dashboard now also supports runtime authentication bootstrap. When neither variable is set, the app either:
- uses the built-in local single-user fallback when only the default tenant exists and that tenant has no API key configured, or
- starts on a bootstrap screen and accepts either a service-account key or tenant key at runtime

The selected browser credential is stored in session storage by default and is cleared when the browser session ends. Operators can explicitly choose to remember a runtime key in local storage on trusted workstations.

Prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` for normal dashboard access when you intentionally pre-seed a development build. Reserve `VITE_VIADUCT_API_KEY` for tenant bootstrap, short-lived admin work, or break-glass access.

## API Authentication Headers
- `X-API-Key`: tenant-scoped API key for inventory, migration, lifecycle, and summary routes
- `X-Service-Account-Key`: scoped machine credential for tenant service accounts
- `X-Admin-Key`: admin-only API key for tenant creation and deletion
- `X-Request-ID`: optional caller-supplied request correlation ID; when absent, the API generates one

## Tenant Defaults
- The built-in `default` tenant exists automatically in both the memory store and PostgreSQL store.
- The API can fall back to the default tenant only when there are no active custom tenants and the default tenant has no API key configured.
- `viaduct start` relies on that default-tenant fallback for the standard local lab path so a fresh clone can reach the WebUI without manual key seeding.
- Any shared or persistent deployment should use explicit tenant keys rather than relying on fallback behavior.
- Any pilot or packaged dashboard deployment should prefer named service-account credentials over a shared tenant-wide key.

## State Store Notes
- in-memory store: useful for demos, tests, and local evaluation only
- PostgreSQL store: recommended for any persistent environment
- the PostgreSQL backend auto-creates its schema on startup

## Connector Config Shape

All built-in connectors consume the shared config fields below:

```yaml
address: "https://endpoint.example.local"
username: "operator"
password: "replace-me"
insecure: false
port: 443
```

Not every connector uses every field. For example, KVM fixture discovery typically uses only `address`.

## Plugin Configuration Notes
- Plugin mappings are keyed by platform identifier, for example `example` or `custom-kvm`.
- Values can be executable paths or `grpc://host:port` endpoints for already-running plugin processes.
- Executable plugins should ship a `plugin.json` sidecar manifest with name, platform, version, protocol version metadata, and optional host-version constraints.
