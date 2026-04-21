import { useDeferredValue, useEffect, useMemo, useState } from "react";
import type { Platform } from "../../types";
import {
	filterInventoryRows,
	sortInventoryRows,
	summarizeInventoryRows,
	type InventoryAssessmentRow,
	type InventoryFilterState,
	type InventorySortKey,
} from "./inventoryModel";

const defaultFilters: InventoryFilterState = {
	search: "",
	platform: "all",
	power: "all",
	readiness: "all",
	risk: "all",
	scope: "all",
};

interface InventoryWorkspaceState {
	filters: InventoryFilterState;
	sortKey: InventorySortKey;
	sortDirection: "asc" | "desc";
	filteredRows: InventoryAssessmentRow[];
	selectedIds: string[];
	selectedRows: InventoryAssessmentRow[];
	visibleSelectedCount: number;
	availablePlatforms: Platform[];
	activeRow: InventoryAssessmentRow | null;
	summary: ReturnType<typeof summarizeInventoryRows>;
	hasActiveFilters: boolean;
	setActiveWorkloadId: (id: string) => void;
	updateFilters: (updates: Partial<InventoryFilterState>) => void;
	changeSort: (key: InventorySortKey) => void;
	toggleSelection: (id: string) => void;
	toggleSelectAllVisible: () => void;
	clearSelection: () => void;
	replaceSelection: (ids: string[]) => void;
	resetFilters: () => void;
}

export function useInventoryWorkspace(
	rows: InventoryAssessmentRow[],
	initialSelectedIDs?: readonly string[],
): InventoryWorkspaceState {
	const [filters, setFilters] = useState<InventoryFilterState>(defaultFilters);
	const [sortKey, setSortKey] = useState<InventorySortKey>("risk");
	const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");
	const [selectedIds, setSelectedIds] = useState<string[]>(() =>
		initialSelectedIDs ? Array.from(new Set(initialSelectedIDs)) : [],
	);
	const [activeWorkloadId, setActiveWorkloadId] = useState<string | null>(null);
	const deferredSearch = useDeferredValue(filters.search);
	const selectedIdSet = useMemo(() => new Set(selectedIds), [selectedIds]);

	const filteredRows = useMemo(() => {
		const filtered = filterInventoryRows(rows, {
			...filters,
			scope: "all",
			search: deferredSearch,
		});
		const scopedRows =
			filters.scope === "selected"
				? filtered.filter((row) => selectedIdSet.has(row.id))
				: filtered;

		return sortInventoryRows(scopedRows, sortKey, sortDirection);
	}, [deferredSearch, filters, rows, selectedIdSet, sortDirection, sortKey]);
	const hasActiveFilters = useMemo(
		() =>
			filters.scope !== defaultFilters.scope ||
			filters.search !== defaultFilters.search ||
			filters.platform !== defaultFilters.platform ||
			filters.power !== defaultFilters.power ||
			filters.readiness !== defaultFilters.readiness ||
			filters.risk !== defaultFilters.risk,
		[filters],
	);
	const selectedRows = useMemo(
		() => rows.filter((row) => selectedIdSet.has(row.id)),
		[rows, selectedIdSet],
	);
	const visibleSelectedCount = useMemo(
		() => filteredRows.filter((row) => selectedIdSet.has(row.id)).length,
		[filteredRows, selectedIdSet],
	);
	const availablePlatforms = useMemo<Platform[]>(
		() =>
			[...new Set(rows.map((row) => row.vm.platform))].sort((left, right) =>
				left.localeCompare(right),
			),
		[rows],
	);
	const activeRow = useMemo(
		() => filteredRows.find((row) => row.id === activeWorkloadId) ?? null,
		[activeWorkloadId, filteredRows],
	);
	const summary = useMemo(
		() => summarizeInventoryRows(rows, selectedRows.length),
		[rows, selectedRows.length],
	);

	useEffect(() => {
		setSelectedIds((current) =>
			current.filter((id) => rows.some((row) => row.id === id)),
		);
	}, [rows]);

	useEffect(() => {
		if (initialSelectedIDs === undefined) {
			return;
		}

		setSelectedIds(Array.from(new Set(initialSelectedIDs)));
	}, [initialSelectedIDs]);

	useEffect(() => {
		if (
			activeWorkloadId &&
			filteredRows.some((row) => row.id === activeWorkloadId)
		) {
			return;
		}
		setActiveWorkloadId(filteredRows[0]?.id ?? null);
	}, [activeWorkloadId, filteredRows]);

	function changeSort(nextKey: InventorySortKey) {
		if (sortKey === nextKey) {
			setSortDirection((current) => (current === "asc" ? "desc" : "asc"));
			return;
		}

		setSortKey(nextKey);
		setSortDirection(
			nextKey === "name" || nextKey === "platform" ? "asc" : "desc",
		);
	}

	function updateFilters(updates: Partial<InventoryFilterState>) {
		setFilters((current) => ({ ...current, ...updates }));
	}

	function toggleSelection(id: string) {
		setSelectedIds((current) =>
			current.includes(id)
				? current.filter((item) => item !== id)
				: [...current, id],
		);
	}

	function toggleSelectAllVisible() {
		const visibleIds = filteredRows.map((row) => row.id);
		const allSelected =
			visibleIds.length > 0 && visibleIds.every((id) => selectedIdSet.has(id));

		setSelectedIds((current) => {
			if (allSelected) {
				return current.filter((id) => !visibleIds.includes(id));
			}

			return Array.from(new Set([...current, ...visibleIds]));
		});
	}

	function clearSelection() {
		setSelectedIds([]);
	}

	function replaceSelection(ids: string[]) {
		setSelectedIds(Array.from(new Set(ids)));
	}

	function resetFilters() {
		setFilters(defaultFilters);
	}

	return {
		filters,
		sortKey,
		sortDirection,
		filteredRows,
		selectedIds,
		selectedRows,
		visibleSelectedCount,
		availablePlatforms,
		activeRow,
		summary,
		hasActiveFilters,
		setActiveWorkloadId,
		updateFilters,
		changeSort,
		toggleSelection,
		toggleSelectAllVisible,
		clearSelection,
		replaceSelection,
		resetFilters,
	};
}
