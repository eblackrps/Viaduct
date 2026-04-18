# Security Policy

Viaduct publishes security fixes to `main` and, when practical, to the most recent tagged release. This remains a best-effort open-source process, not a guaranteed support window.

## Maintained References

| Version | Maintenance |
| --- | --- |
| `main` | Best effort |
| most recent tagged release | Best effort when practical |
| older snapshots or forks | No |

## Reporting A Vulnerability

Do not open public GitHub issues for suspected security vulnerabilities.

Instead:
- open a private GitHub security advisory for this repository, or
- contact the maintainer through GitHub with enough detail to reproduce the issue safely

When reporting a vulnerability, include:
- affected version or commit
- reproduction steps
- impact assessment
- any known mitigations or workarounds

## Response Expectations

- Good-faith reports will be acknowledged as promptly as practical.
- Coordinated disclosure is preferred whenever possible.
- Fixes may land on `main` before a follow-up patch release is cut.

## Operational Notes

- Never include secrets, tenant keys, service-account keys, or real environment credentials in a report.
- Prefer service-account keys for normal operator access. Reserve tenant keys for bootstrap or intentional tenant-admin actions.
- Tenant and service-account keys are persisted as non-recoverable hashes. Capture the raw value only when Viaduct returns it during create or rotate operations.
- The dashboard runtime auth flow keeps the actual API credential server-side and in an `httpOnly` cookie. The browser stores only an opaque session identifier.
- When `VIADUCT_ALLOWED_ORIGINS` is empty, Viaduct stays same-origin only. Do not use `*` for API-key deployments.
- Treat the local operator bootstrap as a direct loopback-only session flow. Protected tenant routes should not rely on ambient anonymous fallback in any shared or proxied environment.
- Keep `VIADUCT_ALLOW_UNAUTHENTICATED_REMOTE=false` outside disposable break-glass scenarios. `viaduct serve-api` defaults to loopback and should not be exposed remotely without explicit API credentials.
- The API and bundled dashboard now emit CSP, `X-Content-Type-Options`, `X-Frame-Options`, and `Referrer-Policy` headers, with HSTS added automatically on HTTPS requests.
- Do not expose the Vite development server as a shared or internet-facing surface.

See [SUPPORT.md](SUPPORT.md) for non-security usage questions.
