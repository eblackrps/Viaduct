import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { ArrowDownUp, Search } from "lucide-react";
import { StatusBadge, type StatusTone } from "./primitives/StatusBadge";
import type { Pagination, Platform } from "../types";
import {
	formatRelativeTime,
	type InventoryAssessmentRow,
	type InventoryFilterState,
	type InventoryReadinessState,
	type InventoryRiskState,
	type InventorySortKey,
} from "../features/inventory/inventoryModel";

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
		<section className="panel overflow-hidden p-5" aria-live="polite">
			<div className="flex flex-col gap-4">
				<div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
					<div>
						<div className="flex flex-wrap gap-2">
							<StatusBadge tone="accent">Operational inventory</StatusBadge>
							<StatusBadge tone="info">{filteredCount} visible</StatusBadge>
							<StatusBadge tone="neutral">{selectedCount} selected</StatusBadge>
						</div>
						<h2 className="mt-3 font-display text-2xl text-ink">
							Workload assessment table
						</h2>
						<p className="mt-2 text-sm leading-6 text-slate-600">
							{filteredCount.toLocaleString()} of {totalCount.toLocaleString()}{" "}
							workload(s) shown. {selectedCount.toLocaleString()} selected for
							planning handoff.
						</p>
					</div>

					<div className="flex flex-wrap items-center gap-2">
						<div className="operator-toggle">
							<button
								type="button"
								onClick={() => onFiltersChange({ scope: "all" })}
								className={`operator-toggle-button ${filters.scope === "all" ? "operator-toggle-button-active" : ""}`}
							>
								All workloads
							</button>
							<button
								type="button"
								onClick={() => onFiltersChange({ scope: "selected" })}
								className={`operator-toggle-button ${filters.scope === "selected" ? "operator-toggle-button-active" : ""}`}
								disabled={selectedCount === 0}
							>
								Selected only
							</button>
						</div>

						{hasActiveFilters && (
							<button
								type="button"
								onClick={onResetFilters}
								className="operator-button-secondary px-3 py-2"
							>
								Clear filters
							</button>
						)}

						{actions && (
							<div className="flex flex-wrap items-center gap-2">{actions}</div>
						)}
					</div>
				</div>

				<div className="grid gap-3 xl:grid-cols-[minmax(0,1.2fr)_repeat(4,minmax(150px,1fr))]">
					<label className="metric-surface">
						<span className="operator-kicker">Search</span>
						<span className="mt-2 flex items-center gap-2 rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-500">
							<Search className="h-4 w-4" />
							<input
								className="w-full border-none bg-transparent outline-none"
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

				{selectedCount > 0 && (
					<div className="metric-surface flex flex-wrap items-center gap-2 text-sm text-slate-600">
						<StatusBadge tone="accent">{selectedCount} selected</StatusBadge>
						<button
							type="button"
							onClick={onToggleSelectAllVisible}
							className="operator-button-secondary px-3 py-2"
						>
							{allVisibleSelected ? "Unselect visible" : "Select visible"}
						</button>
						<button
							type="button"
							onClick={onClearSelection}
							className="operator-button-secondary px-3 py-2"
						>
							Clear selection
						</button>
					</div>
				)}
			</div>

			{error && (
				<p className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
					{error}
				</p>
			)}
			{refreshing && rows.length > 0 && (
				<p className="mt-4 rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
					Refreshing dependency and lifecycle assessment signals for the visible
					workloads.
				</p>
			)}

			{loading && rows.length === 0 ? (
				<div className="mt-5 rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
					Loading workload inventory and assessment signals...
				</div>
			) : rows.length === 0 ? (
				<div className="mt-5 rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
					{filters.scope === "selected"
						? "No selected workloads match the current search or filters."
						: "No workloads match the current search or filters."}
				</div>
			) : (
				<div className="mt-5 overflow-x-auto">
					<table className="min-w-full border-separate border-spacing-y-2 text-left">
						<thead>
							<tr className="text-xs uppercase tracking-[0.22em] text-slate-500">
								<th className="px-3 py-2">
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
								/>
								<SortableHeader
									label="Placement"
									sortKey="platform"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
								<SortableHeader
									label="Resource profile"
									sortKey="cpu"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
								<SortableHeader
									label="Readiness"
									sortKey="readiness"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
								<SortableHeader
									label="Risk"
									sortKey="risk"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
								<SortableHeader
									label="Dependencies"
									sortKey="dependencies"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
								<SortableHeader
									label="Recency"
									sortKey="recency"
									activeSortKey={sortKey}
									sortDirection={sortDirection}
									onSortChange={onSortChange}
								/>
							</tr>
						</thead>
						<tbody>
							{rows.map((row) => {
								const selected = selectedIdSet.has(row.id);
								const active = activeWorkloadId === row.id;

								return (
									<tr
										key={row.id}
										className={`cursor-pointer rounded-2xl border text-sm text-slate-700 transition ${
											active
												? "border-sky-200 bg-sky-50 ring-1 ring-sky-200"
												: "border-transparent bg-slate-50/80 hover:border-slate-200 hover:bg-slate-100"
										}`}
										onClick={() => onFocusWorkload(row.id)}
									>
										<td className="rounded-l-2xl px-3 py-3">
											<input
												type="checkbox"
												checked={selected}
												readOnly
												onClick={(event) => {
													event.stopPropagation();
													onToggleSelection(row.id);
												}}
												aria-label={`Select ${row.vm.name}`}
											/>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-1">
												<p className="font-semibold text-ink">{row.vm.name}</p>
												<p className="text-xs text-slate-600">
													{row.vm.guest_os || "Guest OS unavailable"}
													{row.vm.folder ? ` • ${row.vm.folder}` : ""}
												</p>
											</div>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-2">
												<div className="flex flex-wrap gap-2">
													<StatusBadge tone={platformTone(row.vm.platform)}>
														{row.vm.platform}
													</StatusBadge>
													<StatusBadge tone={powerTone(row.vm.power_state)}>
														{row.vm.power_state}
													</StatusBadge>
												</div>
												<p className="text-xs text-slate-600">
													{row.vm.host || "Unknown host"}{" "}
													{row.vm.cluster ? `• ${row.vm.cluster}` : ""}
												</p>
											</div>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-1 text-xs text-slate-600">
												<p className="font-semibold text-ink">
													{row.vm.cpu_count} vCPU •{" "}
													{formatMemory(row.vm.memory_mb)} GB
												</p>
												<p>
													{formatStorage(row.storageTotalMB)} GB storage •{" "}
													{row.vm.nics.length} NIC(s)
												</p>
											</div>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-2">
												<div className="flex flex-wrap gap-2">
													<StatusBadge tone={readinessTone(row.readiness)}>
														{row.readiness}
													</StatusBadge>
												</div>
												<p className="text-xs text-slate-600">
													{row.assessmentIncomplete
														? `Partial signals: ${row.missingSources.join(", ")}`
														: `${row.policyViolations.length} policy • ${row.recommendations.length} recommendation(s)`}
												</p>
											</div>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-2">
												<div className="flex flex-wrap gap-2">
													<StatusBadge tone={riskTone(row.risk)}>
														{row.risk} risk
													</StatusBadge>
													{row.assessmentIncomplete ? (
														<StatusBadge tone="neutral">partial</StatusBadge>
													) : null}
												</div>
												<p className="text-xs text-slate-600">
													Score {row.riskScore} •{" "}
													{row.riskReasons[0] ?? "No immediate derived issues"}
												</p>
											</div>
										</td>
										<td className="px-3 py-3">
											<div className="space-y-1 text-xs text-slate-600">
												<p>Networks: {row.dependencies.networks.length}</p>
												<p>Storage: {row.dependencies.datastores.length}</p>
												<p>Backups: {row.dependencies.backups.length}</p>
											</div>
										</td>
										<td className="rounded-r-2xl px-3 py-3">
											<div className="space-y-1 text-xs text-slate-600">
												<p className="font-semibold text-ink">
													{formatRelativeTime(row.discoveredAt)}
												</p>
												<p>{row.snapshotCount} snapshot(s)</p>
											</div>
										</td>
									</tr>
								);
							})}
						</tbody>
					</table>
				</div>
			)}

			{pagination && pagination.total_pages > 1 && onPageChange && (
				<div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
					<p>
						Page {currentPage} of {pagination.total_pages} •{" "}
						{pagination.total.toLocaleString()} workload(s)
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
}: {
	label: string;
	sortKey: InventorySortKey;
	activeSortKey: InventorySortKey;
	sortDirection: "asc" | "desc";
	onSortChange: (key: InventorySortKey) => void;
}) {
	const active = activeSortKey === sortKey;
	const ariaSort = !active
		? "none"
		: sortDirection === "asc"
			? "ascending"
			: "descending";

	return (
		<th className="px-3 py-2" aria-sort={ariaSort}>
			<button
				type="button"
				className="flex items-center gap-2"
				onClick={() => onSortChange(sortKey)}
			>
				{label}
				<ArrowDownUp
					className={`h-3.5 w-3.5 ${active ? "text-ink" : ""} ${active && sortDirection === "desc" ? "rotate-180" : ""}`}
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
