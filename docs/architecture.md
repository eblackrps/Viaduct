# Architecture Overview

Viaduct is an API-first, hypervisor-agnostic control plane for workload discovery, migration, and lifecycle management across heterogeneous virtualization estates.

## Design Goals
- normalize inventory from multiple platforms into one schema
- model dependencies before migration work begins
- make migrations declarative, repeatable, and reversible
- treat mixed-platform operations as a steady-state reality

## Core Layers

### 1. Discovery Engine

The discovery layer connects to hypervisor management planes and pulls raw inventory into Viaduct's universal schema. It is responsible for VMs, networks, storage, hosts, snapshots, metadata, and platform-specific references.

### 2. Dependency Mapper

The dependency layer correlates infrastructure inventory with network flow, DNS, storage, and backup information to expose migration complexity and workload relationships.

### 3. Migration Orchestrator

The migration layer converts declarative specifications into executable steps, including validation, disk movement, network remapping, cutover, and rollback.

### 4. Lifecycle Manager

The lifecycle layer handles continuous operations after migration, including drift detection, cost modeling, policy checks, and backup portability.

## Connector Model

Each platform connector lives under `internal/connectors/<platform>/` and implements the shared `Connector` interface. This keeps the CLI and orchestration layers insulated from platform-specific SDKs and transport details.

Current connector targets:
- VMware vSphere via `govmomi`
- Proxmox VE via REST
- Microsoft Hyper-V via WMI
- KVM/libvirt
- Nutanix AHV via Prism

## Source of Truth

The universal schema in `internal/models/` is the canonical representation of discovered workloads. Connector implementations may expose platform-native details, but Viaduct's internal coordination should always flow through the normalized model.
