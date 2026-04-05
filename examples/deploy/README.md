# Deployment Examples

This directory contains reference deployment assets for evaluating or packaging Viaduct in lab and pilot environments.

## Contents

- `docker-compose.yml`: single-host deployment for API and bundled dashboard assets
- `systemd/viaduct.service`: service unit for Linux package installs
- `kubernetes/`: reference manifests for a basic in-cluster API deployment

## Notes

- These examples are intended for evaluation and controlled internal environments.
- Persistent environments should point `state_store_dsn` at PostgreSQL instead of using the in-memory store.
- The bundled dashboard is built into the release package; the Vite dev server is not part of these deployment examples.
