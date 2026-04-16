import { describe, expect, it } from "vitest";
import { buildInventoryAssessmentRows } from "./inventoryModel";
import type { VirtualMachine } from "../../types";

describe("buildInventoryAssessmentRows", () => {
	it("tolerates null NIC address lists from the API payload", () => {
		const vm = {
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
					ip_addresses: null,
				},
			],
			host: "kvm-lab-01",
		} satisfies VirtualMachine;

		const rows = buildInventoryAssessmentRows(
			[vm],
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

		expect(rows).toHaveLength(1);
		expect(rows[0].searchDocument).toContain("ubuntu-web-01");
		expect(rows[0].connectedNetworks).toEqual(["vmbr0"]);
	});
});
