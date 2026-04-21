import type {
	MigrationExecutionRequest,
	MigrationSpec,
	Platform,
	VirtualMachine,
} from "../../types";

interface ExecutionWindowState {
	kind: "unset" | "open" | "not-started" | "closed";
	label: string;
	summary: string;
}

export function buildMigrationSpec({
	migrationName,
	sourcePlatform,
	sourceAddress,
	targetPlatform,
	targetAddress,
	defaultHost,
	defaultStorage,
	selectedWorkloads,
	networkMap,
	parallelism,
	waveSize,
	dependencyAware,
	shutdownSource,
	verifyBoot,
	scheduledStart,
	scheduledEnd,
	approvalRequired,
	approvedBy,
	approvalTicket,
	approvalRecordedAt,
}: {
	migrationName: string;
	sourcePlatform: Platform;
	sourceAddress: string;
	targetPlatform: Platform;
	targetAddress: string;
	defaultHost: string;
	defaultStorage: string;
	selectedWorkloads: VirtualMachine[];
	networkMap: Record<string, string>;
	parallelism: number;
	waveSize: number;
	dependencyAware: boolean;
	shutdownSource: boolean;
	verifyBoot: boolean;
	scheduledStart: string;
	scheduledEnd: string;
	approvalRequired: boolean;
	approvedBy: string;
	approvalTicket: string;
	approvalRecordedAt: string;
}): MigrationSpec {
	return {
		name: migrationName.trim() || "dashboard-migration",
		source: { address: sourceAddress.trim(), platform: sourcePlatform },
		target: {
			address: targetAddress.trim(),
			platform: targetPlatform,
			default_host: defaultHost.trim() || undefined,
			default_storage: defaultStorage.trim() || undefined,
		},
		workloads: selectedWorkloads.map((vm) => ({
			match: { name_pattern: toExactNamePattern(vm.name) },
			overrides: {
				target_host: defaultHost.trim() || undefined,
				target_storage: defaultStorage.trim() || undefined,
				network_map:
					Object.keys(networkMap).length > 0
						? Object.fromEntries(
								Object.entries(networkMap).filter(
									([, value]) => value.trim() !== "",
								),
							)
						: undefined,
			},
		})),
		options: {
			parallel: Math.max(1, parallelism),
			shutdown_source: shutdownSource,
			verify_boot: verifyBoot,
			window:
				scheduledStart || scheduledEnd
					? {
							not_before: scheduledStart
								? new Date(scheduledStart).toISOString()
								: undefined,
							not_after: scheduledEnd
								? new Date(scheduledEnd).toISOString()
								: undefined,
						}
					: undefined,
			approval: approvalRequired
				? {
						required: true,
						approved_by: approvedBy.trim() || undefined,
						approved_at:
							approvedBy.trim() && approvalRecordedAt
								? approvalRecordedAt
								: undefined,
						ticket: approvalTicket.trim() || undefined,
					}
				: undefined,
			waves: { size: Math.max(1, waveSize), dependency_aware: dependencyAware },
		},
	};
}

export function validateMigrationDraft(
	selectedCount: number,
	sourceAddress: string,
	targetAddress: string,
): string | null {
	if (selectedCount === 0) {
		return "Select at least one workload before validation or plan save.";
	}
	if (sourceAddress.trim() === "") {
		return "Source address is required before validation or plan save.";
	}
	if (targetAddress.trim() === "") {
		return "Target address is required before validation or plan save.";
	}
	return null;
}

export function buildExecutionPayload(
	approvedBy: string,
	ticket: string,
): MigrationExecutionRequest | undefined {
	const payload = {
		approved_by: approvedBy.trim() || undefined,
		ticket: ticket.trim() || undefined,
	};
	return payload.approved_by || payload.ticket ? payload : undefined;
}

export function resolveExecutionWindowState(
	start: string,
	end: string,
): ExecutionWindowState {
	const now = Date.now();
	const startDate = start ? new Date(start) : null;
	const endDate = end ? new Date(end) : null;
	const startMs =
		startDate && !Number.isNaN(startDate.getTime())
			? startDate.getTime()
			: null;
	const endMs =
		endDate && !Number.isNaN(endDate.getTime()) ? endDate.getTime() : null;

	if (!startMs && !endMs) {
		return {
			kind: "unset",
			label: "No execution window",
			summary: "No execution window",
		};
	}
	if (startMs && now < startMs) {
		return {
			kind: "not-started",
			label: startDate!.toLocaleString(),
			summary: "Window not open yet",
		};
	}
	if (endMs && now > endMs) {
		return {
			kind: "closed",
			label: endDate!.toLocaleString(),
			summary: "Execution window closed",
		};
	}
	return {
		kind: "open",
		label: (startDate ?? endDate)!.toLocaleString(),
		summary: "Execution window open",
	};
}

function toExactNamePattern(name: string): string {
	return `regex:^${escapeRegExp(name)}$`;
}

function escapeRegExp(value: string): string {
	return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
