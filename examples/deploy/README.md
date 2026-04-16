# Deployment Examples

This directory contains reference deployment assets for evaluating or packaging Viaduct in lab and pilot environments.

## Contents

- `docker-compose.yml`: single-host deployment for the API and bundled dashboard assets
- `viaduct.env.example`: example environment file for Compose and pilot installs
- `systemd/viaduct.service`: Linux service unit for package installs
- `kubernetes/`: reference manifests for a basic in-cluster API deployment with probes and secret-based admin auth

## Docker Compose

```bash
mkdir -p examples/deploy/config
cp configs/config.example.yaml examples/deploy/config/config.yaml
docker build -t viaduct:latest .
docker compose -f examples/deploy/docker-compose.yml up
```

The Compose stack expects `examples/deploy/config/config.yaml` and reads environment overrides from `examples/deploy/viaduct.env.example`.

When the image includes built dashboard assets, the same HTTP listener serves:
- dashboard shell: `/`
- API routes: `/api/v1/*`
- live API docs: `/api/v1/docs`

For the default local mapping in this example, open [http://localhost:8080](http://localhost:8080).

## systemd

Use `systemd/viaduct.service` as a starting point for package-based installs. The unit expects:
- an extracted bundle layout rooted at `/opt/viaduct`
- the `viaduct` binary at `/opt/viaduct/bin/viaduct`
- a config file at `/etc/viaduct/config.yaml`
- an optional environment file at `/etc/viaduct/viaduct.env`
- persistent writable state under `/var/lib/viaduct`

If the extracted package or install layout also includes `share/viaduct/web`, the same service can expose the dashboard at `/` without a separate static web server.

## Kubernetes

See [kubernetes/README.md](kubernetes/README.md) for apply order and manifest notes.

## Notes

- These examples are intended for evaluation and controlled pilot environments.
- Persistent environments should point `state_store_dsn` at PostgreSQL instead of using the in-memory store.
- The bundled dashboard is built into the release package and served by the same `viaduct serve-api` process when those assets are present.
- Keep `VIADUCT_ALLOWED_ORIGINS` empty for same-origin deployments and set it only for explicit trusted cross-origin dashboards.
- `viaduct start` is the default local operator entrypoint; use `viaduct serve-api` directly in service and container contexts.
- Treat these manifests as references to adapt, not as comprehensive hardening guidance.
