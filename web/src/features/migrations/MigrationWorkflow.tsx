import { ArrowRight } from "lucide-react";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { SectionCard } from "../../components/primitives/SectionCard";
import type { InventoryPlanningDraft } from "../inventory/inventoryPlanningDraft";
import {
	MigrationExecuteStage,
	MigrationMetric,
	MigrationPrepareStage,
	MigrationScopeStage,
	MigrationValidateStage,
} from "./MigrationStagePanels";
import {
	getMigrationWorkflowStatus,
	getPreflightSummary,
	getWorkflowStatusPresentation,
} from "./migrationStatus";
import { useMigrationWorkspace } from "./useMigrationWorkspace";

const stages = [
	["Scope", "Bring workloads into the plan."],
	["Prepare", "Set target and execution controls."],
	["Validate", "Run preflight and save plan state."],
	["Execute", "Start, resume, or roll back the saved plan."],
] as const;

interface MigrationWorkflowProps {
	planningDraft?: InventoryPlanningDraft | null;
	onPlanningDraftCleared?: () => void;
	onMigrationChange?: () => void;
}

export function MigrationWorkflow({
	planningDraft,
	onPlanningDraftCleared,
	onMigrationChange,
}: MigrationWorkflowProps) {
	const workspace = useMigrationWorkspace({
		planningDraft,
		onPlanningDraftCleared,
		onMigrationChange,
	});

	return (
		<section className="space-y-5">
			<SectionCard
				title="Planning workflow"
				description="Use the real backend sequence: draft, validate, save plan state, then execute."
			>
				<div className="grid gap-3 xl:grid-cols-4">
					<MigrationMetric
						label="State"
						value={workspace.workflowPresentation.label}
						tone={workspace.workflowPresentation.tone}
						detail={workspace.workflowPresentation.description}
					/>
					<MigrationMetric
						label="Scope"
						value={
							workspace.selectedWorkloads.length > 0
								? `${workspace.selectedWorkloads.length} selected`
								: "No scope"
						}
						tone={workspace.selectedWorkloads.length > 0 ? "info" : "neutral"}
						detail={
							workspace.selectedWorkloads.length > 0
								? `${workspace.sourceNetworks.length} source network(s)`
								: "Load inventory or start from inventory selection."
						}
					/>
					<MigrationMetric
						label="Preflight"
						value={
							workspace.preflight
								? `${workspace.preflight.pass_count}/${workspace.preflight.checks.length} passed`
								: "Not run"
						}
						tone={
							!workspace.preflight
								? "neutral"
								: workspace.preflightStale
									? "warning"
									: workspace.preflight.fail_count > 0
										? "danger"
										: workspace.preflight.warn_count > 0
											? "warning"
											: "success"
						}
						detail={getPreflightSummary(
							workspace.preflight,
							workspace.preflightStale,
						)}
					/>
					<MigrationMetric
						label="Saved plan"
						value={
							workspace.migrationState
								? workspace.planStale
									? "Plan stale"
									: workspace.migrationState.spec_name
								: "None"
						}
						tone={
							workspace.migrationState
								? workspace.planStale
									? "danger"
									: getWorkflowStatusPresentation(
											getMigrationWorkflowStatus(workspace.migrationState),
										).tone
								: "neutral"
						}
						detail={
							workspace.migrationState
								? workspace.migrationState.id
								: "Persist a plan before execution."
						}
					/>
				</div>
			</SectionCard>

			<SectionCard
				title="Execution flow"
				description="Move left to right from scope definition into saved execution state."
			>
				<div className="grid gap-3 xl:grid-cols-4">
					{stages.map(([label, description], index) => (
						<button
							key={label}
							type="button"
							onClick={() => workspace.setStage(index)}
							className={`rounded-[24px] border px-4 py-4 text-left transition duration-200 ${
								workspace.stage === index
									? "border-ink bg-ink text-white shadow-[0_18px_30px_rgba(15,23,42,0.22)]"
									: index < workspace.stage
										? "border-sky-200/90 bg-sky-50/90 text-sky-950"
										: "border-slate-200/80 bg-slate-50/85 text-slate-700 hover:bg-white/85"
							}`}
						>
							<p className="operator-kicker !text-inherit opacity-70">
								Step {index + 1}
							</p>
							<p className="mt-3 font-semibold">{label}</p>
							<p className="mt-2 text-sm leading-6 opacity-80">{description}</p>
						</button>
					))}
				</div>

				{workspace.error ? (
					<div className="mt-5">
						<InlineNotice message={workspace.error} tone="danger" />
					</div>
				) : null}

				{workspace.stage === 0 ? <MigrationScopeStage workspace={workspace} /> : null}
				{workspace.stage === 1 ? (
					<MigrationPrepareStage workspace={workspace} />
				) : null}
				{workspace.stage === 2 ? (
					<MigrationValidateStage workspace={workspace} />
				) : null}
				{workspace.stage === 3 ? (
					<MigrationExecuteStage workspace={workspace} />
				) : null}

				<div className="mt-6 flex flex-wrap items-center justify-between gap-3">
					<button
						type="button"
						className="operator-button-secondary"
						onClick={() =>
							workspace.setStage((current) => Math.max(0, current - 1))
						}
						disabled={workspace.stage === 0}
					>
						Back
					</button>
					{workspace.stage < stages.length - 1 ? (
						<button
							type="button"
							className="operator-button"
							onClick={() =>
								workspace.setStage((current) =>
									Math.min(stages.length - 1, current + 1),
								)
							}
							disabled={
								(workspace.stage === 0 &&
									workspace.selectedWorkloads.length === 0) ||
								(workspace.stage === 2 && !workspace.migrationState)
							}
						>
							Continue
							<ArrowRight className="h-4 w-4" />
						</button>
					) : null}
				</div>
			</SectionCard>
		</section>
	);
}
