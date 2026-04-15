import { useEffect, useMemo, useState } from "react";
import { RefreshCcw } from "lucide-react";
import type { MigrationPhase, MigrationState } from "../types";

interface MigrationProgressProps {
  state: MigrationState | null;
  onRollback?: () => void;
}

const phaseOrder: MigrationPhase[] = [
  "plan",
  "export",
  "convert",
  "import",
  "configure",
  "verify",
  "complete",
];

export function MigrationProgress({ state, onRollback }: MigrationProgressProps) {
  const [elapsedLabel, setElapsedLabel] = useState("0s");

  useEffect(() => {
    if (!state) {
      return;
    }

    const update = () => {
      const started = new Date(state.started_at).getTime();
      const ended = state.completed_at ? new Date(state.completed_at).getTime() : Date.now();
      const seconds = Math.max(0, Math.round((ended - started) / 1000));
      setElapsedLabel(`${seconds}s`);
    };

    update();
    const interval = window.setInterval(update, 1000);
    return () => window.clearInterval(interval);
  }, [state]);

  const percentage = useMemo(() => {
    if (!state || state.workloads.length === 0) {
      return 0;
    }

    const scores = state.workloads.map((workload) => {
      if (workload.phase === "complete") {
        return 100;
      }
      if (workload.phase === "failed") {
        return 0;
      }
      const index = Math.max(0, phaseOrder.indexOf(workload.phase));
      return Math.round((index / (phaseOrder.length - 1)) * 100);
    });

    return Math.round(scores.reduce((total, score) => total + score, 0) / scores.length);
  }, [state]);

  if (!state) {
    return (
      <div className="rounded-2xl border border-dashed border-slate-300 px-4 py-8 text-sm text-slate-500">
        Start a migration to see per-VM progress.
      </div>
    );
  }

  return (
    <section className="space-y-5">
      <div className="panel p-5">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="font-display text-2xl text-ink">Migration Progress</p>
            <p className="mt-1 text-sm text-slate-500">Phase: <span className="font-semibold text-ink">{state.phase}</span> · Duration: {elapsedLabel}</p>
            {state.pending_approval && <p className="mt-2 text-sm font-semibold text-amber-700">Execution is waiting on approval.</p>}
          </div>
          {state.phase === "failed" && onRollback && (
            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-full bg-rose-600 px-4 py-2 text-sm font-semibold text-white"
              onClick={onRollback}
            >
              <RefreshCcw className="h-4 w-4" />
              Rollback
            </button>
          )}
        </div>

        <div className="mt-5 h-3 overflow-hidden rounded-full bg-slate-200">
          <div className="h-full rounded-full bg-accent transition-all" style={{ width: `${percentage}%` }} />
        </div>
        <p className="mt-2 text-sm text-slate-500">{percentage}% complete</p>

        {state.errors && state.errors.length > 0 && (
          <div className="mt-4 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
            {state.errors.map((error, index) => (
              <p key={`${state.id}-error-${index}`}>{error}</p>
            ))}
          </div>
        )}

        {state.checkpoints && state.checkpoints.length > 0 && (
          <div className="mt-4 grid gap-3 md:grid-cols-3">
            {state.checkpoints.map((checkpoint) => (
              <div key={checkpoint.phase} className="rounded-2xl bg-slate-50 px-4 py-3 text-sm">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-semibold text-ink">{checkpoint.phase}</p>
                  <span className={`rounded-full px-3 py-1 text-xs font-semibold ${checkpointClass(checkpoint.status)}`}>{checkpoint.status}</span>
                </div>
                {checkpoint.message && <p className="mt-2 text-slate-500">{checkpoint.message}</p>}
              </div>
            ))}
          </div>
        )}

        {state.plan && state.plan.waves.length > 0 && (
        <div className="mt-4 rounded-2xl bg-slate-50 p-4 text-sm text-slate-600">
            <p className="font-semibold text-ink">Runbook</p>
            <p className="mt-1">Planned across {state.plan.waves.length} wave(s) with batch size {state.plan.wave_strategy.size ?? state.plan.waves.length}.</p>
          </div>
        )}
      </div>

      <div className="space-y-3">
        {state.workloads.map((workload) => (
          <article key={workload.vm.id || workload.vm.name} className="panel p-4">
            <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div>
                <p className="font-semibold text-ink">{workload.vm.name}</p>
                <p className="text-sm text-slate-500">{workload.vm.platform} → {workload.target_vm_id || "pending target VM"}</p>
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-semibold ${phaseClass(workload.phase)}`}>{workload.phase}</span>
            </div>
            {workload.error && <p className="mt-3 text-sm text-rose-700">{workload.error}</p>}
          </article>
        ))}
      </div>
    </section>
  );
}

function phaseClass(phase: MigrationPhase) {
  switch (phase) {
    case "plan":
      return "bg-sky-100 text-sky-700";
    case "export":
      return "bg-amber-100 text-amber-700";
    case "convert":
      return "bg-orange-100 text-orange-700";
    case "import":
      return "bg-violet-100 text-violet-700";
    case "configure":
      return "bg-teal-100 text-teal-700";
    case "verify":
      return "bg-emerald-100 text-emerald-700";
    case "complete":
      return "bg-emerald-100 text-emerald-700";
    case "failed":
      return "bg-rose-100 text-rose-700";
    case "rolled_back":
      return "bg-slate-200 text-slate-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

function checkpointClass(status: "pending" | "running" | "completed" | "failed") {
  switch (status) {
    case "running":
      return "bg-sky-100 text-sky-700";
    case "completed":
      return "bg-emerald-100 text-emerald-700";
    case "failed":
      return "bg-rose-100 text-rose-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}
