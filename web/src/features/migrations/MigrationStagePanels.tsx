import { Play, RefreshCcw } from "lucide-react";
import { getRouteHref } from "../../app/navigation";
import { MigrationProgress } from "../../components/MigrationProgress";
import { EmptyState } from "../../components/primitives/EmptyState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import {
	StatusBadge,
	type StatusTone,
} from "../../components/primitives/StatusBadge";
import type { Platform } from "../../types";
import { getVirtualMachineIdentity } from "../inventory/workloadIdentity";
import { MigrationPlanSummary } from "./MigrationPlanSummary";
import { PreflightResults } from "./PreflightResults";
import {
	describeMigrationPhase,
	getMigrationWorkflowStatus,
	getPreflightSummary,
	getWorkflowStatusPresentation,
} from "./migrationStatus";
import type { MigrationWorkspaceState } from "./useMigrationWorkspace";

interface StageProps {
	workspace: MigrationWorkspaceState;
}

export function MigrationScopeStage({ workspace }: StageProps) {
	return (
		<div className="mt-6 space-y-5">
			{workspace.importedDraft && (
				<SectionCard
					title="Imported scope"
					description="These workloads were handed off from inventory as a local dashboard draft."
				>
					<div className="flex flex-wrap gap-2">
						<StatusBadge tone="accent">Imported from inventory</StatusBadge>
						<StatusBadge tone="info">
							{workspace.importedDraft.sourcePlatform}
						</StatusBadge>
						<StatusBadge tone="neutral">
							{workspace.importedDraft.workloads.length} workload(s)
						</StatusBadge>
					</div>
					<div className="mt-4">
						<InlineNotice
							tone="info"
							message="This draft is local session state only. Save a migration plan to create a real backend migration record."
						/>
					</div>
					{workspace.draftNotice && (
						<div className="mt-4">
							<InlineNotice tone="warning" message={workspace.draftNotice} />
						</div>
					)}
					<div className="mt-4 flex flex-wrap gap-2">
						<a
							href={getRouteHref("/inventory")}
							className="operator-button-secondary"
						>
							Review inventory
						</a>
						<button
							type="button"
							onClick={workspace.clearImportedSelection}
							className="operator-button-secondary"
						>
							Clear imported selection
						</button>
					</div>
				</SectionCard>
			)}

			<div className="grid gap-5 xl:grid-cols-[0.95fr_1.05fr]">
				<SectionCard
					title="Source context"
					description="Record the source endpoint and load the latest platform inventory."
				>
					<div className="grid gap-4 md:grid-cols-2">
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Migration name</span>
							<input
								className="operator-input"
								value={workspace.migrationName}
								onChange={(event) =>
									workspace.setMigrationName(event.target.value)
								}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Source platform</span>
							<PlatformSelect
								value={workspace.sourcePlatform}
								onChange={workspace.changeSourcePlatform}
							/>
						</label>
						<label className="space-y-2 text-sm md:col-span-2">
							<span className="font-semibold text-ink">Source address</span>
							<input
								className="operator-input"
								value={workspace.sourceAddress}
								onChange={(event) =>
									workspace.setSourceAddress(event.target.value)
								}
								placeholder="vcsa.lab.local"
							/>
						</label>
					</div>
					<div className="mt-5 flex flex-wrap gap-3">
						<button
							type="button"
							className="operator-button"
							onClick={workspace.loadInventory}
							disabled={workspace.loading}
						>
							<RefreshCcw className="h-4 w-4" />
							{workspace.loading
								? "Loading inventory..."
								: "Load latest inventory"}
						</button>
						<StatusBadge tone={workspace.inventory ? "success" : "neutral"}>
							{workspace.inventory
								? `${workspace.inventory.vms.length} workload(s)`
								: "No inventory loaded"}
						</StatusBadge>
					</div>
					<div className="mt-5 space-y-3">
						<InlineNotice
							tone="neutral"
							message="Inventory loading does not auto-select every workload. Start from inventory for an explicit handoff, or pick the exact scope here before validation."
						/>
						<InlineNotice
							tone="neutral"
							message="The source address is still required because preflight and execution build real connector instances from the spec."
						/>
					</div>
				</SectionCard>

				<SectionCard
					title="Workload scope"
					description="Choose the workloads that should move into the plan."
				>
					{!workspace.inventory ? (
						<EmptyState
							title="No inventory loaded"
							message="Load platform inventory or start from the inventory route so workloads are available for planning."
						/>
					) : (
						<div className="space-y-4">
							<div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
								<input
									className="operator-input md:w-72"
									placeholder="Filter workloads"
									value={workspace.selectionSearch}
									onChange={(event) =>
										workspace.setSelectionSearch(event.target.value)
									}
								/>
								<div className="flex flex-wrap gap-2">
									<StatusBadge
										tone={
											workspace.selectedWorkloads.length > 0
												? "accent"
												: "neutral"
										}
									>
										{workspace.selectedWorkloads.length} selected
									</StatusBadge>
									<button
										type="button"
										className="operator-button-secondary"
										onClick={() =>
											workspace.setSelectedWorkloadKeys(
												workspace.filteredWorkloads.map((vm) =>
													getVirtualMachineIdentity(vm),
												),
											)
										}
										disabled={workspace.filteredWorkloads.length === 0}
									>
										Select visible
									</button>
									<button
										type="button"
										className="operator-button-secondary"
										onClick={() => workspace.setSelectedWorkloadKeys([])}
										disabled={workspace.selectedWorkloads.length === 0}
									>
										Clear
									</button>
								</div>
							</div>

							{workspace.filteredWorkloads.length === 0 ? (
								<EmptyState
									title="No workloads match this filter"
									message="Adjust the filter or reload inventory if you expected a workload to appear here."
								/>
							) : (
								<div className="grid max-h-[520px] gap-3 overflow-y-auto pr-1">
									{workspace.filteredWorkloads.map((vm) => {
										const key = getVirtualMachineIdentity(vm);
										const selected =
											workspace.selectedWorkloadKeys.includes(key);

										return (
											<label
												key={key}
												className={`flex items-start gap-3 rounded-xl border px-4 py-4 text-body-sm transition ${selected ? "border-sky-200 bg-sky-50/90 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]" : "border-slate-200/80 bg-white/85 hover:bg-slate-50/90"}`}
											>
												<input
													type="checkbox"
													checked={selected}
													onChange={() =>
														workspace.setSelectedWorkloadKeys((current) =>
															current.includes(key)
																? current.filter((item) => item !== key)
																: [...current, key],
														)
													}
												/>
												<div>
													<p className="font-semibold text-ink">{vm.name}</p>
													<p className="mt-1 text-slate-500">
														{vm.platform} • {vm.host || "Unknown host"}
														{vm.cluster ? ` • ${vm.cluster}` : ""}
													</p>
												</div>
											</label>
										);
									})}
								</div>
							)}
						</div>
					)}
				</SectionCard>
			</div>
		</div>
	);
}

export function MigrationPrepareStage({ workspace }: StageProps) {
	const unmappedNetworkCount = workspace.sourceNetworks.filter(
		(network) => !workspace.networkMap[network]?.trim(),
	).length;

	return (
		<div className="mt-6 space-y-5">
			<div className="grid gap-5 xl:grid-cols-[1fr_1fr]">
				<SectionCard
					title="Target environment"
					description="Set the target endpoint and default placement."
				>
					<div className="grid gap-4 md:grid-cols-2">
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Target platform</span>
							<PlatformSelect
								value={workspace.targetPlatform}
								onChange={workspace.setTargetPlatform}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Target address</span>
							<input
								className="operator-input"
								value={workspace.targetAddress}
								onChange={(event) =>
									workspace.setTargetAddress(event.target.value)
								}
								placeholder="pve.lab.local"
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Default host</span>
							<input
								className="operator-input"
								value={workspace.defaultHost}
								onChange={(event) =>
									workspace.setDefaultHost(event.target.value)
								}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Default storage</span>
							<input
								className="operator-input"
								value={workspace.defaultStorage}
								onChange={(event) =>
									workspace.setDefaultStorage(event.target.value)
								}
							/>
						</label>
					</div>
				</SectionCard>

				<SectionCard
					title="Execution controls"
					description="Set batching, windows, and approval details before validation."
				>
					<div className="grid gap-4 md:grid-cols-2">
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Parallel workers</span>
							<input
								className="operator-input"
								type="number"
								min={1}
								value={workspace.parallelism}
								onChange={(event) =>
									workspace.setParallelism(Number(event.target.value) || 1)
								}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Wave size</span>
							<input
								className="operator-input"
								type="number"
								min={1}
								value={workspace.waveSize}
								onChange={(event) =>
									workspace.setWaveSize(Number(event.target.value) || 1)
								}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Window opens</span>
							<input
								className="operator-input"
								type="datetime-local"
								value={workspace.scheduledStart}
								onChange={(event) =>
									workspace.setScheduledStart(event.target.value)
								}
							/>
						</label>
						<label className="space-y-2 text-sm">
							<span className="font-semibold text-ink">Window closes</span>
							<input
								className="w-full rounded-2xl border border-slate-200 px-4 py-3"
								type="datetime-local"
								value={workspace.scheduledEnd}
								onChange={(event) =>
									workspace.setScheduledEnd(event.target.value)
								}
							/>
						</label>
					</div>
					<div className="mt-5 space-y-3 rounded-xl border border-slate-200/80 bg-slate-50/88 px-4 py-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]">
						<label className="flex items-center gap-3 text-sm font-semibold text-ink">
							<input
								type="checkbox"
								checked={workspace.dependencyAware}
								onChange={(event) =>
									workspace.setDependencyAware(event.target.checked)
								}
							/>
							Keep wave planning dependency-aware
						</label>
						<label className="flex items-center gap-3 text-sm font-semibold text-ink">
							<input
								type="checkbox"
								checked={workspace.shutdownSource}
								onChange={(event) =>
									workspace.setShutdownSource(event.target.checked)
								}
							/>
							Shut down source workloads before export
						</label>
						<label className="flex items-center gap-3 text-sm font-semibold text-ink">
							<input
								type="checkbox"
								checked={workspace.verifyBoot}
								onChange={(event) =>
									workspace.setVerifyBoot(event.target.checked)
								}
							/>
							Verify target boot
						</label>
						<label className="flex items-center gap-3 text-sm font-semibold text-ink">
							<input
								type="checkbox"
								checked={workspace.approvalRequired}
								onChange={(event) => {
									workspace.setApprovalRequired(event.target.checked);
									if (!event.target.checked) {
										workspace.setApprovedBy("");
										workspace.setApprovalTicket("");
										workspace.setApprovalRecordedAt("");
									}
								}}
							/>
							Require approval before execution
						</label>
						{workspace.approvalRequired && (
							<div className="grid gap-4 md:grid-cols-2">
								<input
									className="operator-input"
									value={workspace.approvedBy}
									onChange={(event) => {
										workspace.setApprovedBy(event.target.value);
										if (
											event.target.value.trim() &&
											!workspace.approvalRecordedAt
										)
											workspace.setApprovalRecordedAt(new Date().toISOString());
										if (!event.target.value.trim())
											workspace.setApprovalRecordedAt("");
									}}
									placeholder="Approved by"
								/>
								<input
									className="operator-input"
									value={workspace.approvalTicket}
									onChange={(event) =>
										workspace.setApprovalTicket(event.target.value)
									}
									placeholder="Ticket"
								/>
							</div>
						)}
					</div>
				</SectionCard>
			</div>

			<SectionCard
				title="Network mappings"
				description="Map discovered source networks to target-side names."
			>
				{workspace.sourceNetworks.length === 0 ? (
					<InlineNotice
						message="No source networks are currently in scope."
						tone="neutral"
					/>
				) : (
					<div className="space-y-3">
						{workspace.sourceNetworks.map((network) => (
							<label
								key={network}
								className="grid gap-2 text-sm md:grid-cols-[1fr_1fr] md:items-center"
							>
								<span className="font-semibold text-ink">{network}</span>
								<input
									className="operator-input"
									value={workspace.networkMap[network] ?? ""}
									onChange={(event) =>
										workspace.setNetworkMap((current) => ({
											...current,
											[network]: event.target.value,
										}))
									}
									placeholder="Target network name"
								/>
							</label>
						))}
						{unmappedNetworkCount > 0 && (
							<InlineNotice
								tone="warning"
								message={`${unmappedNetworkCount} source network(s) do not have an explicit target mapping yet. Preflight will verify whether the target environment can still resolve them.`}
							/>
						)}
					</div>
				)}
			</SectionCard>
		</div>
	);
}

export function MigrationValidateStage({ workspace }: StageProps) {
	const hasUnmappedNetworks = workspace.sourceNetworks.some(
		(network) => !workspace.networkMap[network]?.trim(),
	);

	return (
		<div className="mt-6 space-y-5">
			<SectionCard
				title="Draft review"
				description="Review the current draft before validation or plan save."
			>
				<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
					<SimpleCell
						label="Source"
						value={`${workspace.sourcePlatform} • ${workspace.sourceAddress || "address required"}`}
					/>
					<SimpleCell
						label="Target"
						value={`${workspace.targetPlatform} • ${workspace.targetAddress || "address required"}`}
					/>
					<SimpleCell
						label="Workloads"
						value={`${workspace.selectedWorkloads.length} selected`}
					/>
					<SimpleCell
						label="Controls"
						value={`${workspace.parallelism} parallel • ${workspace.waveSize} per wave`}
					/>
				</div>
				<div className="mt-4 flex flex-wrap gap-2">
					<StatusBadge
						tone={
							workspace.approvalRequired
								? workspace.approvedBy
									? "success"
									: "warning"
								: "neutral"
						}
					>
						{workspace.approvalRequired
							? workspace.approvedBy
								? "Approval recorded"
								: "Approval required"
							: "No approval gate"}
					</StatusBadge>
					<StatusBadge
						tone={
							workspace.windowState.kind === "open"
								? "success"
								: workspace.windowState.kind === "unset"
									? "neutral"
									: "warning"
						}
					>
						{workspace.windowState.summary}
					</StatusBadge>
					<StatusBadge tone={hasUnmappedNetworks ? "warning" : "success"}>
						{hasUnmappedNetworks ? "Mappings need review" : "Mappings reviewed"}
					</StatusBadge>
				</div>
				{workspace.validationError ? (
					<div className="mt-4">
						<InlineNotice tone="warning" message={workspace.validationError} />
					</div>
				) : null}
				{workspace.preflightStale ? (
					<div className="mt-4">
						<InlineNotice
							tone="warning"
							message="Preflight results reflect an older draft. Run validation again before relying on readiness."
						/>
					</div>
				) : null}
				<div className="mt-4">
					<InlineNotice
						tone="neutral"
						message="Workloads are matched back into the saved plan by exact-name regex selectors derived from the current scope. That avoids accidental wildcard matching when VM names contain glob characters."
					/>
				</div>
			</SectionCard>

			<SectionCard
				title="Preflight validation"
				description="Run the current draft against the backend preflight checks."
				actions={
					<div className="flex flex-wrap gap-2">
						<button
							type="button"
							className="operator-button-secondary"
							onClick={workspace.handlePreflight}
							disabled={workspace.loading}
						>
							{workspace.loading ? "Running preflight..." : "Run preflight"}
						</button>
						<button
							type="button"
							className="operator-button"
							onClick={workspace.handleSavePlan}
							disabled={
								Boolean(workspace.validationError) ||
								workspace.loading ||
								Boolean(
									workspace.migrationState?.phase === "plan" &&
									!workspace.planStale,
								)
							}
						>
							{workspace.migrationState?.phase === "plan" &&
							!workspace.planStale
								? "Plan already saved"
								: workspace.loading
									? "Saving plan..."
									: "Save migration plan"}
						</button>
					</div>
				}
			>
				<div className="panel-muted px-4 py-4">
					<div className="flex flex-wrap gap-2">
						<StatusBadge tone={workspace.workflowPresentation.tone}>
							{workspace.workflowPresentation.label}
						</StatusBadge>
						{workspace.preflight && (
							<StatusBadge
								tone={
									workspace.preflight.fail_count > 0
										? "danger"
										: workspace.preflight.warn_count > 0
											? "warning"
											: "success"
								}
							>
								{workspace.preflight.fail_count} blocker(s) •{" "}
								{workspace.preflight.warn_count} warning(s)
							</StatusBadge>
						)}
					</div>
					<p className="mt-3 text-sm text-slate-600">
						{getPreflightSummary(workspace.preflight, workspace.preflightStale)}
					</p>
				</div>
				{workspace.preflight ? (
					<div className="mt-5">
						<PreflightResults checks={workspace.preflight.checks} />
					</div>
				) : (
					<InlineNotice
						message="No preflight results yet. Validation is what turns this draft into a clear operational readiness statement."
						tone="neutral"
						className="mt-5"
					/>
				)}
			</SectionCard>

			{(workspace.migrationState?.plan ?? workspace.preflight?.plan) && (
				<MigrationPlanSummary
					title="Execution runbook"
					description="The wave plan Viaduct derived from the current selectors and controls."
					plan={(workspace.migrationState?.plan ?? workspace.preflight?.plan)!}
				/>
			)}
		</div>
	);
}

export function MigrationExecuteStage({ workspace }: StageProps) {
	if (!workspace.migrationState) {
		return (
			<div className="mt-6 space-y-5">
				<SectionCard
					title="Execute"
					description="A saved plan is required before execution routes can be used."
				>
					<EmptyState
						title="No saved plan"
						message="Save a migration plan from the validation stage first."
					/>
				</SectionCard>
			</div>
		);
	}

	const status = getWorkflowStatusPresentation(
		getMigrationWorkflowStatus(workspace.migrationState),
	);

	return (
		<div className="mt-6 space-y-5">
			<SectionCard
				title="Saved plan state"
				description="Track the persisted migration object and any remaining execution blockers."
				actions={
					<div className="flex flex-wrap gap-2">
						<button
							type="button"
							className="operator-button-secondary"
							onClick={() => void workspace.refreshSavedState()}
							disabled={workspace.loading || workspace.refreshingState}
						>
							<RefreshCcw className="h-4 w-4" />
							{workspace.refreshingState
								? "Refreshing state..."
								: "Refresh saved state"}
						</button>
						<button
							type="button"
							className="operator-button-secondary"
							onClick={() => void workspace.handlePreflight()}
							disabled={workspace.loading}
						>
							<RefreshCcw className="h-4 w-4" />
							Rerun preflight
						</button>
						{workspace.migrationState.phase === "plan" && (
							<button
								type="button"
								className="operator-button"
								onClick={workspace.handleExecute}
								disabled={
									workspace.loading || workspace.executionBlockers.length > 0
								}
							>
								<Play className="h-4 w-4" />
								Execute
							</button>
						)}
						{workspace.migrationState.phase === "failed" && (
							<>
								<button
									type="button"
									className="operator-button"
									onClick={workspace.handleResume}
									disabled={
										workspace.loading || workspace.executionBlockers.length > 0
									}
								>
									Resume
								</button>
								<button
									type="button"
									className="operator-button-danger"
									onClick={workspace.handleRollback}
									disabled={workspace.loading}
								>
									Rollback
								</button>
							</>
						)}
					</div>
				}
			>
				<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
					<SimpleCell
						label="Migration ID"
						value={workspace.migrationState.id}
					/>
					<SimpleCell label="Status" value={status.label} />
					<SimpleCell
						label="Phase"
						value={describeMigrationPhase(workspace.migrationState.phase)}
					/>
					<SimpleCell
						label="Updated"
						value={new Date(
							workspace.migrationState.updated_at,
						).toLocaleString()}
					/>
				</div>
				<div className="mt-4 flex flex-wrap gap-2">
					<StatusBadge tone={status.tone}>{status.label}</StatusBadge>
					<StatusBadge tone={workspace.planStale ? "danger" : "success"}>
						{workspace.planStale ? "Plan stale" : "Plan matches draft"}
					</StatusBadge>
					<StatusBadge
						tone={
							workspace.windowState.kind === "open"
								? "success"
								: workspace.windowState.kind === "unset"
									? "neutral"
									: "warning"
						}
					>
						{workspace.windowState.summary}
					</StatusBadge>
					{workspace.isPolling && (
						<StatusBadge tone="accent">Refreshing live state</StatusBadge>
					)}
				</div>
				{workspace.executionBlockers.length > 0 && (
					<div className="mt-4 space-y-2">
						{workspace.executionBlockers.map((item) => (
							<InlineNotice key={item} tone="warning" message={item} />
						))}
					</div>
				)}
				{workspace.executionAdvisories.length > 0 && (
					<div className="mt-4 space-y-2">
						{workspace.executionAdvisories.map((item) => (
							<InlineNotice key={item} tone="neutral" message={item} />
						))}
					</div>
				)}
				{workspace.rollbackResult && (
					<div className="mt-4 space-y-2">
						<InlineNotice
							tone="neutral"
							message={`Rollback removed ${workspace.rollbackResult.target_vms_removed} target workload(s), cleaned up ${workspace.rollbackResult.files_cleaned_up} file artifact(s), and restored ${workspace.rollbackResult.source_vms_restored} source workload(s).`}
						/>
						{(workspace.rollbackResult.errors ?? []).map((item) => (
							<InlineNotice key={item} tone="warning" message={item} />
						))}
					</div>
				)}
			</SectionCard>
			<MigrationProgress
				state={workspace.migrationState}
				onRollback={workspace.handleRollback}
			/>
		</div>
	);
}

function PlatformSelect({
	value,
	onChange,
}: {
	value: Platform;
	onChange: (value: Platform) => void;
}) {
	return (
		<select
			className="operator-select"
			value={value}
			onChange={(event) => onChange(event.target.value as Platform)}
		>
			<option value="vmware">VMware</option>
			<option value="proxmox">Proxmox</option>
			<option value="hyperv">Hyper-V</option>
			<option value="kvm">KVM</option>
			<option value="nutanix">Nutanix</option>
		</select>
	);
}

export function MigrationMetric({
	label,
	value,
	tone,
	detail,
}: {
	label: string;
	value: string;
	tone: StatusTone;
	detail: string;
}) {
	return (
		<StatCard
			label={label}
			value={value}
			detail={detail}
			badge={{ label: value, tone }}
		/>
	);
}

function SimpleCell({ label, value }: { label: string; value: string }) {
	return <StatCard label={label} value={value} />;
}
