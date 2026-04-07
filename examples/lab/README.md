# Lab Environment

This directory contains a lightweight local evaluation environment for Viaduct.

## Contents
- `kvm/`: KVM/libvirt XML fixtures for local discovery
- `migration-window.yaml`: example migration spec with execution window, approval gate, and wave planning
- `tenant-create.json`: sample tenant creation payload for the admin API, including starter quotas
- `config.yaml`: minimal local config for the lab

## Recommended Flow

```bash
mkdir -p ~/.viaduct
cp examples/lab/config.yaml ~/.viaduct/config.yaml
./bin/viaduct discover --type kvm --source examples/lab/kvm --save
./bin/viaduct plan --spec examples/lab/migration-window.yaml
./bin/viaduct serve-api --port 8080
```

Then launch the dashboard:

```bash
cd web
npm ci
npm run dev
```
