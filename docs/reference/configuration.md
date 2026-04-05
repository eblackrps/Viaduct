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

sources:
  vmware:
    address: "https://vcenter.example.local"
    username: "administrator@vsphere.local"
    password: "replace-me"
    insecure: true
  kvm:
    address: "examples/lab/kvm"
```

Fields:
- `username`: default username used when a source-specific value is absent
- `password`: default password used when a source-specific value is absent
- `insecure`: default TLS skip-verify behavior
- `state_store_dsn`: PostgreSQL DSN for persistent state; when empty, Viaduct uses the in-memory store
- `sources`: keyed by source address or platform name, mapped to shared connector config

## CLI Environment Variables
- `VIADUCT_USERNAME`: overrides config file username for CLI connector auth
- `VIADUCT_PASSWORD`: overrides config file password for CLI connector auth
- `VIADUCT_ADMIN_KEY`: admin API key used by the REST server for tenant administration
- `VIADUCT_PLUGIN_ADDR`: plugin listener address used by community connector plugins

## Dashboard Environment Variables
- `VITE_VIADUCT_API_KEY`: tenant API key injected into dashboard requests

The dashboard reads this through Vite. See [`../../web/.env.example`](../../web/.env.example).

## API Authentication Headers
- `X-API-Key`: tenant-scoped API key for inventory, migration, lifecycle, and summary routes
- `X-Admin-Key`: admin-only API key for tenant creation and deletion

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
