# Plugin Example

This example starts a community connector plugin that serves a tiny static inventory over gRPC.

## Run It Directly

```bash
go run ./examples/plugin-example
```

By default it listens on `127.0.0.1:50071`. To let Viaduct launch it dynamically, the plugin host will set `VIADUCT_PLUGIN_ADDR`.

## Build It

```bash
go build ./examples/plugin-example
```

Keep `plugin.json` next to the built executable when Viaduct launches the plugin directly. The example manifest is included in this directory.

## Register It

```yaml
plugins:
  example: "grpc://127.0.0.1:50071"
```

## What It Demonstrates
- implementing `ConnectorPluginServer`
- health reporting
- platform lookup
- manifest-based compatibility metadata
- optional host-version compatibility markers
- returning normalized discovery results
- clean plugin shutdown
- receiving connector config from Viaduct during `Connect`

## Validation Tips
- start with the plugin author guide in [`../../docs/reference/plugin-author-guide.md`](../../docs/reference/plugin-author-guide.md)
- use the host behavior in `internal/connectors/plugin/host.go` as the compatibility contract
- ensure your plugin returns a non-empty platform and a non-nil discovery result
