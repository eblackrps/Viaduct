import type { ReactNode } from "react";
import { getRouteHref } from "../../app/navigation";
import { EmptyState } from "../../components/primitives/EmptyState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import {
	StatusBadge,
	type StatusTone,
} from "../../components/primitives/StatusBadge";
import type { SnapshotMeta } from "../../types";
import {
	formatRelativeTime,
	formatTimestamp,
	type InventoryAssessmentRow,
	type InventoryReadinessState,
	type InventoryRiskState,
} from "./inventoryModel";

interface WorkloadDetailPanelProps {
	row: InventoryAssessmentRow | null;
	latestSnapshot: SnapshotMeta | null;
	assessmentErrors?: {
		graph?: string;
		policies?: string;
		remediation?: string;
	};
	onPrimaryAction: (row: InventoryAssessmentRow) => void | Promise<void>;
	primaryActionLabel?: string;
	primaryActionPendingLabel?: string;
	primaryActionBusy?: boolean;
}

export function WorkloadDetailPanel({
	row,
	latestSnapshot,
	assessmentErrors,
	onPrimaryAction,
	primaryActionLabel = "Open migration plan",
	primaryActionPendingLabel,
	primaryActionBusy = false,
}: WorkloadDetailPanelProps) {
	if (!row) {
		return (
			<SectionCard
				title="Workload detail"
				description="Select a workload from the inventory table to inspect its operational detail."
			>
				<EmptyState
					title="No workload selected"
					message="This panel will show overview, dependency, risk, and activity details for the currently focused workload."
				/>
			</SectionCard>
		);
	}

	const { vm } = row;

	return (
		<SectionCard
			title="Workload detail"
			description="Operator-facing workload context derived from normalized inventory, lifecycle signals, and the current dependency graph."
			actions={
				<>
					<button
						type="button"
						onClick={() => void onPrimaryAction(row)}
						className="operator-button"
						disabled={primaryActionBusy}
					>
						{primaryActionBusy && primaryActionPendingLabel
							? primaryActionPendingLabel
							: primaryActionLabel}
					</button>
					<a
						href={getRouteHref("/graph")}
						className="operator-button-secondary"
					>
						Open graph
					</a>
				</>
			}
		>
			<div className="space-y-5">
				<div className="panel-muted px-4 py-4">
					<div className="flex flex-wrap items-start justify-between gap-3">
						<div>
							<h3 className="font-display text-[1.7rem] tracking-[-0.03em] text-ink">
								{vm.name}
							</h3>
							<p className="mt-2 text-sm leading-6 text-slate-600">
								{vm.guest_os || "Guest OS unavailable"}
								{vm.folder ? ` • ${vm.folder}` : ""}
							</p>
						</div>
						<div className="flex flex-wrap gap-2">
							<StatusBadge tone="info">{vm.platform}</StatusBadge>
							<StatusBadge tone={powerTone(vm.power_state)}>
								{vm.power_state}
							</StatusBadge>
							<StatusBadge tone={readinessTone(row.readiness)}>
								{row.readiness}
							</StatusBadge>
							<StatusBadge tone={riskTone(row.risk)}>
								{row.risk} risk
							</StatusBadge>
						</div>
					</div>
				</div>

				<DetailSection title="Overview">
					<div className="grid gap-3 md:grid-cols-2">
						<StatCard label="Host" value={vm.host || "Unavailable"} />
						<StatCard label="Cluster" value={vm.cluster || "Unavailable"} />
						<StatCard
							label="Resource pool"
							value={vm.resource_pool || "Unavailable"}
						/>
						<StatCard
							label="Source reference"
							value={vm.source_ref || "Unavailable"}
						/>
						<StatCard label="CPU" value={`${vm.cpu_count} vCPU`} />
						<StatCard
							label="Memory"
							value={`${formatMemory(vm.memory_mb)} GB`}
						/>
						<StatCard
							label="Storage"
							value={`${formatStorage(row.storageTotalMB)} GB across ${vm.disks.length} disk(s)`}
						/>
						<StatCard
							label="Networks"
							value={`${vm.nics.length} NIC(s) • ${row.connectedNicCount} connected`}
						/>
					</div>
					{Object.keys(vm.tags ?? {}).length > 0 ? (
						<div className="mt-4 flex flex-wrap gap-2">
							{Object.entries(vm.tags ?? {}).map(([key, value]) => (
								<StatusBadge key={key} tone="neutral">
									{key}: {value}
								</StatusBadge>
							))}
						</div>
					) : null}
				</DetailSection>

				<DetailSection title="Dependencies">
					{assessmentErrors?.graph ? (
						<InlineNotice message={assessmentErrors.graph} tone="warning" />
					) : null}
					<div className="grid gap-3 md:grid-cols-3">
						<StatCard
							label="Networks"
							value={String(row.dependencies.networks.length)}
						/>
						<StatCard
							label="Datastores"
							value={String(row.dependencies.datastores.length)}
						/>
						<StatCard
							label="Backup jobs"
							value={String(row.dependencies.backups.length)}
						/>
					</div>
					<div className="mt-4 space-y-4">
						<RelationGroup
							title="Connected networks"
							emptyLabel={
								row.dependencies.graphResolved
									? "No network relationships were resolved."
									: "Dependency graph signals are unavailable for this workload."
							}
							labels={row.dependencies.networks.map((node) => node.label)}
						/>
						<RelationGroup
							title="Storage backends"
							emptyLabel={
								row.dependencies.graphResolved
									? "No datastore relationships were resolved."
									: "Dependency graph signals are unavailable for this workload."
							}
							labels={row.dependencies.datastores.map((node) => node.label)}
						/>
						<RelationGroup
							title="Backup protection"
							emptyLabel={
								row.dependencies.graphResolved
									? "No backup job relationships were resolved."
									: "Dependency graph signals are unavailable for this workload."
							}
							labels={row.dependencies.backups.map((node) => node.label)}
						/>
					</div>
				</DetailSection>

				<DetailSection title="Risks">
					{assessmentErrors?.policies ||
					assessmentErrors?.remediation ||
					row.assessmentIncomplete ? (
						<InlineNotice
							tone="warning"
							message={
								assessmentErrors?.policies || assessmentErrors?.remediation
									? [assessmentErrors.policies, assessmentErrors.remediation]
											.filter(Boolean)
											.join(" ")
									: `Risk posture is partial while ${row.missingSources.join(", ")} signals are unavailable.`
							}
						/>
					) : null}
					<div className="grid gap-3 md:grid-cols-2">
						<StatCard label="Risk score" value={String(row.riskScore)} />
						<StatCard
							label="Policy violations"
							value={String(row.policyViolations.length)}
						/>
						<StatCard
							label="Recommendations"
							value={String(row.recommendations.length)}
						/>
						<StatCard label="Snapshots" value={String(row.snapshotCount)} />
					</div>
					{row.riskReasons.length > 0 ? (
						<div className="mt-4 space-y-2">
							{row.riskReasons.map((reason) => (
								<InlineNotice key={reason} message={reason} tone="warning" />
							))}
						</div>
					) : (
						<div className="mt-4">
							<InlineNotice
								tone="success"
								message="No immediate operator risk signals are currently derived for this workload."
							/>
						</div>
					)}
					{row.policyViolations.length > 0 ? (
						<div className="mt-4 space-y-2">
							{row.policyViolations.slice(0, 4).map((violation) => (
								<div
									key={`${violation.policy.name}:${violation.rule.field}:${violation.vm.id || violation.vm.name}`}
									className="metric-surface text-sm text-slate-700"
								>
									<p className="font-semibold text-ink">
										{violation.policy.name}
									</p>
									<p className="mt-1 text-slate-500">
										{violation.rule.field} {violation.rule.operator}{" "}
										{String(violation.rule.value)}
									</p>
									{violation.remediation ? (
										<p className="mt-2 leading-6 text-slate-600">
											{violation.remediation}
										</p>
									) : null}
								</div>
							))}
						</div>
					) : null}
					{row.recommendations.length > 0 ? (
						<div className="mt-4 space-y-2">
							{row.recommendations.slice(0, 4).map((recommendation) => (
								<div
									key={`${recommendation.type}:${recommendation.summary}`}
									className="metric-surface text-sm text-slate-700"
								>
									<div className="flex items-center justify-between gap-3">
										<p className="font-semibold text-ink">
											{recommendation.summary}
										</p>
										<StatusBadge tone="neutral">
											{recommendation.type}
										</StatusBadge>
									</div>
									<p className="mt-2 leading-6 text-slate-500">
										{recommendation.action}
									</p>
								</div>
							))}
						</div>
					) : null}
				</DetailSection>

				<DetailSection title="Activity">
					<div className="grid gap-3 md:grid-cols-2">
						<StatCard label="Created" value={formatTimestamp(row.createdAt)} />
						<StatCard
							label="Last discovered"
							value={`${formatTimestamp(row.discoveredAt)} (${formatRelativeTime(row.discoveredAt)})`}
						/>
						<StatCard
							label="Last observed activity"
							value={`${formatTimestamp(row.lastActivityAt)} (${formatRelativeTime(row.lastActivityAt)})`}
						/>
						<StatCard
							label="Inventory baseline"
							value={
								latestSnapshot
									? `${latestSnapshot.id} • ${formatTimestamp(latestSnapshot.discovered_at)}`
									: "No saved snapshot baseline"
							}
						/>
					</div>
					{vm.snapshots && vm.snapshots.length > 0 ? (
						<div className="mt-4 space-y-2">
							{vm.snapshots.slice(0, 4).map((snapshot) => (
								<div
									key={snapshot.id || snapshot.name}
									className="metric-surface text-sm text-slate-700"
								>
									<div className="flex items-center justify-between gap-3">
										<p className="font-semibold text-ink">{snapshot.name}</p>
										<StatusBadge tone="warning">
											{formatTimestamp(snapshot.created_at)}
										</StatusBadge>
									</div>
									<p className="mt-2 leading-6 text-slate-500">
										{snapshot.description ||
											"No snapshot description provided."}
									</p>
								</div>
							))}
						</div>
					) : (
						<div className="mt-4">
							<InlineNotice
								tone="neutral"
								message="Viaduct does not currently expose a VM-scoped activity or audit feed in this screen, so activity is limited to inventory timestamps and snapshot metadata."
							/>
						</div>
					)}
				</DetailSection>
			</div>
		</SectionCard>
	);
}

function DetailSection({
	title,
	children,
}: {
	title: string;
	children: ReactNode;
}) {
	return (
		<section className="panel-subtle px-4 py-4">
			<h3 className="operator-kicker">{title}</h3>
			<div className="mt-3">{children}</div>
		</section>
	);
}

function RelationGroup({
	title,
	labels,
	emptyLabel,
}: {
	title: string;
	labels: string[];
	emptyLabel: string;
}) {
	return (
		<div>
			<p className="operator-kicker">{title}</p>
			<div className="mt-2 flex flex-wrap gap-2">
				{labels.length > 0 ? (
					labels.map((label) => (
						<StatusBadge key={`${title}:${label}`} tone="neutral">
							{label}
						</StatusBadge>
					))
				) : (
					<p className="text-sm text-slate-500">{emptyLabel}</p>
				)}
			</div>
		</div>
	);
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
