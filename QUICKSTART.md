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
```

## 6. Start The Dashboard

```bash
cd web
npm ci
npm run dev
```

The dashboard expects the API at `/api` and can use `VITE_VIADUCT_API_KEY` from [web/.env.example](web/.env.example) when tenant-scoped access is enabled.

## Next Steps
- Review [docs/operations/migration-operations.md](docs/operations/migration-operations.md) for execution and rollback workflows.
- Review [docs/operations/backup-portability.md](docs/operations/backup-portability.md) for Veeam portability guidance.
- Review [docs/operations/multi-tenancy.md](docs/operations/multi-tenancy.md) before enabling shared-tenant operation.
