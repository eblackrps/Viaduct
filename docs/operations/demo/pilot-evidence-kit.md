# Pilot Evidence Kit

Use this checklist when you need one clean evaluator packet from the Viaduct assessment workflow.

The goal is simple: prove that the same signed-in session can discover inventory, inspect it, simulate a plan, save that plan, and export evidence without switching tools or adding manual steps.

## Required Artifacts

Capture these five items from one run of the default local lab or an equivalent seeded pilot:

1. Get started entry point
   Show the `Get started` screen before sign-in, or capture the local-session entry if you are using the loopback `Start local session` path.

2. Discovery evidence
   Capture the assessment after discovery has finished and the workload table is visible.

3. Planning evidence
   Capture the assessment after dependency graph generation and readiness simulation, with the `Save plan` action visible.

4. Saved-plan evidence
   Capture the assessment after the plan has been saved and the `Export report` action is available.

5. Exported report evidence
   Keep the exported Markdown report itself. This is the evaluator handoff artifact because it includes source scope, target assumptions, graph output, readiness, and saved-plan metadata in one document.

## Recommended Supporting Evidence

- `viaduct doctor`
Capture the output when you need to prove config, store, auth status, and runtime readiness from the CLI side.

- `viaduct status --runtime`
  Capture this when you need to show the recorded local runtime URL, PID, and ready versus degraded status.

- Request IDs
  If an assessment step fails, capture the request ID shown in the API response or dashboard error panel along with the assessment or job identifier.

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

That combination gives you the same dashboard path that CI validates for the real `viaduct start` path.

## Reference Assets

- [Assessment workflow](../pilot-workspace-flow.md)
- [Demo kit](README.md)
- [Screenshot assets](screenshots/README.md)
