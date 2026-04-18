# Lab Environment

This directory contains the default local evaluation path for Viaduct.

## Contents

- `kvm/`: KVM/libvirt XML fixtures for local discovery
- `migration-window.yaml`: example migration spec with execution window, approval requirement, and wave planning
- `tenant-create.json`: sample tenant creation payload for the admin API
- `service-account-create.json`: deterministic operator service-account payload for the dashboard bootstrap flow
- `pilot-workspace-create.json`: seeded pilot workspace intake payload for the workspace APIs
- `config.yaml`: minimal local config for the lab

## Recommended Flow

```bash
make build
make web-build
./bin/viaduct start
```

On a fresh source checkout, `viaduct start` generates `~/.viaduct/config.yaml` automatically when it is missing and points it at the fixtures in this directory.

Then open [http://127.0.0.1:8080](http://127.0.0.1:8080). For the default local lab path, the bootstrap screen offers `Use local operator session`, so no pasted browser key is required.
Live operator API docs are also available at [http://127.0.0.1:8080/api/v1/docs](http://127.0.0.1:8080/api/v1/docs).

If you intentionally want to exercise the tenant-scoped bootstrap path, use the seeded payloads in this directory:
- `tenant-create.json`
- `service-account-create.json`

The bootstrap screen now creates a server-backed session. The browser stores only an opaque session marker, while any tenant or service-account key stays server-side for that session instead of landing in browser storage. Local operator sessions do not use an API key at all. Use the remember option only when you want that non-sensitive marker to survive browser restarts on a trusted workstation.

If you are actively editing the dashboard, you can still run `npm run dev` inside `web/` and use the Vite server instead. The default operator path for the lab is the same-origin shell served by `viaduct start`.

The default dashboard sequence is:

1. create workspace
2. discover
3. inspect
4. simulate
5. save plan
6. export report

When you are finished evaluating the flow, you can delete the workspace from the dashboard. That removes the workspace record and its job history without deleting the underlying snapshots or saved migration records outside the workspace document.

## Visual Reference

Current local-lab screenshots are in [screenshots/README.md](screenshots/README.md). The broader workspace-first demo gallery used by the root README also lives in [../../docs/operations/demo/screenshots/README.md](../../docs/operations/demo/screenshots/README.md).

## CLI Corroboration

If you want to exercise the same fixture set through the CLI:

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

Focused smoke coverage for this path lives in `tests/integration/pilot_workspace_smoke_test.go`.
