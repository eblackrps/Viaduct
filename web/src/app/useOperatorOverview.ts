import { useCallback, useEffect, useRef, useState } from "react";
import {
	getInventory,
	getSnapshots,
	getTenantSummary,
	isAbortError,
	listMigrations,
} from "../api";
import type {
	DiscoveryResult,
	InventoryListResponse,
	MigrationMeta,
	PaginatedResponse,
	Pagination,
	SnapshotMeta,
	TenantSummary,
} from "../types";

export interface OperatorOverviewErrors {
	inventory?: string;
	snapshots?: string;
	migrations?: string;
	summary?: string;
}

export interface OperatorOverviewState {
	inventory: DiscoveryResult | null;
	inventoryPagination: Pagination | null;
	inventoryPage: number;
	snapshots: SnapshotMeta[];
	snapshotsPagination: Pagination | null;
	snapshotsPage: number;
	latestSnapshot: SnapshotMeta | null;
	migrations: MigrationMeta[];
	migrationsPagination: Pagination | null;
	migrationsPage: number;
	summary: TenantSummary | null;
	loading: boolean;
	refreshing: boolean;
	refreshToken: number;
	errors: OperatorOverviewErrors;
	error: string | null;
	refresh: () => Promise<void>;
	setInventoryPage: (page: number) => void;
	setSnapshotsPage: (page: number) => void;
	setMigrationsPage: (page: number) => void;
}

export function useOperatorOverview(): OperatorOverviewState {
	const [inventory, setInventory] = useState<DiscoveryResult | null>(null);
	const [inventoryPagination, setInventoryPagination] =
		useState<Pagination | null>(null);
	const [inventoryPage, setInventoryPage] = useState(1);
	const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([]);
	const [snapshotsPagination, setSnapshotsPagination] =
		useState<Pagination | null>(null);
	const [snapshotsPage, setSnapshotsPage] = useState(1);
	const [migrations, setMigrations] = useState<MigrationMeta[]>([]);
	const [migrationsPagination, setMigrationsPagination] =
		useState<Pagination | null>(null);
	const [migrationsPage, setMigrationsPage] = useState(1);
	const [summary, setSummary] = useState<TenantSummary | null>(null);
	const [loading, setLoading] = useState(true);
	const [refreshing, setRefreshing] = useState(false);
	const [refreshToken, setRefreshToken] = useState(0);
	const [errors, setErrors] = useState<OperatorOverviewErrors>({});
	const [error, setError] = useState<string | null>(null);
	const hasLoadedRef = useRef(false);
	const requestSequenceRef = useRef(0);
	const refreshControllerRef = useRef<AbortController | null>(null);
	const mountedRef = useRef(true);

	const refresh = useCallback(async () => {
		refreshControllerRef.current?.abort();

		const controller = new AbortController();
		refreshControllerRef.current = controller;
		const requestSequence = requestSequenceRef.current + 1;
		requestSequenceRef.current = requestSequence;
		const initialLoad = !hasLoadedRef.current;
		setLoading(initialLoad);
		setRefreshing(!initialLoad);

		try {
			const [inventoryResult, snapshotResult, migrationResult, summaryResult] =
				await Promise.allSettled([
					getInventory(undefined, {
						page: inventoryPage,
						perPage: 50,
						signal: controller.signal,
					}),
					getSnapshots({
						page: snapshotsPage,
						perPage: 50,
						signal: controller.signal,
					}),
					listMigrations({
						page: migrationsPage,
						perPage: 50,
						signal: controller.signal,
					}),
					getTenantSummary({ signal: controller.signal }),
				]);

			if (
				controller.signal.aborted ||
				!mountedRef.current ||
				requestSequence !== requestSequenceRef.current
			) {
				return;
			}

			const nextErrors: OperatorOverviewErrors = {};

			if (inventoryResult.status === "fulfilled") {
				applyInventoryResult(
					inventoryResult.value,
					setInventory,
					setInventoryPagination,
				);
			} else if (!isAbortError(inventoryResult.reason)) {
				nextErrors.inventory = errorMessage(
					"inventory",
					inventoryResult.reason,
				);
			}

			if (snapshotResult.status === "fulfilled") {
				applyPagedItems(
					snapshotResult.value,
					setSnapshots,
					setSnapshotsPagination,
				);
			} else if (!isAbortError(snapshotResult.reason)) {
				nextErrors.snapshots = errorMessage(
					"discovery snapshots",
					snapshotResult.reason,
				);
			}

			if (migrationResult.status === "fulfilled") {
				applyPagedItems(
					migrationResult.value,
					setMigrations,
					setMigrationsPagination,
				);
			} else if (!isAbortError(migrationResult.reason)) {
				nextErrors.migrations = errorMessage(
					"migration history",
					migrationResult.reason,
				);
			}

			if (summaryResult.status === "fulfilled") {
				setSummary(summaryResult.value);
			} else if (!isAbortError(summaryResult.reason)) {
				nextErrors.summary = errorMessage(
					"tenant summary",
					summaryResult.reason,
				);
			}

			const settledErrors: OperatorOverviewErrors = {
				inventory: nextErrors.inventory,
				snapshots: nextErrors.snapshots,
				migrations: nextErrors.migrations,
				summary: nextErrors.summary,
			};
			setErrors(settledErrors);
			setError(composeGlobalError(settledErrors));
			hasLoadedRef.current = true;
			if (!initialLoad) {
				setRefreshToken((current) => current + 1);
			}
		} finally {
			if (
				mountedRef.current &&
				requestSequence === requestSequenceRef.current &&
				refreshControllerRef.current === controller
			) {
				setLoading(false);
				setRefreshing(false);
			}
		}
	}, [inventoryPage, migrationsPage, snapshotsPage]);

	useEffect(() => {
		void refresh();
		return () => {
			refreshControllerRef.current?.abort();
		};
	}, [refresh]);

	useEffect(() => {
		return () => {
			mountedRef.current = false;
			refreshControllerRef.current?.abort();
		};
	}, []);

	return {
		inventory,
		inventoryPagination,
		inventoryPage,
		snapshots,
		snapshotsPagination,
		snapshotsPage,
		latestSnapshot: snapshots[0] ?? null,
		migrations,
		migrationsPagination,
		migrationsPage,
		summary,
		loading,
		refreshing,
		refreshToken,
		errors,
		error,
		refresh,
		setInventoryPage,
		setSnapshotsPage,
		setMigrationsPage,
	};
}

function applyInventoryResult(
	payload: InventoryListResponse,
	setInventory: (value: DiscoveryResult | null) => void,
	setPagination: (value: Pagination | null) => void,
) {
	setInventory(payload.inventory);
	setPagination(payload.pagination);
}

function applyPagedItems<T>(
	payload: PaginatedResponse<T>,
	setItems: (value: T[]) => void,
	setPagination: (value: Pagination | null) => void,
) {
	setItems(payload.items);
	setPagination(payload.pagination);
}

function composeGlobalError(errors: OperatorOverviewErrors): string | null {
	const primaryMessages = [
		errors.inventory,
		errors.snapshots,
		errors.migrations,
		errors.summary,
	].filter(Boolean);
	return primaryMessages.length > 0 ? primaryMessages.join(" ") : null;
}

function errorMessage(scope: string, reason: unknown): string {
	if (reason instanceof Error && reason.message.trim() !== "") {
		return `Unable to load ${scope}: ${reason.message}`;
	}
	return `Unable to load ${scope}.`;
}
