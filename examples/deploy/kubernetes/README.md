# Kubernetes Deployment

These manifests provide a minimal in-cluster Viaduct API deployment for labs, demos, and controlled pilot environments.

## Included Assets

- `configmap.yaml`: mounts a starter `config.yaml`
- `secret.example.yaml`: example secret manifest for the admin API key
- `deployment.yaml`: single-replica API deployment with probes and basic container safeguards
- `service.yaml`: ClusterIP service exposing the HTTP API

## Apply Order

```bash
kubectl apply -f secret.example.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

## Notes

- replace `change-me` in `secret.example.yaml` before applying it outside a throwaway lab
- set `state_store_dsn` in the mounted config if you need persistent state
- the bundled dashboard is not served by these manifests; they focus on the API control plane
- use an ingress or reverse proxy only after you have explicit auth, TLS, and tenant handling in place
