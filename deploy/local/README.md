# Local Docker Config

`config.yaml` is the keyless configuration used by the repo-root `compose.yaml`.

It is intentionally scoped to localhost evaluation:

- no tenant key
- no service-account key
- no admin key
- KVM source pointed at the lab fixtures bundled into the Viaduct image

Start it from the repo root with:

```bash
docker compose up -d --build
```

Then open `http://127.0.0.1:8080`.
