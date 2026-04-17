# Release Notes

This directory contains release notes and other release-facing narrative assets that support tagged Viaduct releases.

## Files

- [v2.4.0 release notes](v2.4.0.md)
- [v2.3.0 release notes](v2.3.0.md)
- [v2.2.0 release notes](v2.2.0.md)
- [v2.1.0 release notes](v2.1.0.md)
- [v2.0.0 release notes](v2.0.0.md)
- [v1.9.0 release notes](v1.9.0.md)
- [v1.8.0 release notes](v1.8.0.md)
- [v1.7.0 release notes](v1.7.0.md)
- [v1.6.0 release notes](v1.6.0.md)

Keep these notes aligned with:
- [CHANGELOG.md](../../CHANGELOG.md)
- [OpenAPI reference](../reference/openapi.yaml)
- [Screenshot assets](../operations/demo/screenshots/README.md)
- [RELEASE.md](../../RELEASE.md)

The tag workflow in [`.github/workflows/release.yml`](../../.github/workflows/release.yml) uses the versioned note file that matches the tag (for example `docs/releases/v2.4.0.md`) as the source-controlled GitHub release body when it exists. Keep the matching changelog entry and site links in sync before tagging.
