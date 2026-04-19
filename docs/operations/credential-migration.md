# Credential Migration Remediation

Viaduct persists tenant and service-account credentials as non-recoverable hashes. During PostgreSQL startup migrations, Viaduct also verifies that every persisted credential hash is globally unique before it applies or validates the durable uniqueness index.

If startup fails with a duplicate-credential conflict:

1. Identify the tenant IDs listed in the startup error.
2. Inspect the tenant API keys and service-account keys owned by those tenants.
3. Rotate any duplicated tenant or service-account key so each identity has a unique credential.
4. Restart Viaduct after the duplicates have been remediated.

Notes:

- This is treated as a configuration error because Viaduct cannot safely infer which tenant or service account should retain a duplicated credential.
- The migration does not clear plaintext credentials or apply the unique index when the preflight detects duplicates.
- Service-account rotation is already available through `POST /api/v1/service-accounts/{serviceAccountID}/rotate`.
