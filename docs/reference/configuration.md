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
- `state_store_dsn`: PostgreSQL DSN for persistent state; when empty, Viaduct uses the in-memory store unless production mode is enabled
- `credentials`: reusable auth and transport blocks keyed by migration-spec `credential_ref`
- `sources`: keyed by source address or platform name, mapped to shared connector config
- `plugins`: optional platform-to-plugin map for external connector processes or already-running gRPC plugin endpoints

## CLI Environment Variables
- `VIADUCT_USERNAME`: overrides config file username for CLI connector auth
- `VIADUCT_PASSWORD`: overrides config file password for CLI connector auth
- `VIADUCT_ADMIN_KEY`: admin API key used by the REST server for tenant administration. Production mode requires the persisted `sha256:<hex>` digest. Non-production local evaluation can still use the legacy plaintext secret for compatibility. The request header still carries the plaintext `X-Admin-Key`; see [`../operations/admin-key.md`](../operations/admin-key.md).
- `VIADUCT_PLUGIN_ADDR`: plugin listener address used by community connector plugins
- `VIADUCT_ALLOWED_ORIGINS`: comma-separated browser origins allowed to call the API from another origin; defaults to same-origin only when empty
- `VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE`: explicit dangerous override that permits a non-loopback `serve-api` bind without configured admin, tenant, or service account credentials; leave this unset outside disposable break-glass scenarios
- `VIADUCT_ENVIRONMENT`: set to `production` for persistent deployments; production mode refuses in-memory state, missing auth, missing lifecycle policies, and missing dashboard assets at startup
- `VIADUCT_STATE_STORE_DSN`: PostgreSQL DSN override for container, service, and Helm deployments; when set, it takes precedence over `state_store_dsn`
- `VIADUCT_WEB_DIR`: override path for built dashboard assets when they are not in `web/dist`, `web/`, or the installed `share/viaduct/web` layout
- `VIADUCT_OTEL_ENABLED`: enables backend OpenTelemetry trace export when set to `true`; defaults to disabled
- `VIADUCT_OTEL_ENDPOINT`: OTLP/HTTP endpoint for backend trace export; defaults to `http://127.0.0.1:4318`
- `VIADUCT_OTEL_SERVICE_NAME`: override for the reported OpenTelemetry service name; defaults to `viaduct-api`
- `VIADUCT_OTEL_ENVIRONMENT`: deployment environment label added to traces; defaults to `local` for `viaduct start` and `self-hosted` otherwise
- `VIADUCT_OTEL_SAMPLER`: trace sampler name; supported values are `parentbased_traceidratio`, `traceidratio`, `always_on`, `always_off`, `parentbased_always_on`, and `parentbased_always_off`
- `VIADUCT_OTEL_SAMPLER_ARG`: numeric sample ratio used by the ratio-based samplers; defaults to `1`
- `VIADUCT_WORKSPACE_JOB_TIMEOUT`: per-job server-side timeout for pilot workspace discovery, graph, simulation, and plan generation; defaults to `2m`
- `VIADUCT_WORKSPACE_ENQUEUE_TIMEOUT`: maximum time an API request waits for the bounded workspace executor to acknowledge queue admission before returning `ErrEnqueueTimeout`; defaults to `30s`
- `VIADUCT_WORKSPACE_JOB_CONCURRENCY`: bounded worker count for queued and recovered workspace jobs; defaults to `4`
- `VIADUCT_HTTP_READ_HEADER_TIMEOUT`: maximum time to read request headers; defaults to `10s`
- `VIADUCT_HTTP_READ_TIMEOUT`: maximum time to read a full request; defaults to `30s`
- `VIADUCT_HTTP_WRITE_TIMEOUT`: maximum time to write a response; defaults to `60s`
- `VIADUCT_HTTP_IDLE_TIMEOUT`: maximum idle keep-alive connection time; defaults to `120s`
- `VIADUCT_HTTP_SHUTDOWN_TIMEOUT`: maximum graceful HTTP shutdown time after server cancellation; defaults to `5s`
- `VIADUCT_AUTH_SESSION_TTL`: dashboard runtime auth-session lifetime for non-persistent browser sessions; defaults to `12h`
- `VIADUCT_AUTH_REMEMBER_TTL`: dashboard runtime auth-session lifetime for remembered browser sessions; defaults to `168h` (7 days) and is capped there unless `VIADUCT_LONG_SESSION_DAYS` is set
- `VIADUCT_LONG_SESSION_DAYS`: explicit override that permits remembered dashboard sessions longer than 7 days when you intentionally accept the larger persistence window
- `VIADUCT_TRUSTED_PROXIES`: comma-separated CIDR list of reverse proxies allowed to supply forwarded scheme and client-IP headers; when empty, Viaduct trusts only the direct peer address and TLS state
- `VIADUCT_API_CSP`: override the default API response Content Security Policy
- `VIADUCT_DASHBOARD_CSP`: override the default bundled-dashboard Content Security Policy
- `VIADUCT_DB_READ_TIMEOUT`: PostgreSQL read-query timeout; defaults to `5s`
- `VIADUCT_DB_WRITE_TIMEOUT`: PostgreSQL write-query timeout; defaults to `10s`
- `VIADUCT_DB_MAX_OPEN_CONNS`: PostgreSQL maximum open connections; defaults to `25`
- `VIADUCT_DB_MAX_IDLE_CONNS`: PostgreSQL maximum idle connections; defaults to `5`
- `VIADUCT_DB_CONN_MAX_LIFETIME`: PostgreSQL connection max lifetime; defaults to `5m`

## Dashboard Environment Variables
- `VITE_VIADUCT_API_KEY`: tenant API key injected into dashboard requests
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: scoped service account key injected into dashboard requests; when set, the dashboard prefers this header over `VITE_VIADUCT_API_KEY`
- `VITE_VIADUCT_API_TIMEOUT_MS`: dashboard fetch timeout in milliseconds; defaults to `30000`

The dashboard reads this through Vite. See [`../../web/.env.example`](../../web/.env.example).

The dashboard now also supports a runtime Get started flow. When neither variable is set, the app either:
- offers a direct loopback-only local operator session when `viaduct start` is running against the default local lab path and the default tenant is still unkeyed, or
- starts on the Get started screen and accepts a service account key or, when needed, a tenant key under advanced options at runtime

The runtime Get started flow creates a server-backed session. The browser stores only an opaque session identifier, and any tenant or service account key stays server-side for that session instead of landing in browser storage. Local operator sessions do not use an API key at all. Non-persistent sessions keep that identifier in session storage. Operators can explicitly choose to remember only that non-sensitive marker in local storage on trusted workstations.

Prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` for normal dashboard access when you intentionally pre-seed a development build. Reserve `VITE_VIADUCT_API_KEY` for tenant setup, short-lived admin work, or break-glass access.

Tenant and service account credentials are persisted as non-recoverable hashes in both the in-memory and PostgreSQL stores. Viaduct only reveals a raw key during tenant creation or an explicit service-account create/rotate response.

## API Authentication Headers
- `X-API-Key`: tenant-scoped API key for inventory, migration, lifecycle, and summary routes
- `X-Service-Account-Key`: scoped machine credential for tenant service accounts
- `X-Admin-Key`: admin-only plaintext API key for tenant creation and deletion; the stored server-side `VIADUCT_ADMIN_KEY` must be `sha256:<hex>` in production mode
- `X-Request-ID`: optional caller-supplied request correlation ID; when absent, the API generates one
- `X-Trace-ID`: response header exposing the current backend trace identifier when tracing is active
- `Traceparent`: optional inbound W3C trace context header for request-to-request correlation

## Observability Notes

- Trace export is opt-in and safe to leave disabled.
- Viaduct continues serving requests if the configured OTLP backend is unreachable after startup; trace export is best-effort.
- If the exporter cannot be created during startup, Viaduct logs a warning and continues without telemetry rather than failing the server process.
- The built-in `/metrics` endpoint remains the official metrics contract for now. OpenTelemetry metrics export is intentionally deferred until Viaduct adopts a single operator-facing metrics path.

## Tenant Defaults
- The built-in `default` tenant exists automatically in both the memory store and PostgreSQL store.
- `viaduct start` exposes the local operator session only through the explicit loopback auth-session flow. A fresh clone can still reach the WebUI without manual key seeding, but protected routes require the issued session cookie rather than ambient fallback access.
- Any shared or persistent deployment should use explicit tenant or service account keys instead of relying on the local session path.
- Any pilot or packaged dashboard deployment should prefer named service account credentials over a shared tenant-wide key.

## State Store Notes
- in-memory store: useful for demos, tests, and local evaluation only
- PostgreSQL store: recommended for any persistent environment
- the PostgreSQL backend auto-creates its schema on startup
- production mode requires PostgreSQL-backed state and reports the active schema version through `/readyz` and `/api/v1/about`

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
