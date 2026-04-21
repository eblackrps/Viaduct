-- 008 state machine:
--   008a seeds credential_hashes rows idempotently for every tenant and service-account credential.
--   008b clears legacy plaintext only after the expected hash rows are present for that tenant.
--   008 finalizes the phased migration by enforcing global uniqueness on the seeded hash registry.
-- rollback contract:
--   008a is safe to re-run after a crash because it uses idempotent inserts.
--   008b rolls back inside its transaction if any expected hash row is missing or a tenant update fails.
-- MUST NOT be wrapped in a transaction; uses CREATE INDEX CONCURRENTLY.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS credential_hashes_hash_unique
ON credential_hashes (credential_hash);
