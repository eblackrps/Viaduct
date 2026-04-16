import type { Platform, VirtualMachine } from "../../types";

type WorkloadIdentityInput = Pick<
	VirtualMachine,
	"platform" | "source_ref" | "id" | "name"
>;

export function getVirtualMachineIdentity(vm: WorkloadIdentityInput): string {
	const platform = normalizeToken(vm.platform);
	const primaryToken =
		normalizeToken(vm.source_ref) ||
		normalizeToken(vm.id) ||
		normalizeToken(vm.name) ||
		"unknown-workload";

	return `${platform}:${primaryToken}`;
}

export function getVirtualMachineLookupKeys(
	vm: Partial<WorkloadIdentityInput>,
): string[] {
	const platform = normalizeToken(vm.platform);
	const keys = new Set<string>();
	const candidates = [
		["source_ref", vm.source_ref],
		["id", vm.id],
		["name", vm.name],
	] as const;

	for (const [field, value] of candidates) {
		const normalized = normalizeToken(value);
		if (!normalized) {
			continue;
		}

		keys.add(`${field}:${normalized}`);
		if (platform) {
			keys.add(`${platform}:${field}:${normalized}`);
		}
	}

	return Array.from(keys);
}

export function getPlatformScopeLabel(
	platform: Platform | "" | null | undefined,
): string {
	return normalizeToken(platform) || "mixed";
}

function normalizeToken(value: string | null | undefined): string {
	return value?.trim().toLowerCase() ?? "";
}
