# Security Policy

Viaduct publishes fixes to the latest supported code on `main` and to the most recent tagged stable release when practical.

## Supported Versions

| Version | Supported |
| --- | --- |
| `main` | Yes |
| latest stable release | Yes |
| older snapshots or forks | No |

## Reporting A Vulnerability

Please do not open public GitHub issues for suspected security vulnerabilities.

Instead:
- open a private GitHub security advisory for this repository, or
- contact the maintainer through GitHub and include enough detail to reproduce the issue safely

When reporting a vulnerability, include:
- affected version or commit
- reproduction steps
- impact assessment
- any known mitigations or workarounds

## Response Expectations
- Viaduct aims to acknowledge good-faith reports promptly.
- Coordinated disclosure is preferred whenever possible.
- Fixes may land on `main` before a follow-up patch release is cut.

## Operational Notes
- Never include secrets, tenant API keys, or production credentials in a report.
- Use explicit tenant API keys in non-demo environments.
- Do not expose the Vite development server as a public production surface.

See [SUPPORT.md](SUPPORT.md) for non-security usage questions and operational help.
