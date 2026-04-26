# Pilot Evidence Kit

Use this checklist when you need one clean evaluator packet from the workspace-first Viaduct flow.

The goal is simple: prove that the same operator session can discover inventory, inspect it, simulate a plan, save that plan, and export evidence without switching tools or inventing extra manual steps.

## Required Artifacts

Capture these five items from one run of the default local lab or an equivalent seeded pilot:

1. Get started entry point
   Show the `Get started` screen before sign-in, or capture the local-session entry if you are using the loopback `Start local session` path.

2. Discovery evidence
   Capture the workspace after discovery has finished and the workload assessment table is visible.

3. Planning evidence
   Capture the workspace after dependency graph generation and readiness simulation, with the `Save plan` action visible.

4. Saved-plan evidence
   Capture the workspace after the plan has been saved and the `Export report` action is available.

5. Exported report evidence
   Keep the exported Markdown report itself. This is the evaluator handoff artifact because it includes source scope, target assumptions, graph output, readiness, and saved-plan metadata in one document.

## Recommended Supporting Evidence

- `viaduct doctor`
Capture the output when you need to prove config, store, auth status, and runtime readiness from the CLI side.

- `viaduct status --runtime`
  Capture this when you need to show the recorded local runtime URL, PID, and ready versus degraded status.

- Request IDs
  If a workspace step fails, capture the request ID shown in the API response or dashboard error panel along with the workspace or job identifier.

## Suggested File Names

Use one folder per evaluation run and keep the artifacts predictable:

- `01-get-started.png`
- `02-discovery.png`
- `03-simulation.png`
- `04-saved-plan.png`
- `05-pilot-report.md`
- `doctor.txt`
- `status-runtime.txt`

## Fastest Local Path

From the repo root:

```bash
make build
make web-build
./bin/viaduct start
```

If you want the browser smoke prerequisites on a fresh shell:

```bash
make web-e2e-setup
make pilot-smoke
```

That combination gives you the same operator story that CI validates for the real `viaduct start` path.

## Reference Assets

- [Pilot workspace flow](../pilot-workspace-flow.md)
- [Demo kit](README.md)
- [Screenshot assets](screenshots/README.md)
