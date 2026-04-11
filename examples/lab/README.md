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
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
export VIADUCT_ADMIN_KEY=lab-admin
./bin/viaduct serve-api --port 8080
```

Seed the lab tenant and service account:

```bash
curl -X POST \
  -H "X-Admin-Key: lab-admin" \
  -H "Content-Type: application/json" \
  --data @examples/lab/tenant-create.json \
  http://localhost:8080/api/v1/admin/tenants

curl -X POST \
  -H "X-API-Key: lab-tenant-key" \
  -H "Content-Type: application/json" \
  --data @examples/lab/service-account-create.json \
  http://localhost:8080/api/v1/service-accounts
```

Then launch the dashboard and sign in with `lab-operator-key`:

```bash
cd web
npm ci
npm run dev
```

The bootstrap screen stores the key in session storage by default. Use the remember option only when you want the browser to retain the key across restarts.

The default dashboard sequence is:

1. create workspace
2. discover
3. inspect
4. simulate
5. save plan
6. export report

When you are finished evaluating the flow, you can delete the workspace from the dashboard. That removes the workspace record and its job history without deleting the underlying snapshots or saved migration records outside the workspace document.

## Visual Reference

Local lab screenshots are in [screenshots/README.md](screenshots/README.md). The broader workspace-first demo assets also live in [../../docs/operations/demo/screenshots/README.md](../../docs/operations/demo/screenshots/README.md).

## CLI Corroboration

If you want to exercise the same fixture set through the CLI:

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

Focused smoke coverage for this path lives in `tests/integration/pilot_workspace_smoke_test.go`.
