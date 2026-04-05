import type { MigrationMeta } from "../types";

interface MigrationHistoryProps {
  migrations: MigrationMeta[];
}

export function MigrationHistory({ migrations }: MigrationHistoryProps) {
  return (
    <section className="panel p-5">
      <p className="font-display text-2xl text-ink">Migration History</p>
      <p className="mt-1 text-sm text-slate-500">Track completed, failed, and in-flight migrations from the shared state store.</p>

      <div className="mt-5 space-y-3">
        {migrations.map((migration) => (
          <article key={migration.id} className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="font-semibold text-ink">{migration.spec_name}</p>
                <p className="text-slate-500">{migration.id}</p>
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-semibold ${phaseClass(migration.phase)}`}>{migration.phase}</span>
            </div>
            <p className="mt-3 text-slate-500">Started {new Date(migration.started_at).toLocaleString()}</p>
            <p className="mt-1 text-slate-500">
              {migration.completed_at ? `Completed ${new Date(migration.completed_at).toLocaleString()}` : `Updated ${new Date(migration.updated_at).toLocaleString()}`}
            </p>
          </article>
        ))}

        {migrations.length === 0 && <p className="text-sm text-slate-500">No migrations have been recorded yet.</p>}
      </div>
    </section>
  );
}

function phaseClass(phase: MigrationMeta["phase"]) {
  switch (phase) {
    case "complete":
      return "bg-emerald-100 text-emerald-700";
    case "failed":
      return "bg-rose-100 text-rose-700";
    case "rolled_back":
      return "bg-slate-200 text-slate-700";
    default:
      return "bg-ink text-white";
  }
}
