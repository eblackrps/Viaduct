# Deployment Notes

## Loopback Runtime Mode

Viaduct's loopback-only local runtime protections are TCP-only.

- `viaduct start` and `viaduct serve-api --local-runtime` expect a TCP host and port such as `127.0.0.1:8080`.
- Unix socket-style listener addresses are not considered loopback-equivalent for the local session flow, same-origin checks, or direct-loopback trust decisions.
- If you try to pair local-runtime loopback enforcement with a unix-style bind address, Viaduct logs a startup warning so the trust boundary stays explicit.

## Operator Guidance

- Keep the local-runtime Get started session flow on a loopback TCP bind such as `127.0.0.1`.
- When you deploy behind a reverse proxy, configure `VIADUCT_TRUSTED_PROXIES` with the proxy CIDR ranges before relying on forwarded client IP or scheme headers.
- For remotely reachable packaged environments, prefer explicit admin, tenant, or service account credentials instead of the local-runtime session path.
