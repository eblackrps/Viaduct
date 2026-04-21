# Release Cadence

Viaduct `v3.x` moves to milestone-based release management.

## Minor Releases

- `v3.x` minor releases are feature cuts
- cadence is monthly or milestone-driven, not per pull request
- release readiness is defined by the documented release gate, not by merge velocity

## Patch Releases

- patch releases are reserved for shipped regressions
- if no regression ships, no patch release is cut
- deferred scope belongs in the next planned minor release, not an opportunistic patch

## Edge Image Policy

- `ghcr.io/eblackrps/viaduct:edge` is published from merges to `main`
- `:edge` is explicitly not for production use
- production deployments should pin immutable semver tags

## Deprecation Policy

- breaking changes must be called out in release notes and upgrade guidance
- operators receive at least one minor release of warning before a removal, unless a security issue requires faster action
- deprecated behavior remains documented until it is removed
