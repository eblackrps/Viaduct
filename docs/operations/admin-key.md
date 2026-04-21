# Admin API Key Formats

Viaduct accepts `VIADUCT_ADMIN_KEY` in two formats:

- preferred: `sha256:<hex>` where `<hex>` is the lowercase SHA-256 digest of the plaintext admin key
- legacy compatibility: the plaintext secret itself

Viaduct always compares the presented `X-Admin-Key` header against the SHA-256 digest of the presented value in constant time. When the stored environment variable is still plaintext, Viaduct logs a one-shot startup warning and a one-shot successful-auth warning recommending migration to the hashed form.

## Recommended Format

Store the hashed form:

```bash
printf '%s' 'replace-me' | sha256sum
```

Then set:

```text
VIADUCT_ADMIN_KEY=sha256:<hex>
```

The dashboard and API callers continue to present the plaintext secret in `X-Admin-Key`. Only the stored server-side configuration value changes.

## Migration Guidance

1. Generate the SHA-256 digest for the existing admin key.
2. Prefix the digest with `sha256:`.
3. Replace the plaintext `VIADUCT_ADMIN_KEY` value in the deployment environment.
4. Restart Viaduct and confirm the plaintext compatibility warning no longer appears.

The startup warning links back to this document so operators can rotate the stored value without changing clients.
