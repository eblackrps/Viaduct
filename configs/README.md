# Configurations

This directory contains runnable sample configuration and policy assets for Viaduct.

## Files
- `config.example.yaml`: baseline CLI and API configuration example, including reusable `credential_ref` entries
- `example-migration.yaml`: fuller migration spec covering source, target, selectors, and mappings
- `example-migration-minimal.yaml`: smaller migration spec for focused testing

## Subdirectories
- `cost-profiles/`: sample lifecycle cost profiles
- `policies/`: sample lifecycle and compliance policy definitions

## Usage Notes
- Treat these files as examples, not defaults for production.
- Replace placeholder credentials and prefer environment variables or external secret management for sensitive values.
- Keep samples parseable and close to real operator usage so docs and tests can rely on them.
