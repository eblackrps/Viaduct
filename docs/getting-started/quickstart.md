# Quickstart

This quickstart uses the local KVM fixture lab so you can evaluate Viaduct end to end without a live hypervisor.

## Prerequisites
- Go 1.24+
- Node.js 20.19+
- `make` if you want the convenience targets

## 1. Build Viaduct

```bash
make build
./bin/viaduct version
```

On Windows without `make`:

```powershell
go build -ldflags "-X main.version=dev -X main.commit=none -X main.date=unknown" -o bin/viaduct.exe ./cmd/viaduct
.\bin\viaduct.exe version
```

## 2. Create A Local Config

```bash
mkdir -p ~/.viaduct
cp configs/config.example.yaml ~/.viaduct/config.yaml
```

For the local lab, the only required source is the built-in KVM fixture path:

```yaml
sources:
  kvm:
    address: "examples/lab/kvm"
```

## 3. Run Discovery

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
```

This loads the sample XML fixtures, normalizes them into the universal schema, and saves a snapshot to the configured state store.

## 4. Inspect A Migration Spec

```bash
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

This validates the example spec and shows the execution window, approval gate, and wave-planning shape that Viaduct will use.

## 5. Start The API And Dashboard

In one terminal:

```bash
./bin/viaduct serve-api --port 8080
```

In another terminal:

```bash
cd web
npm ci
npm run dev
```

Open the dashboard in your browser at the Vite URL shown in the terminal. The dashboard will proxy API calls to `http://localhost:8080`.

For local lab use, the default tenant may work without explicit credentials. For any real tenant-scoped dashboard usage, prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY` over `VITE_VIADUCT_API_KEY` so operator activity is attributable to a named service account instead of the tenant-wide admin credential.

## 6. Explore Operator Flows
- Inventory and dependency views: confirm the lab workloads appear in the dashboard.
- Lifecycle views: review cost, policy, drift, and remediation panels.
- Migration planning: use the migration wizard to run preflight, create a migration, and inspect checkpoints.
- Tenant reporting and trust context: call `/api/v1/tenants/current` and `/api/v1/summary` with a service-account key if you want to validate multi-tenant flows without falling back to the tenant-wide admin credential.

## Next Steps
- Installation details: [installation.md](installation.md)
- Configuration reference: [../reference/configuration.md](../reference/configuration.md)
- Migration operations guide: [../operations/migration-operations.md](../operations/migration-operations.md)
- Auth, role, and auditability model: [../operations/auth-role-audit-model.md](../operations/auth-role-audit-model.md)
- Lab assets: [../../examples/lab/README.md](../../examples/lab/README.md)
