# Kubernetes Deployment

These manifests provide a minimal in-cluster Viaduct API deployment for labs, demos, and controlled pilot environments.

## Included Assets

- `configmap.yaml`: mounts a starter `config.yaml`
- `secret.example.yaml`: example secret manifest for the admin API key
- `deployment.yaml`: single-replica API deployment with probes and basic container safeguards
- `service.yaml`: ClusterIP service exposing the HTTP listener for both the dashboard shell and API routes

## Apply Order

```bash
kubectl apply -f secret.example.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## Notes

- replace the placeholder in `secret.example.yaml` with the stored `sha256:<hex>` admin-key digest before applying it outside a throwaway lab
- set `state_store_dsn` in the mounted config if you need persistent state
- the stock image serves the bundled dashboard at `/` and the API under `/api/v1/` on the same listener
- the same service also exposes live API docs at `/api/v1/docs`
- use an ingress or reverse proxy only after you have explicit auth, TLS, and tenant handling in place
