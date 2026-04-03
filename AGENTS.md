# Viaduct
Hypervisor-agnostic workload migration and lifecycle management platform.
## Project Identity
- Repo: github.com/eblackrps/viaduct
- License: Apache 2.0
- Language: Go 1.22+ (core engine, CLI), Python 3.12+ (SDK), TypeScript/React (dashboard)
- Author: Eric Black (eblackrps)
## Architecture
Four layers, each a separate Go package under internal/:
1. Discovery Engine (internal/discovery/) - Connects to hypervisor APIs, pulls normalized inventory into a universal schema.
2. Dependency Mapper (internal/deps/) - Directed graph correlating VMs with network flows, DNS, backup jobs, storage.
3. Migration Orchestrator (internal/migrate/) - Declarative YAML-driven workload movement. Idempotent and reversible.
4. Lifecycle Manager (internal/lifecycle/) - Drift detection, cost modeling, policy engine, backup job portability.
## Connector Plugin Model
Each connector lives in internal/connectors/<platform>/ and implements the Connector interface in internal/connectors/connector.go.
Connectors: vmware/ (govmomi), proxmox/ (REST), hyperv/ (WMI), kvm/ (libvirt), nutanix/ (Prism v3).
## Code Standards
- Run golangci-lint before committing. Config in .golangci.yml.
- All exported functions require doc comments.
- Wrap errors: fmt.Errorf("context: %w", err). Never swallow errors.
- No panic() in library code.
- context.Context as first param for any I/O function.
- Table-driven tests. Name pattern: TestFunction_Scenario_Expected.
- No global mutable state. Pass dependencies explicitly.
## CLI (cobra)
Root command: viaduct. Subcommands: discover, plan, migrate, status, rollback, version.
Each subcommand in its own file under cmd/viaduct/.
## Build Commands
make build - Build binary to bin/viaduct
make test - go test ./... -v -race
make lint - golangci-lint run ./...
make all - lint + test + build
## Commit Style
Conventional commits: type(scope): description
Types: feat, fix, docs, test, refactor, ci, chore
Scopes: cli, discovery, vmware, proxmox, hyperv, kvm, nutanix, migrate, deps, lifecycle, store, dashboard, proto, ci
## Constraints
- Never hardcode credentials. Use env vars or config files in .gitignore.
- Never commit .env, secrets, or API keys.
- All discovery-phase API calls are read-only.
- Universal schema in internal/models/ is the single source of truth.
- YAML migration specs are declarative, never imperative.
