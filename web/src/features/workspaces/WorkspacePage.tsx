import {
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
	type ReactNode,
} from "react";
import {
	createWorkspace,
	createWorkspaceJob,
	deleteWorkspace,
	describeError,
	exportWorkspaceReport,
	getSnapshot,
	getWorkspace,
	listWorkspaceJobs,
	listWorkspaces,
	updateWorkspace,
	type ErrorDisplay,
} from "../../api";
import { DependencyGraph } from "../../components/DependencyGraph";
import { InventoryTable } from "../../components/InventoryTable";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import {
	StatusBadge,
	type StatusTone,
} from "../../components/primitives/StatusBadge";
import type {
	DiscoveryResult,
	PilotWorkspace,
	Platform,
	SimulationRequest,
	WorkspaceJob,
	WorkspaceJobStatus,
	WorkspaceJobType,
	WorkspaceNote,
	WorkspaceSnapshot,
} from "../../types";
import {
	buildInventoryAssessmentRows,
	type InventoryAssessmentRow,
} from "../inventory/inventoryModel";
import { revealWorkloadDetailPanel } from "../inventory/revealWorkloadDetailPanel";
import { WorkloadDetailPanel } from "../inventory/WorkloadDetailPanel";
import { useInventoryWorkspace } from "../inventory/useInventoryWorkspace";

interface WorkspacePageState {
	workspaces: PilotWorkspace[];
	selectedWorkspace: PilotWorkspace | null;
	jobs: WorkspaceJob[];
	inventory: DiscoveryResult | null;
	loading: boolean;
	refreshing: boolean;
	error: ErrorDisplay | null;
	actionError: ErrorDisplay | null;
}

interface WorkflowStep {
	id: string;
	label: string;
	summary: string;
	evidence: string;
	status: "complete" | "current" | "upcoming";
}

interface CreateFormState {
	name: string;
	description: string;
	sourceName: string;
	sourcePlatform: Platform;
	sourceAddress: string;
	sourceCredentialRef: string;
	targetPlatform: Platform;
	targetAddress: string;
	defaultHost: string;
	defaultStorage: string;
	defaultNetwork: string;
}

const PLATFORM_OPTIONS: Platform[] = [
	"vmware",
	"proxmox",
	"hyperv",
	"kvm",
	"nutanix",
];

const defaultCreateForm: CreateFormState = {
	name: "Examples Lab Assessment",
	description: "Assessment workspace for the local lab flow",
	sourceName: "Lab KVM",
	sourcePlatform: "kvm",
	sourceAddress: "examples/lab/kvm",
	sourceCredentialRef: "lab-kvm",
	targetPlatform: "proxmox",
	targetAddress: "https://pilot-proxmox.local:8006/api2/json",
	defaultHost: "pve-node-01",
	defaultStorage: "local-lvm",
	defaultNetwork: "vmbr0",
};

export function WorkspacePage() {
	const [state, setState] = useState<WorkspacePageState>({
		workspaces: [],
		selectedWorkspace: null,
		jobs: [],
		inventory: null,
		loading: true,
		refreshing: false,
		error: null,
		actionError: null,
	});
	const [selectedWorkspaceID, setSelectedWorkspaceID] = useState<string | null>(
		null,
	);
	const [showCreateForm, setShowCreateForm] = useState(false);
	const [creating, setCreating] = useState(false);
	const [createForm, setCreateForm] =
		useState<CreateFormState>(defaultCreateForm);
	const [noteDraft, setNoteDraft] = useState("");
	const [settingsDraft, setSettingsDraft] = useState<
		Pick<PilotWorkspace, "target_assumptions" | "plan_settings">
	>({
		target_assumptions: {},
		plan_settings: {},
	});
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const selectedWorkspaceIDRef = useRef<string | null>(null);
	const detailPanelRef = useRef<HTMLDivElement | null>(null);

	const refreshWorkspaces = useCallback(
		async (preferredWorkspaceID?: string) => {
			setState((current) => ({ ...current, refreshing: !current.loading }));
			const workspaceResult = await Promise.allSettled([listWorkspaces()]);
			const result = workspaceResult[0];
			if (result.status === "rejected") {
				setState((current) => ({
					...current,
					workspaces: [],
					selectedWorkspace: null,
					jobs: [],
					inventory: null,
					loading: false,
					refreshing: false,
					error: describeError(result.reason, {
						scope: "pilot workspaces",
						fallback: "Unable to load pilot workspaces.",
					}),
				}));
				return;
			}

			const workspaces = result.value;
			const nextWorkspaceID =
				preferredWorkspaceID ??
				selectedWorkspaceIDRef.current ??
				workspaces[0]?.id ??
				null;
			setShowCreateForm((current) =>
				workspaces.length === 0 ? true : current,
			);
			setSelectedWorkspaceID(nextWorkspaceID);
			setState((current) => ({
				...current,
				workspaces,
				loading: false,
				refreshing: false,
				error: null,
			}));
		},
		[],
	);

	const refreshWorkspaceDetail = useCallback(async (workspaceID: string) => {
		setState((current) => ({ ...current, refreshing: true }));
		const [workspaceResult, jobsResult] = await Promise.allSettled([
			getWorkspace(workspaceID),
			listWorkspaceJobs(workspaceID),
		]);
		if (workspaceResult.status === "rejected") {
			setState((current) => ({
				...current,
				selectedWorkspace: null,
				jobs: [],
				inventory: null,
				refreshing: false,
				error: describeError(workspaceResult.reason, {
					scope: "workspace state",
					fallback: "Unable to load workspace state.",
				}),
			}));
			return;
		}

		const workspace = workspaceResult.value;
		let inventory: DiscoveryResult | null = null;
		try {
			inventory = await loadWorkspaceInventory(workspace.snapshots ?? []);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: "workspace snapshots",
					fallback: "Unable to load workspace snapshots.",
				}),
			}));
		}
		setSettingsDraft({
			target_assumptions: workspace.target_assumptions ?? {},
			plan_settings: workspace.plan_settings ?? {},
		});
		setState((current) => ({
			...current,
			selectedWorkspace: workspace,
			jobs: jobsResult.status === "fulfilled" ? jobsResult.value : [],
			inventory,
			refreshing: false,
			error: null,
			actionError:
				jobsResult.status === "rejected"
					? describeError(jobsResult.reason, {
							scope: "workspace jobs",
							fallback: "Unable to load workspace jobs.",
						})
					: null,
		}));
	}, []);

	useEffect(() => {
		selectedWorkspaceIDRef.current = selectedWorkspaceID;
	}, [selectedWorkspaceID]);

	useEffect(() => {
		void refreshWorkspaces();
	}, [refreshWorkspaces]);

	useEffect(() => {
		if (!selectedWorkspaceID) {
			return;
		}
		void refreshWorkspaceDetail(selectedWorkspaceID);
	}, [refreshWorkspaceDetail, selectedWorkspaceID]);

	useEffect(() => {
		if (
			!state.selectedWorkspace ||
			!state.jobs.some(
				(job) => job.status === "queued" || job.status === "running",
			)
		) {
			return;
		}
		const handle = window.setTimeout(() => {
			void refreshWorkspaceDetail(state.selectedWorkspace!.id);
		}, 1500);
		return () => window.clearTimeout(handle);
	}, [refreshWorkspaceDetail, state.jobs, state.selectedWorkspace]);

	const graph = state.selectedWorkspace?.graph?.raw_json ?? null;
	const simulation = state.selectedWorkspace?.simulation?.raw_json;
	const rows = useMemo(
		() =>
			buildInventoryAssessmentRows(
				state.inventory?.vms ?? [],
				graph,
				simulation?.policy_report ?? null,
				simulation?.recommendation_report ?? null,
				{
					graph: Boolean(graph),
					policies: Boolean(simulation?.policy_report),
					remediation: Boolean(simulation?.recommendation_report),
				},
				state.inventory?.discovered_at,
			),
		[graph, simulation, state.inventory],
	);
	const savedSelection = useMemo(
		() => state.selectedWorkspace?.selected_workload_ids ?? [],
		[state.selectedWorkspace?.selected_workload_ids],
	);
	const inventoryWorkspace = useInventoryWorkspace(rows, savedSelection);
	const latestSnapshot = state.selectedWorkspace?.snapshots?.[0]
		? {
				id: state.selectedWorkspace.snapshots[0].snapshot_id,
				source: state.selectedWorkspace.snapshots[0].source,
				platform: state.selectedWorkspace.snapshots[0].platform,
				vm_count: state.selectedWorkspace.snapshots[0].vm_count,
				discovered_at: state.selectedWorkspace.snapshots[0].discovered_at,
			}
		: null;
	const jobHistory = useMemo(
		() =>
			[...state.jobs].sort(
				(left, right) =>
					new Date(right.requested_at).getTime() -
					new Date(left.requested_at).getTime(),
			),
		[state.jobs],
	);
	const activeJob =
		jobHistory.find(
			(job) => job.status === "queued" || job.status === "running",
		) ?? null;
	const reportHistory = useMemo(
		() =>
			[...(state.selectedWorkspace?.reports ?? [])].sort(
				(left, right) =>
					new Date(right.exported_at).getTime() -
					new Date(left.exported_at).getTime(),
			),
		[state.selectedWorkspace?.reports],
	);
	const noteHistory = useMemo(
		() =>
			[...(state.selectedWorkspace?.notes ?? [])].sort(
				(left, right) =>
					new Date(right.created_at).getTime() -
					new Date(left.created_at).getTime(),
			),
		[state.selectedWorkspace?.notes],
	);
	const latestSource = state.selectedWorkspace?.source_connections?.[0] ?? null;
	const workflowSteps = useMemo(
		() =>
			buildWorkflowSteps(
				state.selectedWorkspace,
				rows.length,
				inventoryWorkspace.selectedIds.length,
			),
		[
			inventoryWorkspace.selectedIds.length,
			rows.length,
			state.selectedWorkspace,
		],
	);
	const nextAction = useMemo(
		() =>
			recommendedWorkspaceAction(
				state.selectedWorkspace,
				rows.length,
				inventoryWorkspace.selectedIds.length,
			),
		[
			inventoryWorkspace.selectedIds.length,
			rows.length,
			state.selectedWorkspace,
		],
	);
	const createValidation = useMemo(
		() => validateCreateForm(createForm),
		[createForm],
	);
	const createErrors = Object.values(createValidation).filter(
		(value): value is string => Boolean(value),
	);
	const canCreateWorkspace = createErrors.length === 0;

	async function handleCreateWorkspace() {
		if (!canCreateWorkspace) {
			setState((current) => ({
				...current,
				actionError: {
					message:
						"Complete the required workspace intake fields before creating the workspace.",
					technicalDetails: createErrors,
				},
			}));
			return;
		}

		setCreating(true);
		try {
			const created = await createWorkspace({
				name: createForm.name,
				description: createForm.description,
				status: "draft",
				source_connections: [
					{
						id: "source-lab",
						name: createForm.sourceName,
						platform: createForm.sourcePlatform,
						address: createForm.sourceAddress,
						credential_ref: createForm.sourceCredentialRef,
					},
				],
				target_assumptions: {
					platform: createForm.targetPlatform,
					address: createForm.targetAddress,
					default_host: createForm.defaultHost,
					default_storage: createForm.defaultStorage,
					default_network: createForm.defaultNetwork,
				},
				plan_settings: {
					name: `${createForm.name.toLowerCase().replace(/\s+/g, "-")}-plan`,
					parallel: 2,
					verify_boot: true,
					approval_required: true,
					wave_size: 2,
					dependency_aware: true,
				},
			});
			setState((current) => ({ ...current, actionError: null }));
			setShowCreateForm(false);
			setCreateForm(defaultCreateForm);
			await refreshWorkspaces(created.id);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: "workspace creation",
					fallback: "Unable to create the pilot workspace.",
				}),
			}));
		} finally {
			setCreating(false);
		}
	}

	async function handleDeleteWorkspace() {
		if (!state.selectedWorkspace) {
			return;
		}
		const confirmed = window.confirm(
			`Delete workspace "${state.selectedWorkspace.name}"? Saved snapshots and migration records remain, but the workspace and its job history will be removed.`,
		);
		if (!confirmed) {
			return;
		}
		setActionLoading("delete");
		try {
			await deleteWorkspace(state.selectedWorkspace.id);
			setState((current) => ({
				...current,
				actionError: null,
				selectedWorkspace: null,
				jobs: [],
				inventory: null,
			}));
			setSelectedWorkspaceID(null);
			await refreshWorkspaces();
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: "workspace deletion",
					fallback: "Unable to delete the pilot workspace.",
				}),
			}));
		} finally {
			setActionLoading(null);
		}
	}

	async function handleWorkspaceUpdate(
		payload: Partial<PilotWorkspace>,
		loadingKey: string,
	) {
		if (!state.selectedWorkspace) {
			return;
		}
		setActionLoading(loadingKey);
		try {
			await updateWorkspace(state.selectedWorkspace.id, payload);
			if (loadingKey === "notes") {
				setNoteDraft("");
			}
			setState((current) => ({ ...current, actionError: null }));
			await refreshWorkspaceDetail(state.selectedWorkspace.id);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: "workspace update",
					fallback: "Unable to save workspace updates.",
				}),
			}));
		} finally {
			setActionLoading(null);
		}
	}

	async function handleJob(type: WorkspaceJobType) {
		if (!state.selectedWorkspace) {
			return;
		}
		setActionLoading(type);
		try {
			await createWorkspaceJob(state.selectedWorkspace.id, {
				type,
				selected_workload_ids:
					type === "simulation" || type === "plan"
						? inventoryWorkspace.selectedIds
						: undefined,
			});
			setState((current) => ({ ...current, actionError: null }));
			await refreshWorkspaceDetail(state.selectedWorkspace.id);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: `${type} job`,
					fallback: `Unable to start the ${type} job.`,
				}),
			}));
		} finally {
			setActionLoading(null);
		}
	}

	async function handleRetryJob(job: WorkspaceJob) {
		if (!state.selectedWorkspace) {
			return;
		}
		setActionLoading(`retry:${job.id}`);
		try {
			await createWorkspaceJob(
				state.selectedWorkspace.id,
				workspaceJobRetryPayload(job, inventoryWorkspace.selectedIds),
			);
			setState((current) => ({ ...current, actionError: null }));
			await refreshWorkspaceDetail(state.selectedWorkspace.id);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: `${job.type} retry`,
					fallback: "Unable to retry the workspace job.",
				}),
			}));
		} finally {
			setActionLoading(null);
		}
	}

	async function handleExportReport() {
		if (!state.selectedWorkspace) {
			return;
		}
		setActionLoading("report");
		try {
			const result = await exportWorkspaceReport(
				state.selectedWorkspace.id,
				"markdown",
			);
			const url = window.URL.createObjectURL(result.blob);
			const link = document.createElement("a");
			link.href = url;
			link.download = result.filename;
			link.click();
			window.URL.revokeObjectURL(url);
			await refreshWorkspaceDetail(state.selectedWorkspace.id);
		} catch (reason) {
			setState((current) => ({
				...current,
				actionError: describeError(reason, {
					scope: "report export",
					fallback: "Unable to export the pilot report.",
				}),
			}));
		} finally {
			setActionLoading(null);
		}
	}

	function handleFocusWorkload(id: string) {
		inventoryWorkspace.setActiveWorkloadId(id);
		revealWorkloadDetailPanel(detailPanelRef.current);
	}

	function handleSaveSelection(row: InventoryAssessmentRow) {
		inventoryWorkspace.replaceSelection([row.id]);
		void handleWorkspaceUpdate(
			{ selected_workload_ids: [row.id] },
			"selection",
		);
	}

	if (state.loading) {
		return (
			<LoadingState
				title="Loading pilot workspaces"
				message="Retrieving saved pilot assessments and the latest workspace-first operator flow."
			/>
		);
	}

	if (state.error && state.workspaces.length === 0) {
		return (
			<ErrorState
				title="Pilot workspaces unavailable"
				message={state.error.message}
				technicalDetails={state.error.technicalDetails}
				actions={[
					<button
						key="retry"
						type="button"
						onClick={() => void refreshWorkspaces()}
						className="operator-button-danger"
					>
						Retry
					</button>,
				]}
			/>
		);
	}

	if (state.workspaces.length === 0) {
		return (
			<div className="space-y-6">
				<PageHeader
					eyebrow="Pilot Workspaces"
					title="Start the workspace-first operator flow"
					description="Create a persisted assessment workspace before discovery so source details, job history, readiness results, saved plans, notes, and exported reports stay attached to the same operator record."
					badges={[
						{ label: "No workspaces yet", tone: "warning" },
						{ label: "Lab defaults prefilled", tone: "info" },
					]}
				/>

				{state.actionError ? (
					<ErrorState
						title="Workspace intake needs attention"
						message={state.actionError.message}
						technicalDetails={state.actionError.technicalDetails}
					/>
				) : null}

				<section className="grid gap-5 xl:grid-cols-[1.05fr_0.95fr]">
					<EmptyState
						title="Create the first workspace"
						message="The intake form is prefilled for the local lab so a fresh clone can reach discovery, inspection, simulation, and report export without a live estate."
					/>
					<SectionCard
						eyebrow="Operator flow"
						title="What happens after intake"
						description="The dashboard keeps each step inside one persisted workspace instead of spreading it across disconnected pages and screenshots."
					>
						<div className="space-y-3">
							{buildWorkflowSteps(null, 0, 0).map((step, index) => (
								<WorkflowStepCard key={step.id} step={step} index={index + 1} />
							))}
						</div>
					</SectionCard>
				</section>

				<WorkspaceCreateForm
					createForm={createForm}
					validation={createValidation}
					creating={creating}
					onChange={setCreateForm}
					onSubmit={() => void handleCreateWorkspace()}
				/>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Pilot Workspaces"
				title={state.selectedWorkspace?.name ?? "Pilot Workspaces"}
				description="Guide operators through intake, discovery, inspection, simulation, plan review, and report export from one persisted workspace record."
				badges={[
					{ label: `${state.workspaces.length} workspace(s)`, tone: "neutral" },
					{
						label: state.selectedWorkspace?.status ?? "draft",
						tone: workspaceStatusTone(state.selectedWorkspace?.status),
					},
					{ label: nextAction, tone: "info" },
				]}
				actions={
					<>
						<select
							value={selectedWorkspaceID ?? ""}
							onChange={(event) => setSelectedWorkspaceID(event.target.value)}
							aria-label="Select workspace"
							className="operator-select min-w-[16rem]"
						>
							{state.workspaces.map((workspace) => (
								<option key={workspace.id} value={workspace.id}>
									{workspace.name}
								</option>
							))}
						</select>
						<button
							type="button"
							onClick={() => setShowCreateForm((current) => !current)}
							className="operator-button-secondary"
						>
							{showCreateForm ? "Hide intake" : "New workspace"}
						</button>
						<button
							type="button"
							onClick={() =>
								void (selectedWorkspaceID
									? refreshWorkspaceDetail(selectedWorkspaceID)
									: Promise.resolve())
							}
							className="operator-button-secondary"
							disabled={!selectedWorkspaceID}
						>
							Refresh
						</button>
						<button
							type="button"
							onClick={() => void handleDeleteWorkspace()}
							className="operator-button-danger"
							disabled={!state.selectedWorkspace || actionLoading === "delete"}
						>
							{actionLoading === "delete" ? "Deleting..." : "Delete workspace"}
						</button>
					</>
				}
			/>

			{state.actionError ? (
				<ErrorState
					title="Workspace action needs attention"
					message={state.actionError.message}
					technicalDetails={state.actionError.technicalDetails}
				/>
			) : null}

			{showCreateForm ? (
				<WorkspaceCreateForm
					createForm={createForm}
					validation={createValidation}
					creating={creating}
					onChange={setCreateForm}
					onSubmit={() => void handleCreateWorkspace()}
				/>
			) : null}

			<section className="grid gap-5 xl:grid-cols-[1.05fr_0.95fr]">
				<SectionCard
					eyebrow="Workflow progress"
					title="Workspace progression"
					description="Operators can see what is complete, what is currently recommended, and what still needs evidence before handoff."
				>
					<div className="space-y-3">
						{workflowSteps.map((step, index) => (
							<WorkflowStepCard key={step.id} step={step} index={index + 1} />
						))}
					</div>
				</SectionCard>

				<SectionCard
					eyebrow="Current state"
					title="Workspace summary"
					description="This panel keeps the operator-facing headline state visible without needing to scan every section."
				>
					<div className="grid gap-3 md:grid-cols-2">
						<Metric
							label="Workspace status"
							value={state.selectedWorkspace?.status ?? "draft"}
							tone={workspaceStatusTone(state.selectedWorkspace?.status)}
						/>
						<Metric
							label="Selected workloads"
							value={String(inventoryWorkspace.selectedIds.length)}
							tone={
								inventoryWorkspace.selectedIds.length > 0 ? "accent" : "neutral"
							}
						/>
						<Metric
							label="Snapshots"
							value={String(state.selectedWorkspace?.snapshots?.length ?? 0)}
							tone={
								(state.selectedWorkspace?.snapshots?.length ?? 0) > 0
									? "info"
									: "neutral"
							}
						/>
						<Metric
							label="Reports"
							value={String(reportHistory.length)}
							tone={reportHistory.length > 0 ? "success" : "neutral"}
						/>
						<Metric
							label="Readiness"
							value={state.selectedWorkspace?.readiness?.status ?? "pending"}
							tone={readinessTone(state.selectedWorkspace?.readiness?.status)}
						/>
						<Metric
							label="Notes"
							value={String(noteHistory.length)}
							tone={noteHistory.length > 0 ? "info" : "neutral"}
						/>
					</div>

					{activeJob ? (
						<div className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-900">
							<p className="font-semibold">Current background work</p>
							<p className="mt-1">
								{activeJob.type} is {activeJob.status}.{" "}
								{activeJob.message ??
									"The operator API is still processing the workspace job."}
							</p>
							<p className="mt-2 text-xs text-amber-800">
								Correlation ID {activeJob.correlation_id ?? "n/a"} · Requested{" "}
								{formatTimestamp(activeJob.requested_at)}
							</p>
						</div>
					) : null}

					{state.selectedWorkspace?.saved_plan ? (
						<div className="mt-4 metric-surface">
							<p className="operator-kicker">Saved plan</p>
							<p className="mt-2 text-sm font-semibold text-ink">
								{state.selectedWorkspace.saved_plan.spec_name}
							</p>
							<p className="mt-1 text-sm text-slate-600">
								{state.selectedWorkspace.saved_plan.workload_count} workload(s)
								· Generated{" "}
								{formatTimestamp(
									state.selectedWorkspace.saved_plan.generated_at,
								)}
							</p>
						</div>
					) : null}
				</SectionCard>
			</section>

			<section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
				<SectionCard
					eyebrow="Assessment intake"
					title="Source, target, and plan defaults"
					description="Keep target assumptions and plan settings explicit so discovery, simulation, and exported evidence share the same operator-owned context."
				>
					<div className="grid gap-4 lg:grid-cols-2">
						<WorkspaceSelectField
							label="Target platform"
							value={
								settingsDraft.target_assumptions?.platform ??
								state.selectedWorkspace?.target_assumptions?.platform ??
								"proxmox"
							}
							options={PLATFORM_OPTIONS}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										platform: value as Platform,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Target address"
							value={settingsDraft.target_assumptions?.address ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										address: value,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Default host"
							value={settingsDraft.target_assumptions?.default_host ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										default_host: value,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Default storage"
							value={settingsDraft.target_assumptions?.default_storage ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										default_storage: value,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Default network"
							value={settingsDraft.target_assumptions?.default_network ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										default_network: value,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Plan name"
							value={settingsDraft.plan_settings?.name ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: { ...current.plan_settings, name: value },
								}))
							}
						/>
						<WorkspaceNumberField
							label="Parallel workers"
							value={settingsDraft.plan_settings?.parallel}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: { ...current.plan_settings, parallel: value },
								}))
							}
						/>
						<WorkspaceNumberField
							label="Wave size"
							value={settingsDraft.plan_settings?.wave_size}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: { ...current.plan_settings, wave_size: value },
								}))
							}
						/>
						<WorkspaceField
							label="Approval ticket"
							value={settingsDraft.plan_settings?.approval_ticket ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										approval_ticket: value,
									},
								}))
							}
						/>
						<WorkspaceField
							label="Window start"
							value={settingsDraft.plan_settings?.window_start ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										window_start: value,
									},
								}))
							}
							hint="Optional RFC3339 timestamp for planned execution start."
						/>
						<WorkspaceField
							label="Window end"
							value={settingsDraft.plan_settings?.window_end ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										window_end: value,
									},
								}))
							}
							hint="Optional RFC3339 timestamp for planned execution end."
						/>
						<WorkspaceField
							label="Target notes"
							value={settingsDraft.target_assumptions?.notes ?? ""}
							onChange={(value) =>
								setSettingsDraft((current) => ({
									...current,
									target_assumptions: {
										...current.target_assumptions,
										notes: value,
									},
								}))
							}
							hint="Capture assumptions, pilot caveats, or operator reminders."
						/>
					</div>

					<div className="mt-4 grid gap-3 md:grid-cols-3">
						<WorkspaceCheckbox
							label="Verify boot"
							checked={settingsDraft.plan_settings?.verify_boot ?? false}
							onChange={(checked) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										verify_boot: checked,
									},
								}))
							}
						/>
						<WorkspaceCheckbox
							label="Approval required"
							checked={settingsDraft.plan_settings?.approval_required ?? false}
							onChange={(checked) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										approval_required: checked,
									},
								}))
							}
						/>
						<WorkspaceCheckbox
							label="Dependency aware"
							checked={settingsDraft.plan_settings?.dependency_aware ?? false}
							onChange={(checked) =>
								setSettingsDraft((current) => ({
									...current,
									plan_settings: {
										...current.plan_settings,
										dependency_aware: checked,
									},
								}))
							}
						/>
					</div>

					<div className="mt-4 flex flex-wrap gap-2">
						<button
							type="button"
							onClick={() =>
								void handleWorkspaceUpdate(settingsDraft, "settings")
							}
							className="operator-button"
						>
							{actionLoading === "settings"
								? "Saving..."
								: "Save workspace settings"}
						</button>
					</div>
				</SectionCard>

				<div className="space-y-5">
					<SectionCard
						eyebrow="Recommended actions"
						title="Run the next operator step"
						description="The backend remains the source of truth. These actions start persisted jobs or exports against the selected workspace."
					>
						<div className="metric-surface">
							<p className="operator-kicker">Recommended next action</p>
							<p className="mt-2 text-sm font-semibold text-ink">
								{nextAction}
							</p>
						</div>
						<div className="mt-4 flex flex-wrap gap-2">
							<button
								type="button"
								onClick={() => void handleJob("discovery")}
								className="operator-button-secondary"
							>
								{actionLoading === "discovery"
									? "Running discovery..."
									: "Run discovery"}
							</button>
							<button
								type="button"
								onClick={() => void handleJob("graph")}
								className="operator-button-secondary"
							>
								{actionLoading === "graph"
									? "Building graph..."
									: "Build graph"}
							</button>
							<button
								type="button"
								onClick={() => void handleJob("simulation")}
								className="operator-button-secondary"
							>
								{actionLoading === "simulation"
									? "Running simulation..."
									: "Run simulation"}
							</button>
							<button
								type="button"
								onClick={() => void handleJob("plan")}
								className="operator-button-secondary"
							>
								{actionLoading === "plan" ? "Saving plan..." : "Save plan"}
							</button>
							<button
								type="button"
								onClick={() => void handleExportReport()}
								className="operator-button-secondary"
							>
								{actionLoading === "report" ? "Exporting..." : "Export report"}
							</button>
						</div>
					</SectionCard>

					<SectionCard
						eyebrow="Source connection"
						title="Current source baseline"
						description="Source details, credential references, and the latest snapshot evidence stay attached to the workspace."
					>
						{latestSource ? (
							<div className="grid gap-3">
								<HistoryRow label="Source name" value={latestSource.name} />
								<HistoryRow label="Platform" value={latestSource.platform} />
								<HistoryRow label="Address" value={latestSource.address} />
								<HistoryRow
									label="Credential reference"
									value={latestSource.credential_ref || "Not set"}
								/>
								<HistoryRow
									label="Latest discovery"
									value={
										latestSource.last_discovered_at
											? formatTimestamp(latestSource.last_discovered_at)
											: "No discovery recorded"
									}
								/>
								<HistoryRow
									label="Latest snapshot"
									value={
										latestSource.last_snapshot_id || "No snapshot recorded"
									}
								/>
							</div>
						) : (
							<EmptyState
								title="No source connection recorded"
								message="Create or reload a workspace to attach a source connection before discovery."
							/>
						)}
					</SectionCard>
				</div>
			</section>

			<section className="grid gap-5 xl:grid-cols-[0.95fr_1.05fr]">
				<SectionCard
					eyebrow="Readiness"
					title="Simulation and handoff state"
					description="Readiness status, recommendations, policy issues, and saved plan evidence stay visible before export."
				>
					<div className="grid gap-3 md:grid-cols-2">
						<Metric
							label="Readiness status"
							value={state.selectedWorkspace?.readiness?.status ?? "pending"}
							tone={readinessTone(state.selectedWorkspace?.readiness?.status)}
						/>
						<Metric
							label="Selected workloads"
							value={String(
								state.selectedWorkspace?.readiness?.selected_workload_count ??
									inventoryWorkspace.selectedIds.length,
							)}
							tone={
								inventoryWorkspace.selectedIds.length > 0 ? "accent" : "neutral"
							}
						/>
						<Metric
							label="Policy issues"
							value={String(
								state.selectedWorkspace?.readiness?.policy_violation_count ?? 0,
							)}
							tone={
								(state.selectedWorkspace?.readiness?.policy_violation_count ??
									0) > 0
									? "warning"
									: "success"
							}
						/>
						<Metric
							label="Recommendations"
							value={String(
								state.selectedWorkspace?.readiness?.recommendation_count ?? 0,
							)}
							tone={
								(state.selectedWorkspace?.readiness?.recommendation_count ??
									0) > 0
									? "info"
									: "neutral"
							}
						/>
					</div>

					{(state.selectedWorkspace?.readiness?.blocking_issues?.length ?? 0) >
					0 ? (
						<IssueList
							title="Blocking issues"
							items={state.selectedWorkspace?.readiness?.blocking_issues ?? []}
							tone="danger"
						/>
					) : null}

					{(state.selectedWorkspace?.readiness?.warning_issues?.length ?? 0) >
					0 ? (
						<IssueList
							title="Warnings"
							items={state.selectedWorkspace?.readiness?.warning_issues ?? []}
							tone="warning"
						/>
					) : null}

					{reportHistory.length > 0 ? (
						<div className="mt-4 metric-surface">
							<p className="operator-kicker">Latest export</p>
							<p className="mt-2 text-sm font-semibold text-ink">
								{reportHistory[0].file_name}
							</p>
							<p className="mt-1 text-sm text-slate-600">
								Exported {formatTimestamp(reportHistory[0].exported_at)}
							</p>
						</div>
					) : null}
				</SectionCard>

				<SectionCard
					eyebrow="Job history"
					title="Persisted background jobs"
					description="Queued, running, and failed jobs stay attached to the workspace so operators can retry, correlate, and hand off issues cleanly."
				>
					{jobHistory.length === 0 ? (
						<EmptyState
							title="No jobs recorded yet"
							message="Discovery, graph, simulation, and plan requests will appear here with status, request correlation, and retry guidance."
						/>
					) : (
						<div className="space-y-3">
							{jobHistory.slice(0, 8).map((job) => (
								<div
									key={job.id}
									className="rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-4"
								>
									<div className="flex flex-wrap items-start justify-between gap-3">
										<div className="space-y-2">
											<div className="flex flex-wrap items-center gap-2">
												<StatusBadge tone={workspaceJobTone(job.status)}>
													{job.type}
												</StatusBadge>
												<StatusBadge tone={workspaceJobTone(job.status)}>
													{job.status}
												</StatusBadge>
												{job.retryable ? (
													<StatusBadge tone="info">retryable</StatusBadge>
												) : null}
											</div>
											<p className="text-sm font-semibold text-ink">
												{job.message ?? "No job message recorded."}
											</p>
											<p className="text-xs text-slate-500">
												Requested {formatTimestamp(job.requested_at)}
												{job.requested_by ? ` by ${job.requested_by}` : ""} ·
												Correlation ID {job.correlation_id ?? "n/a"}
											</p>
											{job.error ? (
												<p className="text-xs text-rose-700">{job.error}</p>
											) : null}
										</div>
										<div className="flex flex-wrap gap-2">
											<button
												type="button"
												onClick={() =>
													void refreshWorkspaceDetail(
														state.selectedWorkspace?.id ?? "",
													)
												}
												className="operator-button-secondary px-3 py-2 text-xs"
											>
												Refresh status
											</button>
											{job.retryable || job.status === "failed" ? (
												<button
													type="button"
													onClick={() => void handleRetryJob(job)}
													disabled={actionLoading === `retry:${job.id}`}
													className="operator-button px-3 py-2 text-xs"
												>
													{actionLoading === `retry:${job.id}`
														? "Retrying..."
														: "Retry"}
												</button>
											) : null}
										</div>
									</div>
								</div>
							))}
						</div>
					)}
				</SectionCard>
			</section>

			{rows.length === 0 ? (
				<EmptyState
					title="No discovered workloads yet"
					message="Run discovery for this workspace to persist a snapshot baseline before inspection, simulation, or plan generation."
				/>
			) : (
				<section className="grid gap-5 min-[1800px]:grid-cols-[minmax(0,1.2fr)_minmax(360px,0.8fr)]">
					<InventoryTable
						rows={inventoryWorkspace.filteredRows}
						totalCount={rows.length}
						filteredCount={inventoryWorkspace.filteredRows.length}
						selectedCount={inventoryWorkspace.selectedIds.length}
						hasActiveFilters={inventoryWorkspace.hasActiveFilters}
						loading={state.refreshing}
						refreshing={state.refreshing}
						availablePlatforms={inventoryWorkspace.availablePlatforms}
						filters={inventoryWorkspace.filters}
						sortKey={inventoryWorkspace.sortKey}
						sortDirection={inventoryWorkspace.sortDirection}
						activeWorkloadId={inventoryWorkspace.activeRow?.id ?? null}
						selectedIds={inventoryWorkspace.selectedIds}
						actions={
							<button
								type="button"
								onClick={() =>
									void handleWorkspaceUpdate(
										{ selected_workload_ids: inventoryWorkspace.selectedIds },
										"selection",
									)
								}
								className="operator-button"
							>
								{actionLoading === "selection"
									? "Saving selection..."
									: "Save selection"}
							</button>
						}
						onFiltersChange={inventoryWorkspace.updateFilters}
						onSortChange={inventoryWorkspace.changeSort}
						onToggleSelection={inventoryWorkspace.toggleSelection}
						onToggleSelectAllVisible={inventoryWorkspace.toggleSelectAllVisible}
						onClearSelection={inventoryWorkspace.clearSelection}
						onResetFilters={inventoryWorkspace.resetFilters}
						onFocusWorkload={handleFocusWorkload}
					/>
					<div
						ref={detailPanelRef}
						tabIndex={-1}
						className="scroll-mt-6 outline-none"
					>
						<WorkloadDetailPanel
							row={inventoryWorkspace.activeRow}
							latestSnapshot={latestSnapshot}
							onPrimaryAction={handleSaveSelection}
							primaryActionLabel="Save selection"
							primaryActionPendingLabel="Saving selection..."
							primaryActionBusy={actionLoading === "selection"}
						/>
					</div>
				</section>
			)}

			<section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
				<DependencyGraph graph={graph} />
				<SectionCard
					eyebrow="Notes and artifacts"
					title="Operator commentary and exports"
					description="Notes, approvals, and exported reports stay attached to the workspace so pilot evidence can be handed off without rebuilding context."
				>
					<label className="block">
						<span className="operator-kicker">Operator note</span>
						<textarea
							value={noteDraft}
							onChange={(event) => setNoteDraft(event.target.value)}
							className="operator-textarea mt-2"
							placeholder="Capture operator notes, pilot caveats, or approval context."
						/>
					</label>
					<div className="mt-4 flex flex-wrap gap-2">
						<button
							type="button"
							onClick={() =>
								void handleWorkspaceUpdate(
									{
										notes: [
											...(state.selectedWorkspace?.notes ?? []),
											{
												id: "",
												kind: "operator",
												author:
													state.selectedWorkspace?.tenant_id ?? "operator",
												body: noteDraft,
												created_at: "",
											} as WorkspaceNote,
										],
									},
									"notes",
								)
							}
							className="operator-button"
							disabled={noteDraft.trim() === ""}
						>
							{actionLoading === "notes" ? "Saving note..." : "Save note"}
						</button>
						<button
							type="button"
							onClick={() => void handleExportReport()}
							className="operator-button-secondary"
						>
							Download pilot report
						</button>
					</div>

					<div className="mt-5 space-y-4">
						<ArtifactBlock
							title="Recent notes"
							emptyMessage="No operator notes saved yet."
						>
							{noteHistory.slice(0, 4).map((note) => (
								<div
									key={`${note.author}-${note.created_at}-${note.body}`}
									className="metric-surface"
								>
									<div className="flex flex-wrap items-center gap-2">
										<StatusBadge
											tone={note.kind === "operator" ? "accent" : "neutral"}
										>
											{note.kind}
										</StatusBadge>
										<StatusBadge tone="neutral">{note.author}</StatusBadge>
									</div>
									<p className="mt-3 text-sm text-slate-700">{note.body}</p>
									<p className="mt-2 text-xs text-slate-500">
										{formatTimestamp(note.created_at)}
									</p>
								</div>
							))}
						</ArtifactBlock>

						<ArtifactBlock
							title="Export history"
							emptyMessage="No reports exported yet."
						>
							{reportHistory.slice(0, 4).map((report) => (
								<div key={report.id} className="metric-surface">
									<p className="text-sm font-semibold text-ink">
										{report.file_name}
									</p>
									<p className="mt-1 text-sm text-slate-600">
										{report.name} · {report.format} ·{" "}
										{formatTimestamp(report.exported_at)}
									</p>
									<p className="mt-2 text-xs text-slate-500">
										Correlation ID {report.correlation_id ?? "n/a"}
									</p>
								</div>
							))}
						</ArtifactBlock>

						<ArtifactBlock
							title="Approvals"
							emptyMessage="No approval records attached yet."
						>
							{(state.selectedWorkspace?.approvals ?? [])
								.slice(0, 4)
								.map((approval) => (
									<div key={approval.id} className="metric-surface">
										<div className="flex flex-wrap items-center gap-2">
											<StatusBadge
												tone={
													approval.status === "approved"
														? "success"
														: approval.status === "rejected"
															? "danger"
															: "warning"
												}
											>
												{approval.status}
											</StatusBadge>
											<StatusBadge tone="neutral">{approval.stage}</StatusBadge>
										</div>
										<p className="mt-3 text-sm text-slate-700">
											{approval.approved_by
												? `Handled by ${approval.approved_by}`
												: "Pending operator approval"}
											{approval.ticket ? ` · Ticket ${approval.ticket}` : ""}
										</p>
										{approval.notes ? (
											<p className="mt-2 text-sm text-slate-600">
												{approval.notes}
											</p>
										) : null}
										<p className="mt-2 text-xs text-slate-500">
											{formatTimestamp(approval.created_at)}
										</p>
									</div>
								))}
						</ArtifactBlock>
					</div>
				</SectionCard>
			</section>
		</div>
	);
}

function WorkspaceCreateForm({
	createForm,
	validation,
	creating,
	onChange,
	onSubmit,
}: {
	createForm: CreateFormState;
	validation: Record<keyof CreateFormState, string | null>;
	creating: boolean;
	onChange: (value: CreateFormState) => void;
	onSubmit: () => void;
}) {
	return (
		<SectionCard
			eyebrow="Workspace intake"
			title="Create a pilot workspace"
			description="Use the lab defaults or adjust the source and target assumptions before starting a new assessment."
		>
			<div className="grid gap-4 lg:grid-cols-2">
				<WorkspaceField
					label="Workspace name"
					value={createForm.name}
					onChange={(value) => onChange({ ...createForm, name: value })}
					error={validation.name}
				/>
				<WorkspaceField
					label="Description"
					value={createForm.description}
					onChange={(value) => onChange({ ...createForm, description: value })}
				/>
				<WorkspaceField
					label="Source name"
					value={createForm.sourceName}
					onChange={(value) => onChange({ ...createForm, sourceName: value })}
					error={validation.sourceName}
				/>
				<WorkspaceSelectField
					label="Source platform"
					value={createForm.sourcePlatform}
					options={PLATFORM_OPTIONS}
					onChange={(value) =>
						onChange({ ...createForm, sourcePlatform: value as Platform })
					}
				/>
				<WorkspaceField
					label="Source address"
					value={createForm.sourceAddress}
					onChange={(value) =>
						onChange({ ...createForm, sourceAddress: value })
					}
					error={validation.sourceAddress}
				/>
				<WorkspaceField
					label="Credential reference"
					value={createForm.sourceCredentialRef}
					onChange={(value) =>
						onChange({ ...createForm, sourceCredentialRef: value })
					}
					error={validation.sourceCredentialRef}
				/>
				<WorkspaceSelectField
					label="Target platform"
					value={createForm.targetPlatform}
					options={PLATFORM_OPTIONS}
					onChange={(value) =>
						onChange({ ...createForm, targetPlatform: value as Platform })
					}
				/>
				<WorkspaceField
					label="Target address"
					value={createForm.targetAddress}
					onChange={(value) =>
						onChange({ ...createForm, targetAddress: value })
					}
					error={validation.targetAddress}
				/>
				<WorkspaceField
					label="Default host"
					value={createForm.defaultHost}
					onChange={(value) => onChange({ ...createForm, defaultHost: value })}
					error={validation.defaultHost}
				/>
				<WorkspaceField
					label="Default storage"
					value={createForm.defaultStorage}
					onChange={(value) =>
						onChange({ ...createForm, defaultStorage: value })
					}
					error={validation.defaultStorage}
				/>
				<WorkspaceField
					label="Default network"
					value={createForm.defaultNetwork}
					onChange={(value) =>
						onChange({ ...createForm, defaultNetwork: value })
					}
					error={validation.defaultNetwork}
				/>
			</div>
			<div className="mt-4 flex flex-wrap gap-2">
				<button
					type="button"
					onClick={onSubmit}
					disabled={creating}
					className="operator-button"
				>
					{creating ? "Creating..." : "Create workspace"}
				</button>
			</div>
		</SectionCard>
	);
}

function WorkspaceField({
	label,
	value,
	onChange,
	hint,
	error,
}: {
	label: string;
	value: string;
	onChange: (value: string) => void;
	hint?: string;
	error?: string | null;
}) {
	return (
		<label className="block">
			<span className="operator-kicker">{label}</span>
			<input
				value={value}
				onChange={(event) => onChange(event.target.value)}
				className="operator-input mt-2"
			/>
			{error ? (
				<span className="mt-2 block text-xs text-rose-700">{error}</span>
			) : null}
			{!error && hint ? (
				<span className="mt-2 block text-xs text-slate-500">{hint}</span>
			) : null}
		</label>
	);
}

function WorkspaceSelectField({
	label,
	value,
	options,
	onChange,
}: {
	label: string;
	value: string;
	options: string[];
	onChange: (value: string) => void;
}) {
	return (
		<label className="block">
			<span className="operator-kicker">{label}</span>
			<select
				value={value}
				onChange={(event) => onChange(event.target.value)}
				className="operator-select mt-2"
			>
				{options.map((option) => (
					<option key={option} value={option}>
						{option}
					</option>
				))}
			</select>
		</label>
	);
}

function WorkspaceNumberField({
	label,
	value,
	onChange,
}: {
	label: string;
	value?: number;
	onChange: (value?: number) => void;
}) {
	return (
		<label className="block">
			<span className="operator-kicker">{label}</span>
			<input
				type="number"
				min="0"
				value={value ?? ""}
				onChange={(event) => {
					const nextValue = event.target.value.trim();
					onChange(nextValue === "" ? undefined : Number(nextValue));
				}}
				className="operator-input mt-2"
			/>
		</label>
	);
}

function WorkspaceCheckbox({
	label,
	checked,
	onChange,
}: {
	label: string;
	checked: boolean;
	onChange: (checked: boolean) => void;
}) {
	return (
		<label className="metric-surface flex items-start gap-3">
			<input
				type="checkbox"
				checked={checked}
				onChange={(event) => onChange(event.target.checked)}
				className="mt-1 h-4 w-4 rounded border-slate-300"
			/>
			<span>
				<span className="block text-sm font-semibold text-ink">{label}</span>
				<span className="mt-1 block text-sm text-slate-600">
					Persist this plan behavior in the workspace so future runs and exports
					use the same intent.
				</span>
			</span>
		</label>
	);
}

function WorkflowStepCard({
	step,
	index,
}: {
	step: WorkflowStep;
	index: number;
}) {
	return (
		<div
			className={`rounded-[24px] border px-4 py-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)] ${workflowStepClasses(step.status)}`}
		>
			<div className="flex items-start gap-3">
				<span
					className={`inline-flex h-11 w-11 items-center justify-center rounded-[18px] text-sm font-semibold shadow-[0_10px_20px_rgba(15,23,42,0.12)] ${workflowIndexClasses(step.status)}`}
				>
					{String(index).padStart(2, "0")}
				</span>
				<div className="min-w-0">
					<div className="flex flex-wrap items-center gap-2">
						<p className="text-sm font-semibold text-ink">{step.label}</p>
						<StatusBadge tone={workflowStepTone(step.status)}>
							{step.status === "current" ? "next" : step.status}
						</StatusBadge>
					</div>
					<p className="mt-2 text-sm leading-6 text-slate-600">
						{step.summary}
					</p>
					<p className="mt-2 text-xs text-slate-500">{step.evidence}</p>
				</div>
			</div>
		</div>
	);
}

function Metric({
	label,
	value,
	tone = "neutral",
}: {
	label: string;
	value: string;
	tone?: StatusTone;
}) {
	return <StatCard label={label} value={value} badge={{ label, tone }} />;
}

function ArtifactBlock({
	title,
	emptyMessage,
	children,
}: {
	title: string;
	emptyMessage: string;
	children: ReactNode;
}) {
	const hasChildren = Array.isArray(children)
		? children.length > 0
		: Boolean(children);

	return (
		<div>
			<p className="operator-kicker">{title}</p>
			<div className="mt-3 space-y-3">
				{hasChildren ? (
					children
				) : (
					<InlineNotice message={emptyMessage} tone="neutral" />
				)}
			</div>
		</div>
	);
}

function IssueList({
	title,
	items,
	tone,
}: {
	title: string;
	items: string[];
	tone: StatusTone;
}) {
	return (
		<div className="mt-4">
			<div className="flex items-center gap-2">
				<p className="operator-kicker">{title}</p>
				<StatusBadge tone={tone}>{items.length}</StatusBadge>
			</div>
			<div className="mt-3 space-y-2">
				{items.map((item) => (
					<InlineNotice key={item} message={item} tone={tone} />
				))}
			</div>
		</div>
	);
}

function HistoryRow({ label, value }: { label: string; value: string }) {
	return <StatCard label={label} value={value} />;
}

function buildWorkflowSteps(
	workspace: PilotWorkspace | null,
	workloadCount: number,
	selectedWorkloadCount: number,
): WorkflowStep[] {
	const states = [
		Boolean(workspace),
		(workspace?.snapshots?.length ?? 0) > 0,
		workloadCount > 0 || Boolean(workspace?.graph),
		Boolean(workspace?.simulation) || Boolean(workspace?.readiness),
		Boolean(workspace?.saved_plan),
		(workspace?.reports?.length ?? 0) > 0,
	];
	const currentIndex = states.findIndex((state) => !state);

	return [
		{
			id: "intake",
			label: "Create workspace",
			summary:
				"Capture the source connection, credential reference, target platform, and initial plan defaults.",
			evidence: workspace
				? `Workspace ${workspace.name} is persisted.`
				: "No workspace exists yet.",
			status: stepStatus(states[0], currentIndex, 0),
		},
		{
			id: "discovery",
			label: "Run discovery",
			summary:
				"Persist a snapshot baseline so later graph, readiness, and report views reflect the same source evidence.",
			evidence:
				(workspace?.snapshots?.length ?? 0) > 0
					? `${workspace?.snapshots?.length ?? 0} snapshot(s) recorded.`
					: "No snapshots saved yet.",
			status: stepStatus(states[1], currentIndex, 1),
		},
		{
			id: "inspect",
			label: "Inspect workloads",
			summary:
				"Review the discovered workloads, dependency context, and the selection you want to carry into planning.",
			evidence:
				workloadCount > 0
					? `${workloadCount} workload(s) available, ${selectedWorkloadCount} selected.`
					: "No workload inventory loaded yet.",
			status: stepStatus(states[2], currentIndex, 2),
		},
		{
			id: "simulate",
			label: "Simulate readiness",
			summary:
				"Derive readiness, policy issues, and recommendation signals against the same persisted workspace.",
			evidence: workspace?.readiness
				? `Readiness is ${workspace.readiness.status}.`
				: "Simulation has not been run yet.",
			status: stepStatus(states[3], currentIndex, 3),
		},
		{
			id: "plan",
			label: "Save plan",
			summary:
				"Persist a dry-run migration plan so operator review, approval, and export can use the same state.",
			evidence: workspace?.saved_plan
				? `Saved plan ${workspace.saved_plan.spec_name}.`
				: "No plan saved yet.",
			status: stepStatus(states[4], currentIndex, 4),
		},
		{
			id: "report",
			label: "Export report",
			summary:
				"Keep the handoff artifact attached to the workspace so review and pilot follow-up stay traceable.",
			evidence:
				(workspace?.reports?.length ?? 0) > 0
					? `${workspace?.reports?.length ?? 0} report export(s) recorded.`
					: "No report exported yet.",
			status: stepStatus(states[5], currentIndex, 5),
		},
	];
}

function stepStatus(
	completed: boolean,
	currentIndex: number,
	index: number,
): WorkflowStep["status"] {
	if (completed) {
		return "complete";
	}
	return currentIndex === index || currentIndex === -1 ? "current" : "upcoming";
}

function recommendedWorkspaceAction(
	workspace: PilotWorkspace | null,
	workloadCount: number,
	selectedWorkloadCount: number,
): string {
	if (!workspace) {
		return "Create the first workspace";
	}
	if ((workspace.snapshots?.length ?? 0) === 0) {
		return "Run discovery to persist the baseline";
	}
	if (!workspace.graph) {
		return "Build the dependency graph";
	}
	if (workloadCount > 0 && selectedWorkloadCount === 0) {
		return "Review workloads and save a selection";
	}
	if (!workspace.simulation && !workspace.readiness) {
		return "Run readiness simulation";
	}
	if (!workspace.saved_plan) {
		return "Save the migration plan";
	}
	if ((workspace.reports?.length ?? 0) === 0) {
		return "Export the pilot report";
	}
	return "Review notes, approvals, and exported evidence";
}

function validateCreateForm(
	createForm: CreateFormState,
): Record<keyof CreateFormState, string | null> {
	return {
		name: createForm.name.trim() === "" ? "Workspace name is required." : null,
		description: null,
		sourceName:
			createForm.sourceName.trim() === "" ? "Source name is required." : null,
		sourcePlatform: null,
		sourceAddress:
			createForm.sourceAddress.trim() === ""
				? "Source address is required."
				: null,
		sourceCredentialRef:
			createForm.sourceCredentialRef.trim() === ""
				? "Credential reference is required."
				: null,
		targetPlatform: null,
		targetAddress:
			createForm.targetAddress.trim() === ""
				? "Target address is required."
				: null,
		defaultHost:
			createForm.defaultHost.trim() === "" ? "Default host is required." : null,
		defaultStorage:
			createForm.defaultStorage.trim() === ""
				? "Default storage is required."
				: null,
		defaultNetwork:
			createForm.defaultNetwork.trim() === ""
				? "Default network is required."
				: null,
	};
}

async function loadWorkspaceInventory(
	snapshots: WorkspaceSnapshot[],
): Promise<DiscoveryResult | null> {
	if (snapshots.length === 0) {
		return null;
	}
	const results = await Promise.all(
		snapshots.map((snapshot) => getSnapshot(snapshot.snapshot_id)),
	);
	return mergeDiscoveryResults(results);
}

function mergeDiscoveryResults(results: DiscoveryResult[]): DiscoveryResult {
	return results.reduce<DiscoveryResult>(
		(merged, current) => ({
			source: merged.source || current.source,
			platform: merged.platform || current.platform,
			vms: [...merged.vms, ...(current.vms ?? [])],
			networks: [...(merged.networks ?? []), ...(current.networks ?? [])],
			datastores: [...(merged.datastores ?? []), ...(current.datastores ?? [])],
			hosts: [...(merged.hosts ?? []), ...(current.hosts ?? [])],
			clusters: [...(merged.clusters ?? []), ...(current.clusters ?? [])],
			resource_pools: [
				...(merged.resource_pools ?? []),
				...(current.resource_pools ?? []),
			],
			discovered_at: merged.discovered_at ?? current.discovered_at,
			errors: [...(merged.errors ?? []), ...(current.errors ?? [])],
			duration: (merged.duration ?? 0) + (current.duration ?? 0),
		}),
		{ vms: [] },
	);
}

function workspaceJobRetryPayload(
	job: WorkspaceJob,
	fallbackSelectedIDs: string[],
): {
	type: WorkspaceJobType;
	requested_by?: string;
	source_connection_ids?: string[];
	selected_workload_ids?: string[];
	simulation?: SimulationRequest;
} {
	const input = job.input_json ?? {};
	const selectedWorkloadIDs = stringArrayFromUnknown(
		input.selected_workload_ids,
	);
	const fallbackSelection =
		selectedWorkloadIDs.length > 0
			? selectedWorkloadIDs
			: job.type === "simulation" || job.type === "plan"
				? fallbackSelectedIDs
				: [];

	return {
		type: input.type ?? job.type,
		requested_by:
			input.requested_by?.trim() || job.requested_by?.trim() || undefined,
		source_connection_ids: toOptionalArray(
			stringArrayFromUnknown(input.source_connection_ids),
		),
		selected_workload_ids: toOptionalArray(fallbackSelection),
		simulation: simulationRequestFromUnknown(input.simulation),
	};
}

function simulationRequestFromUnknown(
	value: unknown,
): SimulationRequest | undefined {
	if (!value || typeof value !== "object") {
		return undefined;
	}

	const candidate = value as Partial<SimulationRequest>;
	if (!candidate.target_platform) {
		return undefined;
	}

	return {
		target_platform: candidate.target_platform,
		vm_ids: toOptionalArray(stringArrayFromUnknown(candidate.vm_ids)),
		include_all: candidate.include_all === true ? true : undefined,
	};
}

function stringArrayFromUnknown(value: unknown): string[] {
	if (!Array.isArray(value)) {
		return [];
	}

	return Array.from(
		new Set(
			value
				.filter((item): item is string => typeof item === "string")
				.map((item) => item.trim())
				.filter((item) => item !== ""),
		),
	);
}

function toOptionalArray(values: string[]): string[] | undefined {
	return values.length > 0 ? values : undefined;
}

function formatTimestamp(value?: string): string {
	if (!value) {
		return "at an unknown time";
	}

	const parsed = new Date(value);
	if (Number.isNaN(parsed.getTime())) {
		return value;
	}

	return parsed.toLocaleString(undefined, {
		dateStyle: "medium",
		timeStyle: "short",
	});
}

function workspaceStatusTone(status?: PilotWorkspace["status"]): StatusTone {
	switch (status) {
		case "reported":
			return "success";
		case "planned":
			return "accent";
		case "simulated":
			return "info";
		case "graph-ready":
		case "discovered":
			return "warning";
		default:
			return "neutral";
	}
}

function workspaceJobTone(status: WorkspaceJobStatus): StatusTone {
	switch (status) {
		case "succeeded":
			return "success";
		case "failed":
			return "danger";
		default:
			return "warning";
	}
}

function readinessTone(
	status?: PilotWorkspace["readiness"] extends infer T
		? T extends { status?: infer U }
			? U
			: never
		: never,
): StatusTone {
	switch (status) {
		case "ready":
			return "success";
		case "attention":
			return "warning";
		case "blocked":
			return "danger";
		default:
			return "neutral";
	}
}

function workflowStepTone(status: WorkflowStep["status"]): StatusTone {
	switch (status) {
		case "complete":
			return "success";
		case "current":
			return "accent";
		default:
			return "neutral";
	}
}

function workflowStepClasses(status: WorkflowStep["status"]): string {
	switch (status) {
		case "complete":
			return "border-emerald-200 bg-emerald-50/60";
		case "current":
			return "border-sky-200 bg-sky-50/70";
		default:
			return "border-slate-200 bg-slate-50/70";
	}
}

function workflowIndexClasses(status: WorkflowStep["status"]): string {
	switch (status) {
		case "complete":
			return "bg-emerald-700 text-white";
		case "current":
			return "bg-ink text-white";
		default:
			return "bg-white text-slate-600";
	}
}
