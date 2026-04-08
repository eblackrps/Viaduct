import type { MigrationMeta } from "../types";
import { StatusBadge, type StatusTone } from "./primitives/StatusBadge";
import {
  describeMigrationPhase,
  getPersistedMigrationListPresentation,
  getPersistedMigrationListStatus,
} from "../features/migrations/migrationStatus";

interface MigrationHistoryProps {
  migrations: MigrationMeta[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
}

export function MigrationHistory({
  migrations,
  loading = false,
  error,
  emptyMessage = "No migrations have been recorded yet.",
}: MigrationHistoryProps) {
  return (
    <section className="panel p-5">
      <p className="font-display text-2xl text-ink">Migration History</p>
      <p className="mt-1 text-sm text-slate-500">Track completed, failed, and in-flight migrations from the shared state store.</p>

      {loading && migrations.length === 0 && <p className="mt-5 text-sm text-slate-500">Loading migration history…</p>}
      {error && !loading && <p className="mt-5 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

      <div className="mt-5 space-y-3">
        {migrations.map((migration) => {
          const status = getPersistedMigrationListPresentation(getPersistedMigrationListStatus(migration.phase));

          return (
            <article key={migration.id} className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold text-ink">{migration.spec_name}</p>
                  <p className="text-slate-500">{migration.id}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
                  <StatusBadge tone={phaseTone(migration.phase)}>{describeMigrationPhase(migration.phase)}</StatusBadge>
                </div>
              </div>
              <p className="mt-3 text-slate-500">Started {new Date(migration.started_at).toLocaleString()}</p>
              <p className="mt-1 text-slate-500">
                {migration.completed_at ? `Completed ${new Date(migration.completed_at).toLocaleString()}` : `Updated ${new Date(migration.updated_at).toLocaleString()}`}
              </p>
            </article>
          );
        })}

        {!loading && migrations.length === 0 && !error && <p className="text-sm text-slate-500">{emptyMessage}</p>}
      </div>
    </section>
  );
}

function phaseTone(phase: MigrationMeta["phase"]): StatusTone {
  switch (phase) {
    case "complete":
      return "success";
    case "failed":
      return "danger";
    case "rolled_back":
      return "neutral";
    default:
      return "accent";
  }
}
