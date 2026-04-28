# Deployment Assets

This directory contains container deployment assets for Viaduct.

## Contents

- `../compose.yaml`: keyless local Docker lab from the repo root
- `local/config.yaml`: keyless local Docker lab config used by `../compose.yaml`
- `docker-compose.prod.yml`: hardened single-host production container deployment
- `helm/viaduct/`: Helm chart for Kubernetes installs using the published OCI image

The root Compose path is for localhost evaluation only. Production assets assume the OCI image is the primary packaged release artifact and that writable state is mounted at `/var/lib/viaduct`.

## Observability

- `observability/docker-compose.yml`: local Grafana + Tempo stack for backend trace validation
- `observability/tempo.yaml`: Tempo OTLP ingest and local trace storage config

Use that stack for local backend instrumentation checks, not as a production monitoring deployment. Start it with `make observability-up`, stop it with `make observability-down`, and validate it with `make observability-validate`.
