import { ArrowRight } from "lucide-react";
import { SectionCard } from "../../components/primitives/SectionCard";
import type { InventoryPlanningDraft } from "../inventory/inventoryPlanningDraft";
import {
  MigrationExecuteStage,
  MigrationMetric,
  MigrationPrepareStage,
  MigrationScopeStage,
  MigrationValidateStage,
} from "./MigrationStagePanels";
import {
  getMigrationWorkflowStatus,
  getPreflightSummary,
  getWorkflowStatusPresentation,
} from "./migrationStatus";
import { useMigrationWorkspace } from "./useMigrationWorkspace";

const stages = [
  ["Scope", "Bring workloads into the plan."],
  ["Prepare", "Set target and execution controls."],
  ["Validate", "Run preflight and save plan state."],
  ["Execute", "Start, resume, or roll back the saved plan."],
] as const;

interface MigrationWorkflowProps {
  planningDraft?: InventoryPlanningDraft | null;
  onPlanningDraftCleared?: () => void;
  onMigrationChange?: () => void;
}

export function MigrationWorkflow({
  planningDraft,
  onPlanningDraftCleared,
  onMigrationChange,
}: MigrationWorkflowProps) {
  const workspace = useMigrationWorkspace({
    planningDraft,
    onPlanningDraftCleared,
    onMigrationChange,
  });

  return (
    <section className="space-y-5">
      <SectionCard title="Planning workflow" description="Use the real backend sequence: draft, validate, save plan state, then execute.">
        <div className="grid gap-3 xl:grid-cols-4">
          <MigrationMetric
            label="State"
            value={workspace.workflowPresentation.label}
            tone={workspace.workflowPresentation.tone}
            detail={workspace.workflowPresentation.description}
          />
          <MigrationMetric
            label="Scope"
            value={workspace.selectedWorkloads.length > 0 ? `${workspace.selectedWorkloads.length} selected` : "No scope"}
            tone={workspace.selectedWorkloads.length > 0 ? "info" : "neutral"}
            detail={
              workspace.selectedWorkloads.length > 0
                ? `${workspace.sourceNetworks.length} source network(s)`
                : "Load inventory or start from inventory selection."
            }
          />
          <MigrationMetric
            label="Preflight"
            value={workspace.preflight ? `${workspace.preflight.pass_count}/${workspace.preflight.checks.length} passed` : "Not run"}
            tone={
              !workspace.preflight
                ? "neutral"
                : workspace.preflightStale
                  ? "warning"
                  : workspace.preflight.fail_count > 0
                    ? "danger"
                    : workspace.preflight.warn_count > 0
                      ? "warning"
                      : "success"
            }
            detail={getPreflightSummary(workspace.preflight, workspace.preflightStale)}
          />
          <MigrationMetric
            label="Saved plan"
            value={workspace.migrationState ? (workspace.planStale ? "Plan stale" : workspace.migrationState.spec_name) : "None"}
            tone={
              workspace.migrationState
                ? workspace.planStale
                  ? "danger"
                  : getWorkflowStatusPresentation(getMigrationWorkflowStatus(workspace.migrationState)).tone
                : "neutral"
            }
            detail={workspace.migrationState ? workspace.migrationState.id : "Persist a plan before execution."}
          />
        </div>
      </SectionCard>

      <div className="panel p-5">
        <div className="flex flex-wrap gap-3">
          {stages.map(([label, description], index) => (
            <button
              key={label}
              type="button"
              onClick={() => workspace.setStage(index)}
          className={`min-w-[170px] rounded-2xl border px-4 py-4 text-left transition ${
                workspace.stage === index
                  ? "border-ink bg-ink text-white"
                  : index < workspace.stage
                    ? "border-sky-200 bg-sky-50 text-sky-950"
                    : "border-slate-200 bg-white text-slate-700"
              }`}
            >
              <p className="text-xs uppercase tracking-[0.18em] opacity-70">{index + 1}</p>
              <p className="mt-2 font-semibold">{label}</p>
              <p className="mt-2 text-sm opacity-80">{description}</p>
            </button>
          ))}
        </div>

        {workspace.error && <p className="mt-5 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{workspace.error}</p>}

        {workspace.stage === 0 && <MigrationScopeStage workspace={workspace} />}
        {workspace.stage === 1 && <MigrationPrepareStage workspace={workspace} />}
        {workspace.stage === 2 && <MigrationValidateStage workspace={workspace} />}
        {workspace.stage === 3 && <MigrationExecuteStage workspace={workspace} />}

        <div className="mt-6 flex flex-wrap items-center justify-between gap-3">
          <button
            type="button"
            className="rounded-full border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:bg-slate-100"
            onClick={() => workspace.setStage((current) => Math.max(0, current - 1))}
            disabled={workspace.stage === 0}
          >
            Back
          </button>
          {workspace.stage < stages.length - 1 && (
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:bg-slate-300"
              onClick={() => workspace.setStage((current) => Math.min(stages.length - 1, current + 1))}
              disabled={(workspace.stage === 0 && workspace.selectedWorkloads.length === 0) || (workspace.stage === 2 && !workspace.migrationState)}
            >
              Continue
              <ArrowRight className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    </section>
  );
}
