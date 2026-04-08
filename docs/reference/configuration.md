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

## Dashboard Environment Variables
- `VITE_VIADUCT_API_KEY`: tenant API key injected into dashboard requests
- `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`: scoped service-account key injected into dashboard requests; when set, the dashboard prefers this header over `VITE_VIADUCT_API_KEY`

The dashboard reads this through Vite. See [`../../web/.env.example`](../../web/.env.example).

## API Authentication Headers
- `X-API-Key`: tenant-scoped API key for inventory, migration, lifecycle, and summary routes
- `X-Service-Account-Key`: scoped machine credential for tenant service accounts
- `X-Admin-Key`: admin-only API key for tenant creation and deletion
- `X-Request-ID`: optional caller-supplied request correlation ID; when absent, the API generates one

## Tenant Defaults
- The built-in `default` tenant exists automatically in both the memory store and PostgreSQL store.
- The API can fall back to the default tenant only when there are no active custom tenants and the default tenant has no API key configured.
- Any production deployment should use explicit tenant keys rather than relying on fallback behavior.

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
