# Architecture Overview

Viaduct is an API-first, hypervisor-agnostic control plane for workload discovery, migration, and lifecycle management across heterogeneous virtualization estates.

## Design Goals
- normalize inventory from multiple platforms into one schema
- model dependencies before migration work begins
- make migrations declarative, repeatable, resumable, and reversible
- preserve tenant isolation and plugin safety as first-class concerns
- keep backend, CLI, dashboard, and automation flows aligned around the same persisted state

## Core Layers

### 1. Discovery Engine

The discovery layer connects to hypervisor management planes and pulls raw inventory into Viaduct's universal schema. It is responsible for VMs, networks, storage, hosts, snapshots, metadata, and platform-specific references.

### 2. Dependency Mapper

The dependency layer correlates infrastructure inventory with network flow, DNS, storage, and backup information to expose migration complexity and workload relationships.

### 3. Migration Orchestrator

The migration layer converts declarative specifications into executable steps, including planning, validation, disk movement, network remapping, warm replication, cutover, verification, checkpoints, resume support, and rollback.

### 4. Lifecycle Manager

The lifecycle layer handles continuous operations after migration, including drift detection, cost modeling, policy checks, simulation, remediation guidance, and backup portability planning.

## Key Runtime Surfaces

### CLI

The Cobra CLI in `cmd/viaduct/` exposes discovery, planning, migration, status, rollback, version, and local API serving workflows. It is the fastest way to evaluate the backend from source.

### API Server

`internal/api/server.go` serves REST endpoints for inventory, snapshots, graph views, preflight, migrations, lifecycle analysis, remediation, simulation, audit exports, reporting, metrics, and tenant administration. Tenant and admin access are enforced with explicit API key middleware.

### Dashboard

The React dashboard in `web/` consumes the API and provides inventory, migration, dependency, cost, policy, drift, remediation, and migration-history views. Vite proxies `/api` requests to the local API server during development.

### Store

`internal/store/` provides in-memory and PostgreSQL persistence for snapshots, migrations, recovery points, and tenants. PostgreSQL is the recommended backend for any non-demo environment.

## Connector Model

Each platform connector lives under `internal/connectors/<platform>/` and implements the shared `Connector` interface. This keeps the CLI, API, and orchestration layers insulated from platform-specific SDKs and transport details.

Current built-in connectors:
- VMware vSphere via `govmomi`
- Proxmox VE via REST
- Microsoft Hyper-V via WinRM / PowerShell
- KVM via XML fixtures or libvirt behind the `libvirt` build tag
- Nutanix AHV via Prism Central v3
- Veeam Backup & Replication for backup discovery and portability workflows

Community connectors can also be hosted through the gRPC-based plugin system in `internal/connectors/plugin/`.

## Source Of Truth

The universal schema in `internal/models/` is the canonical representation of discovered workloads. Connector implementations may expose platform-native details, but Viaduct's internal coordination should always flow through the normalized model.

## Release Model

Viaduct now treats `make release-gate` as the canonical release-quality check and `make package-release-matrix` as the canonical packaging path. Release bundles include the CLI binary, built web assets, install scripts, docs, configs, examples, deployment references, a manifest, and checksums.
