import type { Pagination, SnapshotMeta } from "../types";
import { PaginationControls } from "./primitives/PaginationControls";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatusBadge } from "./primitives/StatusBadge";

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
		<SectionCard
			title="Discovery snapshots"
			description="Saved discovery baselines that can anchor drift comparison and migration planning."
		>
			{loading && snapshots.length === 0 ? (
				<InlineNotice message="Loading discovery snapshots…" tone="neutral" />
			) : null}
			{error && !loading ? (
				<InlineNotice message={error} tone="danger" className="mt-4" />
			) : null}

			<div className={loading || error ? "mt-4 space-y-3" : "space-y-3"}>
				{snapshots.map((snapshot) => (
					<article key={snapshot.id} className="list-card text-sm text-slate-600">
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">{snapshot.source}</p>
								<p className="mt-1 text-xs text-slate-500">{snapshot.id}</p>
							</div>
							<div className="flex flex-wrap gap-2">
								<StatusBadge tone="info">{snapshot.platform}</StatusBadge>
								<StatusBadge tone="neutral">
									{snapshot.vm_count} VM{snapshot.vm_count === 1 ? "" : "s"}
								</StatusBadge>
							</div>
						</div>
						<div className="mt-4 metric-surface">
							<p className="operator-kicker">Discovered</p>
							<p className="mt-2 text-sm font-semibold text-ink">
								{new Date(snapshot.discovered_at).toLocaleString()}
							</p>
						</div>
					</article>
				))}

				{!loading && snapshots.length === 0 && !error ? (
					<InlineNotice message={emptyMessage} tone="neutral" />
				) : null}
			</div>

			{pagination && pagination.total_pages > 1 && onPageChange ? (
				<PaginationControls
					currentPage={currentPage}
					totalPages={pagination.total_pages}
					totalItems={pagination.total}
					itemLabel="snapshot(s)"
					onPageChange={onPageChange}
				/>
			) : null}
		</SectionCard>
	);
}
