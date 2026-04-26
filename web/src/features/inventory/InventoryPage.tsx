import { useRef } from "react";
import { getRouteHref } from "../../app/navigation";
import { InventoryTable } from "../../components/InventoryTable";
import { PlatformSummary } from "../../components/PlatformSummary";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type {
	DiscoveryResult,
	Pagination,
	SnapshotMeta,
	TenantSummary,
} from "../../types";
import { saveInventoryPlanningDraft } from "./inventoryPlanningDraft";
import { formatTimestamp, type InventoryAssessmentRow } from "./inventoryModel";
import { revealWorkloadDetailPanel } from "./revealWorkloadDetailPanel";
import { useInventoryAssessment } from "./useInventoryAssessment";
import { getPlatformScopeLabel } from "./workloadIdentity";
import { useInventoryWorkspace } from "./useInventoryWorkspace";
import { WorkloadDetailPanel } from "./WorkloadDetailPanel";

interface InventoryPageProps {
	inventory: DiscoveryResult | null;
	inventoryPagination: Pagination | null;
	inventoryPage: number;
	summary: TenantSummary | null;
	latestSnapshot: SnapshotMeta | null;
	refreshToken: number;
	loading: boolean;
	error: string | null;
	onInventoryPageChange: (page: number) => void;
}

export function InventoryPage({
	inventory,
	inventoryPagination,
	inventoryPage,
	summary,
	latestSnapshot,
	refreshToken,
	loading,
	error,
	onInventoryPageChange,
}: InventoryPageProps) {
	const assessment = useInventoryAssessment({
		inventory,
		latestSnapshot,
		refreshToken,
	});
	const workspace = useInventoryWorkspace(assessment.rows);
	const detailPanelRef = useRef<HTMLDivElement | null>(null);
	const planningBlockedReason = getPlanningBlockedReason(
		workspace.selectedRows,
	);
	const planningNote =
		workspace.selectedRows.length > 0
			? "This action creates a local planning draft and opens migration planning with the selected workloads already in scope. Viaduct still requires source and target endpoint details before preflight, plan save, and execution because the backend contract is spec-driven."
			: null;
	const selectionScope = getSelectionScopeLabel(workspace.selectedRows);
	const assetCoverage = {
		hosts: inventory?.hosts?.length ?? 0,
		clusters: inventory?.clusters?.length ?? 0,
		networks: inventory?.networks?.length ?? 0,
		datastores: inventory?.datastores?.length ?? 0,
		resourcePools: inventory?.resource_pools?.length ?? 0,
	};

	function handlePreparePlan(rowsToPlan: InventoryAssessmentRow[]) {
		if (rowsToPlan.length === 0) {
			return;
		}

		const [firstRow] = rowsToPlan;
		saveInventoryPlanningDraft({
			version: 1,
			createdAt: new Date().toISOString(),
			sourcePlatform: firstRow.vm.platform,
			workloadKeys: rowsToPlan.map((row) => row.id),
			workloads: rowsToPlan.map((row) => row.vm),
		});
		window.location.hash = getRouteHref("/migrations");
	}

	function handleFocusWorkload(id: string) {
		workspace.setActiveWorkloadId(id);
		revealWorkloadDetailPanel(detailPanelRef.current);
	}

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Inventory"
				title="Fleet inventory and assessment"
				description="Review workloads with search, dependency context, and migration planning inputs."
				badges={[
					{
						label: `${summary?.workload_count ?? inventory?.vms.length ?? 0} workloads`,
						tone: "info",
					},
					{
						label: `${workspace.summary.highRisk} high risk`,
						tone: workspace.summary.highRisk > 0 ? "warning" : "success",
					},
					{
						label: `${workspace.selectedIds.length} selected`,
						tone: workspace.selectedIds.length > 0 ? "accent" : "neutral",
					},
					{
						label: assessment.loading
							? "Assessment refreshing"
							: assessment.error
								? "Assessment partial"
								: "Assessment current",
						tone: assessment.loading
							? "info"
							: assessment.error
								? "warning"
								: "success",
					},
				]}
				actions={
					<a
						href={getRouteHref("/graph")}
						className="operator-button-secondary"
					>
						Open dependency graph
					</a>
				}
			/>

			{loading && !inventory ? (
				<LoadingState
					title="Loading inventory"
					message="Retrieving normalized workload inventory, dependency context, and assessment signals from the Viaduct API."
				/>
			) : null}

			{!loading && error && !inventory ? (
				<ErrorState title="Inventory unavailable" message={error} />
			) : null}

			{!loading && !error && !inventory ? (
				<EmptyState
					title="No inventory returned"
					message="Run discovery or connect a source platform so Viaduct can populate the workload inventory surface."
				/>
			) : null}

			{inventory ? (
				<>
					<section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
						<StatCard
							label="Ready"
							value={workspace.summary.ready}
							badge={{ label: "Ready", tone: "success" }}
							emphasis="large"
						/>
						<StatCard
							label="Needs review"
							value={workspace.summary.needsReview}
							badge={{ label: "Review", tone: "warning" }}
							emphasis="large"
						/>
						<StatCard
							label="Blocked"
							value={workspace.summary.blocked}
							badge={{ label: "Blocked", tone: "danger" }}
							emphasis="large"
						/>
						<StatCard
							label="High risk"
							value={workspace.summary.highRisk}
							badge={{ label: "Risk", tone: "warning" }}
							emphasis="large"
						/>
						<StatCard
							label="Selected"
							value={workspace.summary.selected}
							badge={{ label: "Selected", tone: "accent" }}
							emphasis="large"
						/>
					</section>

					<section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
						<SectionCard
							title="Discovery context"
							description="Current inventory source and the latest baseline Viaduct is assessing."
						>
							<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
								<StatCard
									label="Source"
									value={
										inventory.source?.trim()
											? inventory.source
											: "Mixed latest snapshots"
									}
								/>
								<StatCard
									label="Platform scope"
									value={
										inventory.platform?.trim()
											? inventory.platform
											: "Mixed platform inventory"
									}
								/>
								<StatCard
									label="Latest baseline"
									value={
										latestSnapshot
											? formatTimestamp(latestSnapshot.discovered_at)
											: "No saved snapshot"
									}
								/>
								<StatCard
									label="Inventory errors"
									value={String(inventory.errors?.length ?? 0)}
								/>
							</div>
							{inventory.errors && inventory.errors.length > 0 ? (
								<div className="mt-4 space-y-2">
									{inventory.errors.map((item) => (
										<InlineNotice key={item} message={item} tone="warning" />
									))}
								</div>
							) : null}
						</SectionCard>

						<SectionCard
							title="Asset coverage"
							description="Normalized infrastructure context shipped with the current inventory contract."
						>
							<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
								<StatCard label="Hosts" value={assetCoverage.hosts} />
								<StatCard label="Clusters" value={assetCoverage.clusters} />
								<StatCard label="Networks" value={assetCoverage.networks} />
								<StatCard label="Datastores" value={assetCoverage.datastores} />
								<StatCard
									label="Resource pools"
									value={assetCoverage.resourcePools}
								/>
								<StatCard
									label="Snapshot quota free"
									value={summary?.snapshot_quota_free ?? 0}
								/>
							</div>
						</SectionCard>
					</section>

					<PlatformSummary inventory={inventory} />

					<SectionCard
						title="Assessment notes"
						description="Signals that shape readiness and risk in the current view."
					>
						<div className="flex flex-wrap gap-2">
							<StatusBadge
								tone={assessment.sources.graph ? "success" : "warning"}
							>
								Graph {assessment.sources.graph ? "ready" : "partial"}
							</StatusBadge>
							<StatusBadge
								tone={assessment.sources.policies ? "success" : "warning"}
							>
								Policies {assessment.sources.policies ? "ready" : "partial"}
							</StatusBadge>
							<StatusBadge
								tone={assessment.sources.remediation ? "success" : "warning"}
							>
								Remediation{" "}
								{assessment.sources.remediation ? "ready" : "partial"}
							</StatusBadge>
							{assessment.topSignals.length > 0 ? (
								assessment.topSignals.map((signal) => (
									<StatusBadge key={signal.label} tone="neutral">
										{signal.label}: {signal.count}
									</StatusBadge>
								))
							) : (
								<StatusBadge tone="success">
									No derived risk drivers yet
								</StatusBadge>
							)}
						</div>
						{assessment.loading ? (
							<div className="mt-4">
								<InlineNotice
									message="Refreshing dependency and lifecycle assessment signals for the current workload baseline."
									tone="neutral"
								/>
							</div>
						) : null}
						<div className="mt-4">
							<InlineNotice
								message="Readiness in this view is derived from current inventory, policy violations, remediation guidance, and graph relationships. Viaduct does not yet expose a VM-scoped preflight or activity endpoint for a cleaner per-workload readiness contract."
								tone="info"
							/>
						</div>
						{assessment.error ? (
							<div className="mt-4">
								<InlineNotice
									message={`Inventory data loaded, but some assessment signals are partial: ${assessment.error}`}
									tone="warning"
								/>
							</div>
						) : null}
					</SectionCard>

					<section className="grid gap-5 min-[1800px]:grid-cols-[minmax(0,1.2fr)_minmax(360px,0.8fr)]">
						<InventoryTable
							rows={workspace.filteredRows}
							totalCount={inventoryPagination?.total ?? assessment.rows.length}
							filteredCount={workspace.filteredRows.length}
							selectedCount={workspace.selectedIds.length}
							hasActiveFilters={workspace.hasActiveFilters}
							loading={loading && assessment.rows.length === 0}
							refreshing={assessment.loading && assessment.rows.length > 0}
							error={error}
							availablePlatforms={workspace.availablePlatforms}
							filters={workspace.filters}
							sortKey={workspace.sortKey}
							sortDirection={workspace.sortDirection}
							activeWorkloadId={workspace.activeRow?.id ?? null}
							selectedIds={workspace.selectedIds}
							onFiltersChange={workspace.updateFilters}
							onSortChange={workspace.changeSort}
							onToggleSelection={workspace.toggleSelection}
							onToggleSelectAllVisible={workspace.toggleSelectAllVisible}
							onClearSelection={workspace.clearSelection}
							onResetFilters={workspace.resetFilters}
							onFocusWorkload={handleFocusWorkload}
							pagination={inventoryPagination}
							currentPage={inventoryPage}
							onPageChange={onInventoryPageChange}
							actions={
								<>
									<StatusBadge
										tone={
											workspace.visibleSelectedCount > 0 ? "accent" : "neutral"
										}
									>
										{workspace.visibleSelectedCount} visible selected
									</StatusBadge>
									{selectionScope ? (
										<StatusBadge tone="neutral">{selectionScope}</StatusBadge>
									) : null}
									<button
										type="button"
										onClick={() => handlePreparePlan(workspace.selectedRows)}
										disabled={
											workspace.selectedRows.length === 0 ||
											Boolean(planningBlockedReason)
										}
										className="operator-button"
									>
										Open migration plan
									</button>
								</>
							}
						/>

						<div
							ref={detailPanelRef}
							tabIndex={-1}
							className="scroll-mt-6 outline-none"
						>
							<WorkloadDetailPanel
								row={workspace.activeRow}
								latestSnapshot={latestSnapshot}
								assessmentErrors={assessment.errors}
								onPrimaryAction={(row) => handlePreparePlan([row])}
							/>
						</div>
					</section>

					{(planningBlockedReason || planningNote) &&
					workspace.selectedRows.length > 0 ? (
						<InlineNotice
							message={planningBlockedReason ?? planningNote}
							tone={planningBlockedReason ? "warning" : "info"}
						/>
					) : null}
				</>
			) : null}
		</div>
	);
}

function getPlanningBlockedReason(
	rows: InventoryAssessmentRow[],
): string | null {
	if (rows.length === 0) {
		return null;
	}

	const platforms = new Set(rows.map((row) => row.vm.platform));
	if (platforms.size > 1) {
		return "Current migration specs are single-source and single-platform. Narrow the selection to one source platform before preparing a planning draft.";
	}

	return null;
}

function getSelectionScopeLabel(rows: InventoryAssessmentRow[]): string | null {
	if (rows.length === 0) {
		return null;
	}

	const platforms = Array.from(new Set(rows.map((row) => row.vm.platform)));
	if (platforms.length === 1) {
		return `${rows.length} workload(s) from ${getPlatformScopeLabel(platforms[0])}`;
	}

	return `${rows.length} workload(s) across ${platforms.length} platforms`;
}
