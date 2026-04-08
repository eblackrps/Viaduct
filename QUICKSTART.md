# Quickstart

This is the fastest way to evaluate Viaduct from source without a live hypervisor.

## 1. Build The CLI

```bash
go mod tidy
make build
./bin/viaduct version
```

## 2. Seed A Local Config

```bash
mkdir -p ~/.viaduct
cp configs/config.example.yaml ~/.viaduct/config.yaml
```

The sample config already points the KVM source at the local lab fixtures.

## 3. Run Local Discovery

```bash
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
```

## 4. Validate A Migration Spec

```bash
./bin/viaduct plan --spec examples/lab/migration-window.yaml
```

## 5. Start The API

```bash
./bin/viaduct serve-api --port 8080
curl http://localhost:8080/api/v1/about
```

## 6. Start The Dashboard

```bash
cd web
npm ci
npm run dev
```

The dashboard expects the API at `/api` and can use `VITE_VIADUCT_API_KEY` from [web/.env.example](web/.env.example) when tenant-scoped access is enabled.

For normal dashboard and pilot use, prefer `VITE_VIADUCT_SERVICE_ACCOUNT_KEY`. Keep `VITE_VIADUCT_API_KEY` for tenant bootstrap or break-glass admin access.

If you create additional tenants or service accounts, verify the active identity with:

```bash
curl -H "X-Service-Account-Key: <service-account-key>" http://localhost:8080/api/v1/tenants/current
```

Use the tenant key only when you are bootstrapping the tenant or intentionally using tenant-admin access:

```bash
curl -H "X-API-Key: <tenant-key>" http://localhost:8080/api/v1/tenants/current
```

## Next Steps
- Review [docs/operations/migration-operations.md](docs/operations/migration-operations.md) for execution and rollback workflows.
- Review [docs/operations/backup-portability.md](docs/operations/backup-portability.md) for Veeam portability guidance.
- Review [docs/operations/multi-tenancy.md](docs/operations/multi-tenancy.md) before enabling shared-tenant operation.
- Review [docs/operations/auth-role-audit-model.md](docs/operations/auth-role-audit-model.md) for the early-product trust model and attribution boundaries.
- Review [examples/deploy/README.md](examples/deploy/README.md) if you want to move from local evaluation into a packaged pilot environment.
- Review [examples/plugin-example/README.md](examples/plugin-example/README.md) if you want to validate plugin-backed connector loading.
