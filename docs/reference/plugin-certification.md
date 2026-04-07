# Plugin Certification Guide

This guide describes the minimum validation bar for publishing or operating a Viaduct community plugin with confidence.

## Baseline Requirements
- a valid `plugin.json` manifest next to the executable
- a non-empty platform identifier that matches the connector the plugin serves
- `protocol_version` aligned with the Viaduct host contract
- explicit `minimum_viaduct_version` and `maximum_viaduct_version` when you only validate against a bounded host range

## Required Validation
1. Run the manifest validator:

   ```bash
   go run ./scripts/plugin_manifest_check -manifest ./plugin.json -host-version <viaduct-version>
   ```

2. Validate plugin host behavior with automated tests:
   - health check returns `ok` or `healthy`
   - `Platform` is non-empty
   - `Discover` returns a non-nil normalized payload
   - config and secrets passed into `Connect` arrive intact
   - shutdown is safe even after prior failures

3. Confirm the plugin keeps using the universal schema in `internal/models/`.

## Compatibility Expectations
- keep version claims honest; do not advertise host support you have not exercised
- update `minimum_viaduct_version` when you begin relying on newer host features
- treat protocol mismatches and empty-platform responses as hard failures, not soft warnings

## Reference Assets
- [Plugin Author Guide](plugin-author-guide.md)
- [Example Plugin README](../../examples/plugin-example/README.md)
- [Example Plugin Manifest](../../examples/plugin-example/plugin.json)
