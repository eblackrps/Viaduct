import type { MigrationMeta, Pagination } from "../types";
import { PaginationControls } from "./primitives/PaginationControls";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
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
	pagination?: Pagination | null;
	currentPage?: number;
	onPageChange?: (page: number) => void;
}

export function MigrationHistory({
	migrations,
	loading = false,
	error,
	emptyMessage = "No migrations have been recorded yet.",
	pagination,
	currentPage = 1,
	onPageChange,
}: MigrationHistoryProps) {
	return (
		<SectionCard
			title="Migration history"
			description="Track completed, failed, and in-flight migrations from the shared state store."
		>
			{loading && migrations.length === 0 ? (
				<InlineNotice message="Loading migration history…" tone="neutral" />
			) : null}
			{error && !loading ? (
				<InlineNotice message={error} tone="danger" className="mt-4" />
			) : null}

			<div className={loading || error ? "mt-4 space-y-3" : "space-y-3"}>
				{migrations.map((migration) => {
					const status = getPersistedMigrationListPresentation(
						getPersistedMigrationListStatus(migration.phase),
					);

					return (
						<article
							key={migration.id}
							className="list-card text-sm text-slate-600"
						>
							<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
								<div>
									<p className="font-semibold text-ink">
										{migration.spec_name}
									</p>
									<p className="mt-1 break-all text-xs text-slate-500">
										{migration.id}
									</p>
								</div>
								<div className="flex flex-wrap gap-2">
									<StatusBadge tone={status.tone}>{status.label}</StatusBadge>
									<StatusBadge tone={phaseTone(migration.phase)}>
										{describeMigrationPhase(migration.phase)}
									</StatusBadge>
								</div>
							</div>
							<div className="mt-4 grid gap-3 md:grid-cols-2">
								<div className="metric-surface">
									<p className="operator-kicker">Started</p>
									<p className="mt-2 text-sm font-semibold text-ink">
										{new Date(migration.started_at).toLocaleString()}
									</p>
								</div>
								<div className="metric-surface">
									<p className="operator-kicker">
										{migration.completed_at ? "Completed" : "Updated"}
									</p>
									<p className="mt-2 text-sm font-semibold text-ink">
										{new Date(
											migration.completed_at ?? migration.updated_at,
										).toLocaleString()}
									</p>
								</div>
							</div>
						</article>
					);
				})}

				{!loading && migrations.length === 0 && !error ? (
					<InlineNotice message={emptyMessage} tone="neutral" />
				) : null}
			</div>

			{pagination && pagination.total_pages > 1 && onPageChange ? (
				<PaginationControls
					currentPage={currentPage}
					totalPages={pagination.total_pages}
					totalItems={pagination.total}
					itemLabel="migration(s)"
					onPageChange={onPageChange}
				/>
			) : null}
		</SectionCard>
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
