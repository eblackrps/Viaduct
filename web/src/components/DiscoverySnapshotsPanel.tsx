import type { Pagination, SnapshotMeta } from "../types";

interface DiscoverySnapshotsPanelProps {
	snapshots: SnapshotMeta[];
	loading?: boolean;
	error?: string | null;
	emptyMessage?: string;
	pagination?: Pagination | null;
	currentPage?: number;
	onPageChange?: (page: number) => void;
}

export function DiscoverySnapshotsPanel({
	snapshots,
	loading = false,
	error,
	emptyMessage = "No discovery snapshots have been saved yet.",
	pagination,
	currentPage = 1,
	onPageChange,
}: DiscoverySnapshotsPanelProps) {
	return (
		<section className="panel p-5" aria-live="polite">
			<h2 className="font-display text-2xl text-ink">Discovery snapshots</h2>
			<p className="mt-1 text-sm text-slate-500">
				Saved discovery baselines that can anchor drift comparison and migration
				planning.
			</p>

			{loading && snapshots.length === 0 && (
				<p className="mt-5 text-sm text-slate-500">
					Loading discovery snapshots…
				</p>
			)}
			{error && !loading && (
				<p className="mt-5 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
					{error}
				</p>
			)}

			<div className="mt-5 space-y-3">
				{snapshots.map((snapshot) => (
					<article
						key={snapshot.id}
						className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600"
					>
						<p className="font-semibold text-ink">{snapshot.source}</p>
						<p className="text-slate-500">
							{snapshot.platform} · {snapshot.vm_count} VMs
						</p>
						<p className="mt-2 text-slate-500">
							{new Date(snapshot.discovered_at).toLocaleString()}
						</p>
					</article>
				))}
				{!loading && snapshots.length === 0 && !error && (
					<p className="text-sm text-slate-500">{emptyMessage}</p>
				)}
			</div>

			{pagination && pagination.total_pages > 1 && onPageChange && (
				<div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
					<p>
						Page {currentPage} of {pagination.total_pages} •{" "}
						{pagination.total.toLocaleString()} snapshot(s)
					</p>
					<div className="flex flex-wrap items-center gap-2">
						<button
							type="button"
							onClick={() => onPageChange(Math.max(1, currentPage - 1))}
							disabled={currentPage <= 1}
							className="operator-button-secondary px-3 py-2 disabled:cursor-not-allowed disabled:opacity-50"
						>
							Previous
						</button>
						<button
							type="button"
							onClick={() =>
								onPageChange(Math.min(pagination.total_pages, currentPage + 1))
							}
							disabled={currentPage >= pagination.total_pages}
							className="operator-button-secondary px-3 py-2 disabled:cursor-not-allowed disabled:opacity-50"
						>
							Next
						</button>
					</div>
				</div>
			)}
		</section>
	);
}
