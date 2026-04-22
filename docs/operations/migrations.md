# Migration Execution And Crash Safety

Viaduct treats store migrations as operational state transitions, not best-effort startup side effects. Every migration must declare whether it is transactional, idempotent, and safe to re-run after a partial failure.

## Phased Credential Migration (008)

Credential migration uses a crash-safe two-phase model:

1. `008a` seed phase: Viaduct inserts durable credential-hash rows for every tenant and service account credential with `INSERT ... ON CONFLICT DO NOTHING`.
2. `008b` clear phase: Viaduct clears legacy plaintext only after it verifies the expected hash rows already exist for that tenant.
3. `008` uniqueness guard: after the seeded and cleared state is stable, Viaduct applies the concurrent unique index for `credential_hashes`.

If Viaduct crashes after the seed phase but before plaintext is cleared, authentication still succeeds because the legacy plaintext remains present. On restart, Viaduct re-runs the seed phase idempotently and resumes the clear phase only after the preconditions are met.

If the clear phase fails for any tenant, the transaction rolls back for that phase and no plaintext is cleared for that tenant set.

## Transaction Contracts

- transactional migrations either commit completely or roll back completely
- pragma-marked migrations that use `CREATE INDEX CONCURRENTLY` must not be wrapped in a transaction
- phased migrations must document the preconditions that allow the next phase to run

## Operator Diagnostics

Use the migration diagnostic helper to inspect the credential-migration phase state per tenant:

```bash
make migrate-diag STATE_STORE_DSN='postgres://viaduct:change-me@localhost:5432/viaduct?sslmode=disable'
```

The diagnostic reports whether each tenant already has durable hash rows and whether legacy plaintext has been cleared.
