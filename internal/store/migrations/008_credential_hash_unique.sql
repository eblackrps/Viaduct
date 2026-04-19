CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS credential_hashes_hash_unique
ON credential_hashes (credential_hash);
