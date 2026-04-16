import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { VirtualMachine } from "../../types";
import { buildInventoryAssessmentRows } from "./inventoryModel";
import { useInventoryWorkspace } from "./useInventoryWorkspace";

const virtualMachines = [
	{
		id: "vm-1",
		name: "ubuntu-web-01",
		platform: "kvm",
		power_state: "on",
		cpu_count: 2,
		memory_mb: 4096,
		disks: [
			{
				id: "disk-1",
				name: "vda",
				size_mb: 40960,
				thin: false,
				storage_backend: "local-lvm",
			},
		],
		nics: [
			{
				id: "nic-1",
				name: "eth0",
				mac_address: "00:11:22:33:44:55",
				network: "vmbr0",
				connected: true,
				ip_addresses: ["192.168.122.10"],
			},
		],
		host: "kvm-lab-01",
	} satisfies VirtualMachine,
	{
		id: "vm-2",
		name: "windows-app-01",
		platform: "kvm",
		power_state: "on",
		cpu_count: 4,
		memory_mb: 8192,
		disks: [
			{
				id: "disk-2",
				name: "vda",
				size_mb: 81920,
				thin: false,
				storage_backend: "local-lvm",
			},
		],
		nics: [
			{
				id: "nic-2",
				name: "eth0",
				mac_address: "00:11:22:33:44:66",
				network: "vmbr0",
				connected: true,
				ip_addresses: ["192.168.122.11"],
			},
		],
		host: "kvm-lab-01",
	} satisfies VirtualMachine,
];

function createRows() {
	return buildInventoryAssessmentRows(
		virtualMachines,
		null,
		null,
		null,
		{
			graph: false,
			policies: false,
			remediation: false,
		},
		"2026-04-16T12:00:00Z",
	);
}

describe("useInventoryWorkspace", () => {
	it("preserves local selection when the caller does not provide external selection state", () => {
		const initialRows = createRows();
		const { result, rerender } = renderHook(
			(props: {
				rows: ReturnType<typeof createRows>;
				initialSelectedIDs?: readonly string[];
			}) => useInventoryWorkspace(props.rows, props.initialSelectedIDs),
			{
				initialProps: {
					rows: initialRows,
					initialSelectedIDs: undefined,
				},
			},
		);

		act(() => {
			result.current.toggleSelection(initialRows[0].id);
		});

		expect(result.current.selectedIds).toEqual([initialRows[0].id]);

		rerender({
			rows: createRows(),
			initialSelectedIDs: undefined,
		});

		expect(result.current.selectedIds).toEqual([initialRows[0].id]);
	});

	it("syncs saved selections when a caller provides external selection state", () => {
		const rows = createRows();
		const { result, rerender } = renderHook(
			(props: {
				rows: ReturnType<typeof createRows>;
				initialSelectedIDs?: readonly string[];
			}) => useInventoryWorkspace(props.rows, props.initialSelectedIDs),
			{
				initialProps: {
					rows,
					initialSelectedIDs: [rows[0].id],
				},
			},
		);

		expect(result.current.selectedIds).toEqual([rows[0].id]);

		rerender({
			rows,
			initialSelectedIDs: [rows[0].id, rows[1].id],
		});

		expect(result.current.selectedIds).toEqual([rows[0].id, rows[1].id]);
	});
});
