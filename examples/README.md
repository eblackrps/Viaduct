# Examples

This directory contains the runnable evaluation assets and extension examples that support the current Viaduct workflow.

## Available Examples

- `lab/`: default local evaluation path for `viaduct start`, the WebUI-first workspace flow, and CLI corroboration
- `deploy/`: reference deployment assets for lab and pilot environments
- `plugin-example/`: small gRPC connector plugin that demonstrates the extension model

## Recommended First Run

Start with [lab/README.md](lab/README.md) if you want to evaluate Viaduct without a live hypervisor.

## Related Guides

- Plugin authoring: [../docs/reference/plugin-author-guide.md](../docs/reference/plugin-author-guide.md)
- Plugin certification: [../docs/reference/plugin-certification.md](../docs/reference/plugin-certification.md)
- Deployment references: [deploy/README.md](deploy/README.md)
- Live operator API docs: `/api/v1/docs` once the local runtime is running
