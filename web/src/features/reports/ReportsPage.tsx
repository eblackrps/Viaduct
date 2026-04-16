import { useState } from "react";
import {
	dashboardAuthMode,
	describeError,
	downloadReport,
	type ErrorDisplay,
	type ReportFormat,
	type ReportName,
} from "../../api";
import { DiscoverySnapshotsPanel } from "../../components/DiscoverySnapshotsPanel";
import { MigrationHistory } from "../../components/MigrationHistory";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { MigrationMeta, Pagination, SnapshotMeta } from "../../types";

interface ReportsPageProps {
	migrations: MigrationMeta[];
	migrationsPagination: Pagination | null;
	migrationsPage: number;
	snapshots: SnapshotMeta[];
	snapshotsPagination: Pagination | null;
	snapshotsPage: number;
	loading: boolean;
	migrationError?: string;
	snapshotError?: string;
	onMigrationsPageChange: (page: number) => void;
	onSnapshotsPageChange: (page: number) => void;
}

export function ReportsPage({
	migrations,
	migrationsPagination,
	migrationsPage,
	snapshots,
	snapshotsPagination,
	snapshotsPage,
	loading,
	migrationError,
	snapshotError,
	onMigrationsPageChange,
	onSnapshotsPageChange,
}: ReportsPageProps) {
	const authMode = dashboardAuthMode();
	const showLoading =
		loading && migrations.length === 0 && snapshots.length === 0;
	const hasErrors = [migrationError, snapshotError].filter(Boolean).join(" ");
	const showEmpty =
		!loading && !hasErrors && migrations.length === 0 && snapshots.length === 0;
	const [activeExport, setActiveExport] = useState<string | null>(null);
	const [exportError, setExportError] = useState<ErrorDisplay | null>(null);

	async function handleExport(name: ReportName, format: ReportFormat) {
		const exportKey = `${name}:${format}`;
		try {
			setActiveExport(exportKey);
			setExportError(null);
			const result = await downloadReport(name, format);
			const href = URL.createObjectURL(result.blob);
			const anchor = document.createElement("a");
			anchor.href = href;
			anchor.download = result.filename;
			anchor.click();
			window.setTimeout(() => URL.revokeObjectURL(href), 0);
		} catch (err) {
			setExportError(
				describeError(err, {
					fallback: "Unable to export report.",
				}),
			);
		} finally {
			setActiveExport(null);
		}
	}

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Reports"
				title="Reports and history"
				description="Keep migration records, saved discovery baselines, and export-ready operator report endpoints accessible from one administrative surface."
				badges={[
					{ label: `${migrations.length} migrations`, tone: "neutral" },
					{ label: `${snapshots.length} snapshots`, tone: "info" },
				]}
			/>

			{showLoading && (
				<LoadingState
					title="Loading reports surface"
					message="Retrieving migration history and saved discovery baselines for the reporting and audit view."
				/>
			)}

			{showEmpty && (
				<EmptyState
					title="No reports available yet"
					message="Once Viaduct records migrations or discovery snapshots, they will appear here alongside export links."
				/>
			)}

			<SectionCard
				title="Operator API exports"
				description="Direct links into the current backend report surface published by the operator API."
			>
				<div className="flex flex-wrap items-center gap-3">
					<StatusBadge
						tone={
							authMode === "service-account"
								? "info"
								: authMode === "tenant"
									? "success"
									: "warning"
						}
					>
						{authMode === "service-account"
							? "Service account auth"
							: authMode === "tenant"
								? "Tenant key auth"
								: "No dashboard auth configured"}
					</StatusBadge>
					<ExportButton
						label="Summary JSON"
						loading={activeExport === "summary:json"}
						onClick={() => void handleExport("summary", "json")}
					/>
					<ExportButton
						label="Summary CSV"
						loading={activeExport === "summary:csv"}
						onClick={() => void handleExport("summary", "csv")}
					/>
					<ExportButton
						label="Migrations JSON"
						loading={activeExport === "migrations:json"}
						onClick={() => void handleExport("migrations", "json")}
					/>
					<ExportButton
						label="Audit JSON"
						loading={activeExport === "audit:json"}
						onClick={() => void handleExport("audit", "json")}
					/>
				</div>
				<p className="mt-4 text-sm text-slate-500">
					Downloads are performed through the dashboard client so configured
					tenant or service-account credentials are applied consistently.
				</p>
				{exportError && (
					<div className="mt-4 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
						<p>{exportError.message}</p>
						{exportError.technicalDetails.length > 0 && (
							<div className="mt-3 rounded-2xl bg-white/70 px-4 py-3 text-xs text-rose-800">
								{exportError.technicalDetails.map((detail, index) => (
									<p
										key={`${detail}-${index}`}
										className={index === 0 ? undefined : "mt-1"}
									>
										{detail}
									</p>
								))}
							</div>
						)}
					</div>
				)}
			</SectionCard>

			{hasErrors &&
				migrations.length === 0 &&
				snapshots.length === 0 &&
				!loading && (
					<ErrorState title="Reports surface unavailable" message={hasErrors} />
				)}

			{(migrations.length > 0 || snapshots.length > 0) && (
				<section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
					<MigrationHistory
						migrations={migrations}
						loading={loading}
						error={migrationError}
						pagination={migrationsPagination}
						currentPage={migrationsPage}
						onPageChange={onMigrationsPageChange}
					/>
					<DiscoverySnapshotsPanel
						snapshots={snapshots}
						loading={loading}
						error={snapshotError}
						pagination={snapshotsPagination}
						currentPage={snapshotsPage}
						onPageChange={onSnapshotsPageChange}
					/>
				</section>
			)}
		</div>
	);
}

function ExportButton({
	label,
	loading,
	onClick,
}: {
	label: string;
	loading: boolean;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={onClick}
			disabled={loading}
			className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
		>
			{loading ? "Exporting…" : label}
		</button>
	);
}
