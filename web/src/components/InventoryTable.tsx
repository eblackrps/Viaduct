import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { ArrowDownUp, Search } from "lucide-react";
import type { Pagination, Platform } from "../types";
import {
	formatRelativeTime,
	type InventoryAssessmentRow,
	type InventoryFilterState,
	type InventoryReadinessState,
	type InventoryRiskState,
	type InventorySortKey,
} from "../features/inventory/inventoryModel";
import { InlineNotice } from "./primitives/InlineNotice";
import { PaginationControls } from "./primitives/PaginationControls";
import { SectionCard } from "./primitives/SectionCard";
import { StatusBadge, type StatusTone } from "./primitives/StatusBadge";

interface InventoryTableProps {
	rows: InventoryAssessmentRow[];
	totalCount: number;
	filteredCount: number;
	selectedCount: number;
	hasActiveFilters?: boolean;
	loading?: boolean;
	refreshing?: boolean;
	error?: string | null;
	availablePlatforms: Platform[];
	filters: InventoryFilterState;
	sortKey: InventorySortKey;
	sortDirection: "asc" | "desc";
	activeWorkloadId?: string | null;
	selectedIds: string[];
	pagination?: Pagination | null;
	currentPage?: number;
	actions?: ReactNode;
	onFiltersChange: (updates: Partial<InventoryFilterState>) => void;
	onSortChange: (key: InventorySortKey) => void;
	onToggleSelection: (id: string) => void;
	onToggleSelectAllVisible: () => void;
	onClearSelection: () => void;
	onResetFilters: () => void;
	onFocusWorkload: (id: string) => void;
	onPageChange?: (page: number) => void;
}

export function InventoryTable({
	rows,
	totalCount,
	filteredCount,
	selectedCount,
	hasActiveFilters = false,
	loading = false,
	refreshing = false,
	error,
	availablePlatforms,
	filters,
	sortKey,
	sortDirection,
	activeWorkloadId,
	selectedIds,
	pagination,
	currentPage = 1,
	actions,
	onFiltersChange,
	onSortChange,
	onToggleSelection,
	onToggleSelectAllVisible,
	onClearSelection,
	onResetFilters,
	onFocusWorkload,
	onPageChange,
}: InventoryTableProps) {
	const masterCheckboxRef = useRef<HTMLInputElement | null>(null);
	const selectedIdSet = new Set(selectedIds);
	const allVisibleSelected =
		rows.length > 0 && rows.every((row) => selectedIdSet.has(row.id));
	const someVisibleSelected = rows.some((row) => selectedIdSet.has(row.id));

	useEffect(() => {
		if (masterCheckboxRef.current) {
			masterCheckboxRef.current.indeterminate =
				!allVisibleSelected && someVisibleSelected;
		}
	}, [allVisibleSelected, someVisibleSelected]);

	return (
		<SectionCard
			title="Workload assessment table"
			description={`${filteredCount.toLocaleString()} of ${totalCount.toLocaleString()} workload(s) shown. ${selectedCount.toLocaleString()} selected for planning handoff.`}
			actions={
				<div className="flex flex-wrap items-center gap-2">
					<div className="operator-toggle">
						<button
							type="button"
							onClick={() => onFiltersChange({ scope: "all" })}
							aria-pressed={filters.scope === "all"}
							className={`operator-toggle-button ${filters.scope === "all" ? "operator-toggle-button-active" : ""}`}
						>
							All workloads
						</button>
						<button
							type="button"
							onClick={() => onFiltersChange({ scope: "selected" })}
							aria-pressed={filters.scope === "selected"}
							className={`operator-toggle-button ${filters.scope === "selected" ? "operator-toggle-button-active" : ""}`}
							disabled={selectedCount === 0}
						>
							Selected only
						</button>
					</div>
					{hasActiveFilters ? (
						<button
							type="button"
							onClick={onResetFilters}
							className="operator-button-secondary px-3.5 py-2"
						>
							Clear filters
						</button>
					) : null}
					{actions}
				</div>
			}
		>
			<div className="space-y-5" aria-live="polite">
				<div className="grid gap-3 xl:grid-cols-[minmax(0,1.2fr)_repeat(4,minmax(160px,1fr))]">
					<label className="metric-surface">
						<span className="operator-kicker">Search</span>
						<span className="mt-2 flex items-center gap-3 rounded-[18px] border border-slate-200/80 bg-white px-4 py-3 text-sm text-slate-500 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]">
							<Search className="h-4 w-4 text-slate-400" />
							<input
								className="w-full border-none bg-transparent text-ink outline-none"
								placeholder="Search workloads, assets, tags, or policy signals"
								value={filters.search}
								onChange={(event) =>
									onFiltersChange({ search: event.target.value })
								}
							/>
						</span>
					</label>

					<FilterSelect
						label="Platform"
						value={filters.platform}
						onChange={(value) =>
							onFiltersChange({
								platform: value as InventoryFilterState["platform"],
							})
						}
						options={[
							{ label: "All platforms", value: "all" },
							...availablePlatforms.map((platform) => ({
								label: platform,
								value: platform,
							})),
						]}
					/>

					<FilterSelect
						label="Power state"
						value={filters.power}
						onChange={(value) =>
							onFiltersChange({ power: value as InventoryFilterState["power"] })
						}
						options={[
							{ label: "All power states", value: "all" },
							{ label: "Running", value: "on" },
							{ label: "Powered off", value: "off" },
							{ label: "Suspended", value: "suspended" },
							{ label: "Unknown", value: "unknown" },
						]}
					/>

					<FilterSelect
						label="Readiness"
						value={filters.readiness}
						onChange={(value) =>
							onFiltersChange({
								readiness: value as InventoryReadinessState | "all",
							})
						}
						options={[
							{ label: "All readiness states", value: "all" },
							{ label: "Ready", value: "ready" },
							{ label: "Needs review", value: "needs-review" },
							{ label: "Blocked", value: "blocked" },
						]}
					/>

					<FilterSelect
						label="Risk"
						value={filters.risk}
						onChange={(value) =>
							onFiltersChange({ risk: value as InventoryRiskState | "all" })
						}
						options={[
							{ label: "All risk levels", value: "all" },
							{ label: "High risk", value: "high" },
							{ label: "Medium risk", value: "medium" },
							{ label: "Low risk", value: "low" },
						]}
					/>
				</div>

				<div className="grid gap-3 lg:hidden sm:grid-cols-[minmax(0,1fr)_auto]">
					<FilterSelect
						label="Sort by"
						value={sortKey}
						onChange={(value) => {
							if (value !== sortKey) {
								onSortChange(value as InventorySortKey);
							}
						}}
						options={[
							{ label: "Risk", value: "risk" },
							{ label: "Readiness", value: "readiness" },
							{ label: "Recency", value: "recency" },
							{ label: "Dependencies", value: "dependencies" },
							{ label: "CPU", value: "cpu" },
							{ label: "Platform", value: "platform" },
							{ label: "Workload name", value: "name" },
						]}
					/>
					<div className="metric-surface sm:self-end">
						<span className="operator-kicker">Direction</span>
						<button
							type="button"
							onClick={() => onSortChange(sortKey)}
							aria-label={`Sort direction is ${sortDirection === "asc" ? "ascending" : "descending"}. Activate to reverse it.`}
							aria-pressed={sortDirection === "desc"}
							className="operator-button-secondary mt-2 w-full justify-center px-3.5 py-3 sm:min-w-[138px]"
						>
							{sortDirection === "asc" ? "Ascending" : "Descending"}
						</button>
					</div>
				</div>

				{selectedCount > 0 ? (
					<div className="panel-muted flex flex-wrap items-center gap-2 px-4 py-3 text-sm text-slate-600">
						<StatusBadge tone="accent">{selectedCount} selected</StatusBadge>
						<button
							type="button"
							onClick={onToggleSelectAllVisible}
							className="operator-button-secondary px-3.5 py-2"
						>
							{allVisibleSelected ? "Unselect visible" : "Select visible"}
						</button>
						<button
							type="button"
							onClick={onClearSelection}
							className="operator-button-secondary px-3.5 py-2"
						>
							Clear selection
						</button>
					</div>
				) : null}

				{error ? <InlineNotice message={error} tone="warning" /> : null}
				{refreshing && rows.length > 0 ? (
					<InlineNotice
						message="Refreshing dependency and lifecycle assessment signals for the visible workloads."
						tone="neutral"
					/>
				) : null}

				{loading && rows.length === 0 ? (
					<InlineNotice
						message="Loading workload inventory and assessment signals…"
						tone="neutral"
					/>
				) : rows.length === 0 ? (
					<InlineNotice
						message={
							filters.scope === "selected"
								? "No selected workloads match the current search or filters."
								: "No workloads match the current search or filters."
						}
						tone="neutral"
					/>
				) : (
					<>
						<div className="space-y-3 lg:hidden">
							{rows.map((row) => {
								const selected = selectedIdSet.has(row.id);
								const active = activeWorkloadId === row.id;
								const summary = getInventoryRowSummary(row);

								return (
									<article
										key={row.id}
										className={`list-card p-4 ${
											active
												? "border-sky-200 bg-sky-50/80 shadow-[0_20px_34px_rgba(14,116,144,0.1)]"
												: "bg-white/90"
										}`}
									>
										<div className="flex items-start gap-3">
											<SelectionCheckbox
												selected={selected}
												label={`Select ${row.vm.name}`}
												onToggle={() => onToggleSelection(row.id)}
												className="mt-1"
											/>
											<div className="min-w-0 flex-1 space-y-4">
												<div className="flex flex-wrap items-start justify-between gap-3">
													<div className="min-w-0 flex-1">
														<button
															type="button"
															onClick={() => onFocusWorkload(row.id)}
															className="text-left"
														>
															<WorkloadIdentity row={row} />
														</button>
													</div>
													<div className="flex flex-wrap justify-end gap-2">
														<StatusBadge tone={platformTone(row.vm.platform)}>
															{row.vm.platform}
														</StatusBadge>
														<StatusBadge tone={powerTone(row.vm.power_state)}>
															{row.vm.power_state}
														</StatusBadge>
														{selected ? (
															<StatusBadge tone="accent">selected</StatusBadge>
														) : null}
													</div>
												</div>

												<div className="grid gap-3 sm:grid-cols-2">
													<MobileDataPoint
														label="Placement"
														value={summary.placement}
													/>
													<MobileDataPoint
														label="Resource profile"
														value={summary.resource}
														detail={summary.resourceDetail}
													/>
													<MobileDataPoint
														label="Dependencies"
														value={summary.dependencies}
														detail={summary.dependenciesDetail}
													/>
													<MobileDataPoint
														label="Recency"
														value={summary.recency}
														detail={summary.recencyDetail}
													/>
												</div>

												<div className="flex flex-wrap gap-2">
													<StatusBadge tone={readinessTone(row.readiness)}>
														{row.readiness}
													</StatusBadge>
													<StatusBadge tone={riskTone(row.risk)}>
														{row.risk} risk
													</StatusBadge>
													{row.assessmentIncomplete ? (
														<StatusBadge tone="neutral">partial</StatusBadge>
													) : null}
												</div>

												<p className="text-sm leading-6 text-slate-600">
													{summary.assessment}
												</p>

												<div className="flex flex-wrap gap-2">
													<button
														type="button"
														onClick={() => onFocusWorkload(row.id)}
														className="operator-button-secondary px-3.5 py-2"
													>
														Inspect workload
													</button>
													<button
														type="button"
														onClick={() => onToggleSelection(row.id)}
														className="operator-button-secondary px-3.5 py-2"
													>
														{selected
															? "Remove from selection"
															: "Select workload"}
													</button>
												</div>
											</div>
										</div>
									</article>
								);
							})}
						</div>

						<div className="hidden overflow-hidden rounded-[26px] border border-slate-200/80 bg-white/80 shadow-[0_12px_28px_rgba(15,23,42,0.05)] lg:block">
							<div className="overflow-x-auto">
								<table className="min-w-full table-fixed border-collapse text-left text-sm">
									<thead className="bg-slate-50/95 text-xs uppercase tracking-[0.2em] text-slate-500 backdrop-blur">
										<tr>
											<th className="w-12 px-4 py-4">
												<input
													ref={masterCheckboxRef}
													type="checkbox"
													checked={allVisibleSelected}
													onChange={() => onToggleSelectAllVisible()}
													aria-label="Select visible workloads"
												/>
											</th>
											<SortableHeader
												label="Workload"
												sortKey="name"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[18rem]"
											/>
											<SortableHeader
												label="Placement"
												sortKey="platform"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[14rem]"
											/>
											<SortableHeader
												label="Resource profile"
												sortKey="cpu"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[12rem]"
											/>
											<SortableHeader
												label="Readiness"
												sortKey="readiness"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[12rem]"
											/>
											<SortableHeader
												label="Risk"
												sortKey="risk"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[12rem]"
											/>
											<SortableHeader
												label="Dependencies"
												sortKey="dependencies"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[10rem]"
											/>
											<SortableHeader
												label="Recency"
												sortKey="recency"
												activeSortKey={sortKey}
												sortDirection={sortDirection}
												onSortChange={onSortChange}
												className="w-[10rem]"
											/>
										</tr>
									</thead>
									<tbody className="divide-y divide-slate-200/80">
										{rows.map((row) => {
											const selected = selectedIdSet.has(row.id);
											const active = activeWorkloadId === row.id;
											const summary = getInventoryRowSummary(row);

											return (
												<tr
													key={row.id}
													className={`cursor-pointer align-top transition ${
														active
															? "bg-sky-50/80"
															: "bg-white/60 hover:bg-slate-50/90"
													}`}
													onClick={() => onFocusWorkload(row.id)}
												>
													<td className="px-4 py-4">
														<SelectionCheckbox
															selected={selected}
															label={`Select ${row.vm.name}`}
															onToggle={() => onToggleSelection(row.id)}
														/>
													</td>
													<td className="px-4 py-4">
														<WorkloadIdentity row={row} compact />
													</td>
													<td className="px-4 py-4">
														<div className="space-y-2">
															<div className="flex flex-wrap gap-2">
																<StatusBadge
																	tone={platformTone(row.vm.platform)}
																>
																	{row.vm.platform}
																</StatusBadge>
																<StatusBadge
																	tone={powerTone(row.vm.power_state)}
																>
																	{row.vm.power_state}
																</StatusBadge>
															</div>
															<p className="text-xs leading-5 text-slate-600">
																{summary.placement}
															</p>
														</div>
													</td>
													<td className="px-4 py-4">
														<div className="space-y-1 text-xs leading-5 text-slate-600">
															<p className="font-semibold text-ink">
																{summary.resource}
															</p>
															<p>{summary.resourceDetail}</p>
														</div>
													</td>
													<td className="px-4 py-4">
														<div className="space-y-2">
															<StatusBadge tone={readinessTone(row.readiness)}>
																{row.readiness}
															</StatusBadge>
															<p className="text-xs leading-5 text-slate-600">
																{summary.assessment}
															</p>
														</div>
													</td>
													<td className="px-4 py-4">
														<div className="space-y-2">
															<div className="flex flex-wrap gap-2">
																<StatusBadge tone={riskTone(row.risk)}>
																	{row.risk} risk
																</StatusBadge>
																{row.assessmentIncomplete ? (
																	<StatusBadge tone="neutral">
																		partial
																	</StatusBadge>
																) : null}
															</div>
															<p className="text-xs leading-5 text-slate-600">
																{summary.risk}
															</p>
														</div>
													</td>
													<td className="px-4 py-4">
														<div className="space-y-1 text-xs leading-5 text-slate-600">
															<p>{summary.dependencies}</p>
															<p>{summary.dependenciesDetail}</p>
														</div>
													</td>
													<td className="px-4 py-4">
														<div className="space-y-1 text-xs leading-5 text-slate-600">
															<p className="font-semibold text-ink">
																{summary.recency}
															</p>
															<p>{summary.recencyDetail}</p>
														</div>
													</td>
												</tr>
											);
										})}
									</tbody>
								</table>
							</div>
						</div>
					</>
				)}

				{pagination && pagination.total_pages > 1 && onPageChange ? (
					<PaginationControls
						currentPage={currentPage}
						totalPages={pagination.total_pages}
						totalItems={pagination.total}
						itemLabel="workload(s)"
						onPageChange={onPageChange}
					/>
				) : null}
			</div>
		</SectionCard>
	);
}

function SelectionCheckbox({
	selected,
	label,
	onToggle,
	className,
}: {
	selected: boolean;
	label: string;
	onToggle: () => void;
	className?: string;
}) {
	return (
		<input
			type="checkbox"
			checked={selected}
			onClick={(event) => event.stopPropagation()}
			onChange={onToggle}
			aria-label={label}
			className={className}
		/>
	);
}

function WorkloadIdentity({
	row,
	compact = false,
}: {
	row: InventoryAssessmentRow;
	compact?: boolean;
}) {
	return (
		<div className="space-y-1">
			<p
				className={`${compact ? "text-sm" : "text-base"} font-semibold text-ink`}
			>
				{row.vm.name}
			</p>
			<p className="text-xs leading-5 text-slate-600">
				{getWorkloadIdentityDetail(row)}
			</p>
		</div>
	);
}

function MobileDataPoint({
	label,
	value,
	detail,
}: {
	label: string;
	value: string;
	detail?: string;
}) {
	return (
		<div className="metric-surface gap-1 px-4 py-3">
			<span className="operator-kicker">{label}</span>
			<p className="mt-1 text-sm font-semibold text-ink">{value}</p>
			{detail ? (
				<p className="text-xs leading-5 text-slate-600">{detail}</p>
			) : null}
		</div>
	);
}

function getInventoryRowSummary(row: InventoryAssessmentRow) {
	return {
		placement: `${row.vm.host || "Unknown host"}${
			row.vm.cluster ? ` • ${row.vm.cluster}` : ""
		}`,
		resource: `${row.vm.cpu_count} vCPU • ${formatMemory(row.vm.memory_mb)} GB`,
		resourceDetail: `${formatStorage(row.storageTotalMB)} GB storage • ${row.vm.nics.length} NIC(s)`,
		dependencies: `${row.dependencies.networks.length} network • ${row.dependencies.datastores.length} storage`,
		dependenciesDetail: `${row.dependencies.backups.length} backup relationship(s)`,
		assessment: row.assessmentIncomplete
			? `Partial signals: ${row.missingSources.join(", ")}`
			: `${row.policyViolations.length} policy issue(s) and ${row.recommendations.length} recommendation(s) derived for this workload.`,
		risk: `Score ${row.riskScore} • ${row.riskReasons[0] ?? "No immediate derived issues"}`,
		recency: formatRelativeTime(row.discoveredAt),
		recencyDetail: `${row.snapshotCount} snapshot(s)`,
	};
}

function getWorkloadIdentityDetail(row: InventoryAssessmentRow) {
	return [row.vm.guest_os || "Guest OS unavailable", row.vm.folder]
		.filter(Boolean)
		.join(" • ");
}

function FilterSelect({
	label,
	value,
	onChange,
	options,
}: {
	label: string;
	value: string;
	onChange: (value: string) => void;
	options: Array<{ label: string; value: string }>;
}) {
	return (
		<label className="metric-surface">
			<span className="operator-kicker">{label}</span>
			<select
				className="operator-select mt-2"
				value={value}
				onChange={(event) => onChange(event.target.value)}
			>
				{options.map((option) => (
					<option key={option.value} value={option.value}>
						{option.label}
					</option>
				))}
			</select>
		</label>
	);
}

function SortableHeader({
	label,
	sortKey,
	activeSortKey,
	sortDirection,
	onSortChange,
	className,
}: {
	label: string;
	sortKey: InventorySortKey;
	activeSortKey: InventorySortKey;
	sortDirection: "asc" | "desc";
	onSortChange: (key: InventorySortKey) => void;
	className?: string;
}) {
	const active = activeSortKey === sortKey;
	const ariaSort = !active
		? "none"
		: sortDirection === "asc"
			? "ascending"
			: "descending";

	return (
		<th className={`px-4 py-4 ${className ?? ""}`} aria-sort={ariaSort}>
			<button
				type="button"
				className={`flex items-center gap-2 transition ${active ? "text-ink" : "text-slate-500 hover:text-ink"}`}
				onClick={() => onSortChange(sortKey)}
			>
				{label}
				<ArrowDownUp
					className={`h-3.5 w-3.5 transition ${active && sortDirection === "desc" ? "rotate-180" : ""}`}
				/>
			</button>
		</th>
	);
}

function platformTone(platform: Platform): StatusTone {
	switch (platform) {
		case "vmware":
			return "info";
		case "proxmox":
			return "warning";
		case "hyperv":
			return "accent";
		case "kvm":
			return "neutral";
		case "nutanix":
			return "success";
		default:
			return "neutral";
	}
}

function powerTone(powerState: string): StatusTone {
	switch (powerState) {
		case "on":
			return "success";
		case "off":
			return "neutral";
		case "suspended":
			return "warning";
		default:
			return "danger";
	}
}

function readinessTone(readiness: InventoryReadinessState): StatusTone {
	switch (readiness) {
		case "ready":
			return "success";
		case "needs-review":
			return "warning";
		default:
			return "danger";
	}
}

function riskTone(risk: InventoryRiskState): StatusTone {
	switch (risk) {
		case "low":
			return "success";
		case "medium":
			return "warning";
		default:
			return "danger";
	}
}

function formatMemory(memoryMB: number): string {
	return (memoryMB / 1024).toFixed(memoryMB >= 10240 ? 0 : 1);
}

function formatStorage(storageMB: number): string {
	return (storageMB / 1024).toFixed(storageMB >= 10240 ? 0 : 1);
}
