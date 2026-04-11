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
- Do not expose the Vite development server as a shared or internet-facing surface.

See [SUPPORT.md](SUPPORT.md) for non-security usage questions.
