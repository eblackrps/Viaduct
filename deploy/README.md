# Deployment Assets

This directory contains production-oriented container deployment assets for Viaduct.

## Contents

- `docker-compose.prod.yml`: hardened single-host container deployment
- `helm/viaduct/`: Helm chart for Kubernetes installs using the published OCI image

These assets assume the OCI image is the canonical release artifact and that writable state is mounted at `/var/lib/viaduct`.
