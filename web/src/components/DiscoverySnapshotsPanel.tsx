import type { SnapshotMeta } from "../types";

interface DiscoverySnapshotsPanelProps {
  snapshots: SnapshotMeta[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
}

export function DiscoverySnapshotsPanel({
  snapshots,
  loading = false,
  error,
  emptyMessage = "No discovery snapshots have been saved yet.",
}: DiscoverySnapshotsPanelProps) {
  return (
    <section className="panel p-5">
      <p className="font-display text-2xl text-ink">Discovery snapshots</p>
      <p className="mt-1 text-sm text-slate-500">Saved discovery baselines that can anchor drift comparison and migration planning.</p>

      {loading && snapshots.length === 0 && <p className="mt-5 text-sm text-slate-500">Loading discovery snapshots…</p>}
      {error && !loading && <p className="mt-5 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

      <div className="mt-5 space-y-3">
        {snapshots.map((snapshot) => (
          <article key={snapshot.id} className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600">
            <p className="font-semibold text-ink">{snapshot.source}</p>
            <p className="text-slate-500">{snapshot.platform} · {snapshot.vm_count} VMs</p>
            <p className="mt-2 text-slate-500">{new Date(snapshot.discovered_at).toLocaleString()}</p>
          </article>
        ))}
        {!loading && snapshots.length === 0 && !error && <p className="text-sm text-slate-500">{emptyMessage}</p>}
      </div>
    </section>
  );
}
