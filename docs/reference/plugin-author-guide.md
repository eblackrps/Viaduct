# Plugin And Connector Author Guide

Viaduct supports community connectors through the gRPC-based plugin host in `internal/connectors/plugin/`.

## When To Write A Plugin
- you need a connector that is not part of the built-in set
- you want to iterate on a platform integration without changing core Viaduct
- you need to keep a connector in a separate distribution or lifecycle

## Core Requirements

Implement the `ConnectorPluginServer` contract:
- `Connect`
- `Discover`
- `Platform`
- `Close`
- `Health`

The plugin host expects:
- a non-empty platform identifier
- `Health` to return `ok` or `healthy`
- `Discover` to return a non-nil normalized result
- graceful shutdown behavior
- a `plugin.json` manifest when Viaduct launches the plugin executable directly

## Example Plugin

See:
- [`../../examples/plugin-example/main.go`](../../examples/plugin-example/main.go)
- [`../../examples/plugin-example/plugin.json`](../../examples/plugin-example/plugin.json)
- [`../../examples/plugin-example/README.md`](../../examples/plugin-example/README.md)

## Runtime Model
- Viaduct can start a plugin executable and pass `VIADUCT_PLUGIN_ADDR`
- Viaduct can also connect to a pre-running plugin using a `grpc://host:port` address
- plugins are registered by normalized platform key
- loading a replacement plugin for the same platform replaces the prior process

Example manifest:

```json
{
  "name": "Viaduct Example Plugin",
  "platform": "example",
  "version": "1.0.0",
  "protocol_version": "v1"
}
```

Example config:

```yaml
plugins:
  example: "grpc://127.0.0.1:50071"
```

## Compatibility Rules
- preserve the normalized schema in `internal/models/`
- do not return ad hoc parallel schemas
- keep connector auth and transport logic deterministic and testable
- fail clearly on health, connect, or discover errors
- ensure `Close` can be called even when the process is already unhealthy

## Validation And Certification Checklist
- host load test using `internal/connectors/plugin/host.go`
- manifest validation succeeds with the expected platform and protocol version
- health check returns `ok` or `healthy`
- platform lookup returns a non-empty identifier
- discovery returns normalized VMs and metadata
- shutdown path is safe
- regression tests cover unhealthy plugin, empty platform, config propagation, and nil discovery results

## Developer Tips
- keep fixture payloads close to the connector package
- prefer deterministic mapping helpers over logic embedded in transport code
- use the KVM or example plugin patterns as reference implementations before branching into more complex platforms
