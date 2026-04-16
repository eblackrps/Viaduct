import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const apiMocks = vi.hoisted(() => ({
	getInventory: vi.fn(),
	getSnapshots: vi.fn(),
	listMigrations: vi.fn(),
	getTenantSummary: vi.fn(),
}));

vi.mock("../api", () => ({
	getInventory: apiMocks.getInventory,
	getSnapshots: apiMocks.getSnapshots,
	listMigrations: apiMocks.listMigrations,
	getTenantSummary: apiMocks.getTenantSummary,
	isAbortError: (reason: unknown) =>
		reason instanceof DOMException && reason.name === "AbortError",
}));

import { useOperatorOverview } from "./useOperatorOverview";

describe("useOperatorOverview", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		apiMocks.getInventory.mockResolvedValue({
			inventory: {
				source: "lab",
				platform: "vmware",
				vms: [
					{
						id: "vm-1",
						name: "alpha",
						platform: "vmware",
						power_state: "on",
						cpu_count: 2,
						memory_mb: 4096,
						disks: [],
						nics: [],
						host: "esx-01",
					},
				],
			},
			pagination: { total: 75, page: 1, per_page: 50, total_pages: 2 },
		});
		apiMocks.getSnapshots.mockResolvedValue({
			items: [
				{
					id: "snap-1",
					source: "lab",
					platform: "vmware",
					vm_count: 1,
					discovered_at: "2026-04-16T00:00:00Z",
				},
			],
			pagination: { total: 2, page: 1, per_page: 50, total_pages: 1 },
		});
		apiMocks.listMigrations.mockResolvedValue({
			items: [
				{
					id: "mig-1",
					tenant_id: "default",
					spec_name: "wave-1",
					phase: "plan",
					started_at: "2026-04-16T00:00:00Z",
					updated_at: "2026-04-16T00:05:00Z",
				},
			],
			pagination: { total: 3, page: 1, per_page: 50, total_pages: 1 },
		});
		apiMocks.getTenantSummary.mockResolvedValue({
			tenant_id: "default",
			workload_count: 75,
			snapshot_count: 2,
			active_migrations: 1,
			completed_migrations: 0,
			failed_migrations: 0,
			pending_approvals: 0,
			recommendation_count: 0,
			platform_counts: { vmware: 75 },
		});
	});

	it("loads overview data and exposes pagination state", async () => {
		const { result } = renderHook(() => useOperatorOverview());

		expect(result.current.loading).toBe(true);

		await waitFor(() => expect(result.current.loading).toBe(false));

		expect(result.current.inventory?.vms).toHaveLength(1);
		expect(result.current.inventoryPagination?.total).toBe(75);
		expect(result.current.snapshots).toHaveLength(1);
		expect(result.current.migrations).toHaveLength(1);
		expect(result.current.summary?.tenant_id).toBe("default");
	});

	it("refreshes when the inventory page changes", async () => {
		const { result } = renderHook(() => useOperatorOverview());
		await waitFor(() => expect(result.current.loading).toBe(false));

		act(() => {
			result.current.setInventoryPage(2);
		});

		await waitFor(() => {
			const calls = apiMocks.getInventory.mock.calls as Array<
				[undefined, { page: number; perPage: number; signal: AbortSignal }]
			>;
			const [, options] = calls[calls.length - 1];
			expect(options.page).toBe(2);
			expect(options.perPage).toBe(50);
			expect(options.signal).toBeInstanceOf(AbortSignal);
		});
	});
});
