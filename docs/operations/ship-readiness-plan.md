# Ship Readiness Plan

This plan tracks the practical finishing work for Viaduct after the `v3.1.1` release alignment and Get started refactor. It is intentionally scoped around operator trust, evaluator confidence, and repeatable release execution instead of feature redesign.

## Findings Summary

- Release/install surfaces are aligned on `v3.1.1`, but many files still carry hand-maintained version strings and image tags.
- The workspace-first evaluator flow already exists in the API smoke and dashboard runtime smoke, but it needed one obvious golden path that reaches exported evidence.
- `viaduct doctor` was a useful local bootstrap check, but it did not yet answer store, auth, or readiness questions that matter during pilots and release reviews.
- Backend request correlation and metrics were already solid; the missing observability pieces were trace context plumbing on the API side and a lightweight frontend error hook.
- CI already covers lint, tests, Playwright, `gosec`, Trivy, packaging, and release publishing. CodeQL was the most obvious missing static-analysis layer.

## Completed In This Pass

- added `make release-surface-check` plus a source-controlled `docs/releases/current.md` reference so version/install drift becomes a release-gate failure instead of a manual review chore
- expanded the real-runtime dashboard smoke to cover the actual evaluator flow: local session, workspace creation, discovery, graph, simulation, plan save, and report export
- added `make web-install`, `make web-e2e-setup`, and `make pilot-smoke` plus stronger `.codex/setup.sh` Node/npm/Playwright bootstrap checks so local release-owner shells have one clear frontend setup path
- upgraded `viaduct doctor` and `viaduct status --runtime` to report config validity, store posture, shared-auth readiness, and concrete `/readyz` degradation reasons such as missing policies or open connector circuits
- added frontend render/runtime error capture hooks and backend `traceparent` to `X-Trace-ID` correlation so operators can integrate external monitoring without stack churn
- added GitHub CodeQL scanning alongside the existing `gosec`, Trivy, Playwright, and release-gate coverage
- published a small evaluator evidence kit and wired the pilot workspace guide to the root-level `make pilot-smoke` path

## Prioritized Tasks

### P1 Next

#### 1. Consolidate release workflow ownership around the Docker-canonical path
- Why it matters: duplicate or legacy release workflows make operators second-guess which artifact path is authoritative.
- Affected files: `.github/workflows/image.yml`, `.github/workflows/release.yml`, `RELEASE.md`, release docs
- Acceptance criteria:
  - one GitHub workflow is clearly documented as the canonical tagged release path
  - any legacy manual workflow is either retired or explicitly marked non-canonical
  - release docs do not point at conflicting commands

### P2 Later

#### 2. Add opt-in backend exporter wiring for OpenTelemetry-compatible collectors
- Why it matters: the current trace correlation is useful, but larger pilots will eventually want spans shipped to Grafana Tempo or another collector without patching Viaduct.
- Affected files: `internal/api/observability.go`, config docs, operations docs
- Acceptance criteria:
  - trace export is opt-in through environment variables
  - the default local path remains lightweight and dependency-free
  - request IDs and trace IDs stay aligned in logs and responses

#### 3. Add screenshot and doc-link validation to the release owner path
- Why it matters: stale screenshots and broken doc links quietly erode evaluator trust even when the binaries are fine.
- Affected files: screenshot automation, docs index/readmes, CI/release checks
- Acceptance criteria:
  - the checked-in README/demo screenshots are regenerated from the current app state
  - broken internal doc links fail validation before release
  - the release checklist points at the same automation
