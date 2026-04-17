import { useEffect, useMemo, useState } from "react";
import { getRouteHref } from "../../app/navigation";
import { DiscoverySnapshotsPanel } from "../../components/DiscoverySnapshotsPanel";
import { MigrationHistory } from "../../components/MigrationHistory";
import { MigrationWizard } from "../../components/MigrationWizard";
import { ErrorState } from "../../components/primitives/ErrorState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type {
	DiscoveryResult,
	MigrationMeta,
	Pagination,
	SnapshotMeta,
	TenantSummary,
} from "../../types";
import {
	clearInventoryPlanningDraft,
	loadInventoryPlanningDraft,
	type InventoryPlanningDraft,
} from "../inventory/inventoryPlanningDraft";
import { summarizeInventoryRows } from "../inventory/inventoryModel";
import { useInventoryAssessment } from "../inventory/useInventoryAssessment";
import {
	countPersistedMigrationListStatuses,
	describeMigrationPhase,
	getPersistedMigrationListPresentation,
	getPersistedMigrationListStatus,
	getWorkflowStatusPresentation,
} from "./migrationStatus";

interface MigrationsPageProps {
	inventory: DiscoveryResult | null;
	migrations: MigrationMeta[];
	migrationsPagination: Pagination | null;
	migrationsPage: number;
	snapshots: SnapshotMeta[];
	snapshotsPagination: Pagination | null;
	snapshotsPage: number;
	summary: TenantSummary | null;
	latestSnapshot: SnapshotMeta | null;
	refreshToken: number;
	loading: boolean;
	migrationError?: string;
	snapshotError?: string;
	onMigrationChange: () => void | Promise<void>;
	onMigrationsPageChange: (page: number) => void;
	onSnapshotsPageChange: (page: number) => void;
}

export function MigrationsPage({
	inventory,
	migrations,
	migrationsPagination,
	migrationsPage,
	snapshots,
	snapshotsPagination,
	snapshotsPage,
	summary,
	latestSnapshot,
	refreshToken,
	loading,
	migrationError,
	snapshotError,
	onMigrationChange,
	onMigrationsPageChange,
	onSnapshotsPageChange,
}: MigrationsPageProps) {
	const [planningDraft, setPlanningDraft] =
		useState<InventoryPlanningDraft | null>(null);
	const historyError = [migrationError, snapshotError]
		.filter(Boolean)
		.join(" ");
	const assessment = useInventoryAssessment({
		inventory,
		latestSnapshot,
		refreshToken,
	});
	const queueCounts = useMemo(
		() => countPersistedMigrationListStatuses(migrations),
		[migrations],
	);

	useEffect(() => {
		setPlanningDraft(loadInventoryPlanningDraft());
	}, []);

	const planningRows = useMemo(() => {
		if (!planningDraft) {
			return [];
		}

		const selectedKeys = new Set(planningDraft.workloadKeys);
		return assessment.rows.filter((row) => selectedKeys.has(row.id));
	}, [assessment.rows, planningDraft]);

	const planningSummary = useMemo(
		() => summarizeInventoryRows(planningRows, planningRows.length),
		[planningRows],
	);
	const dependencySummary = useMemo(
		() => ({
			networks: planningRows.reduce(
				(total, row) => total + row.dependencies.networks.length,
				0,
			),
			datastores: planningRows.reduce(
				(total, row) => total + row.dependencies.datastores.length,
				0,
			),
			backups: planningRows.reduce(
				(total, row) => total + row.dependencies.backups.length,
				0,
			),
		}),
		[planningRows],
	);
	const unmatchedDraftCount = Math.max(
		0,
		(planningDraft?.workloads.length ?? 0) - planningRows.length,
	);

	function handleClearPlanningDraft() {
		clearInventoryPlanningDraft();
		setPlanningDraft(null);
	}

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Execution"
				title="Migrations"
				description="Move from selected inventory into validated migration plans, save execution-ready plan state, and keep running jobs visible in the same operational surface."
				badges={[
					{
						label: `${summary?.active_migrations ?? 0} active`,
						tone: "accent",
					},
					{
						label: `${summary?.pending_approvals ?? 0} pending approval`,
						tone: "warning",
					},
					{ label: `${migrations.length} recorded`, tone: "neutral" },
				]}
				actions={
					<>
						<a
							href={getRouteHref("/inventory")}
							className="operator-button-secondary"
						>
							Review inventory
						</a>
						<a
							href={getRouteHref("/reports")}
							className="operator-button-secondary"
						>
							Open reports
						</a>
					</>
				}
			/>

			<section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
				<SectionCard
					title="Planning intake"
					description="Inventory selections arrive here as a local dashboard draft. The draft guides planning, but it is not a persisted backend migration record until you save a migration plan."
					actions={
						planningDraft ? (
							<button
								type="button"
								onClick={handleClearPlanningDraft}
								className="operator-button-secondary"
							>
								Clear draft
							</button>
						) : undefined
					}
				>
					{planningDraft ? (
						<div className="space-y-4">
							<div className="flex flex-wrap gap-2">
								<StatusBadge tone="accent">Imported from inventory</StatusBadge>
								<StatusBadge tone="info">
									{planningDraft.sourcePlatform}
								</StatusBadge>
								<StatusBadge tone="neutral">
									{planningDraft.workloads.length} workload(s)
								</StatusBadge>
								<StatusBadge
									tone={planningSummary.highRisk > 0 ? "warning" : "success"}
								>
									{planningSummary.highRisk} high risk
								</StatusBadge>
							</div>

							<div className="grid gap-3 md:grid-cols-4">
								<StatCard
									label="Ready"
									value={planningSummary.ready}
									badge={{ label: "Ready", tone: "success" }}
									emphasis="large"
								/>
								<StatCard
									label="Needs review"
									value={planningSummary.needsReview}
									badge={{ label: "Review", tone: "warning" }}
									emphasis="large"
								/>
								<StatCard
									label="Blocked"
									value={planningSummary.blocked}
									badge={{ label: "Blocked", tone: "danger" }}
									emphasis="large"
								/>
								<StatCard
									label="High risk"
									value={planningSummary.highRisk}
									badge={{ label: "Risk", tone: "warning" }}
									emphasis="large"
								/>
							</div>

							<div className="grid gap-3 md:grid-cols-3">
								<StatCard
									label="Network relationships"
									value={dependencySummary.networks}
								/>
								<StatCard
									label="Storage relationships"
									value={dependencySummary.datastores}
								/>
								<StatCard
									label="Backup relationships"
									value={dependencySummary.backups}
								/>
							</div>

							<InlineNotice
								message="The workload list came from the inventory route. Viaduct still requires source and target endpoint details plus a saved plan state before execution can start."
								tone="info"
							/>

							{assessment.loading ? (
								<InlineNotice
									message="Refreshing dependency and lifecycle signals for the imported workloads."
									tone="neutral"
								/>
							) : null}

							{assessment.error ? (
								<InlineNotice
									message={`Imported workloads are visible, but some readiness context is partial: ${assessment.error}`}
									tone="warning"
								/>
							) : null}

							{unmatchedDraftCount > 0 ? (
								<InlineNotice
									message={`${unmatchedDraftCount} imported workload(s) were not matched back to the current inventory assessment. The draft is still available, but risk and dependency context may be incomplete until the latest inventory lines up again.`}
									tone="warning"
								/>
							) : null}
						</div>
					) : (
						<InlineNotice
							title="No inventory draft is active"
							message="Start from the inventory route when you want a selected workload set to carry directly into migration planning."
							tone="neutral"
						/>
					)}
				</SectionCard>

				<SectionCard
					title="Execution posture"
					description="Live migration activity from the persisted store. Saved plans stay in plan phase until you explicitly execute them."
				>
					<div className="grid gap-3 md:grid-cols-2">
						<StatCard label="Saved plans" value={queueCounts.planned} />
						<StatCard label="Running" value={queueCounts.running} />
						<StatCard
							label="Pending approvals"
							value={summary?.pending_approvals ?? 0}
						/>
						<StatCard label="Failed" value={queueCounts.failed} />
					</div>

					<div className="mt-5 space-y-3">
						{migrations.slice(0, 3).map((migration) => {
							const status = getPersistedMigrationListPresentation(
								getPersistedMigrationListStatus(migration.phase),
							);

							return (
								<article key={migration.id} className="list-card text-sm text-slate-600">
									<div className="flex flex-wrap items-center justify-between gap-3">
										<div>
											<p className="font-semibold text-ink">
												{migration.spec_name}
											</p>
											<p className="text-slate-500">{migration.id}</p>
										</div>
										<div className="flex flex-wrap gap-2">
											<StatusBadge tone={status.tone}>
												{status.label}
											</StatusBadge>
											<StatusBadge tone="neutral">
												{describeMigrationPhase(migration.phase)}
											</StatusBadge>
										</div>
									</div>
									<p className="mt-3 text-slate-500">
										Started {new Date(migration.started_at).toLocaleString()} and
										updated {new Date(migration.updated_at).toLocaleString()}.
									</p>
								</article>
							);
						})}

						{migrations.length === 0 && !loading && !migrationError ? (
							<InlineNotice
								message="No persisted migration runs have been recorded yet."
								tone="neutral"
							/>
						) : null}
					</div>
				</SectionCard>
			</section>

			<SectionCard
				title="Operational states"
				description="Draft, ready, warning, and blocked are detailed planning states inside the workspace. Persisted migration history only exposes plan and phase metadata from the list API."
			>
				<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
					{(
						[
							"draft",
							"ready",
							"warning",
							"blocked",
							"running",
							"failed",
							"completed",
						] as const
					).map((statusKey) => {
						const status = getWorkflowStatusPresentation(statusKey);

						return (
							<StatCard
								key={statusKey}
								label={status.label}
								value={status.description}
								badge={{ label: status.label, tone: status.tone }}
							/>
						);
					})}
				</div>
			</SectionCard>

			<MigrationWizard
				planningDraft={planningDraft}
				onPlanningDraftCleared={handleClearPlanningDraft}
				onMigrationChange={onMigrationChange}
			/>

			{loading &&
			migrations.length === 0 &&
			snapshots.length === 0 &&
			!planningDraft ? (
				<LoadingState
					title="Loading migration operations"
					message="Retrieving migration history and saved discovery baselines so operators can validate the current runbook context."
				/>
			) : null}

			{historyError &&
			migrations.length === 0 &&
			snapshots.length === 0 &&
			!loading ? (
				<ErrorState
					title="Migration history unavailable"
					message={historyError}
				/>
			) : null}

			<section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
				<MigrationHistory
					migrations={migrations.slice(0, 6)}
					loading={loading}
					error={migrationError}
				/>
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
		</div>
	);
}
