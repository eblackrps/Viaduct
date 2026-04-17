import { useEffect, useMemo, useState } from "react";
import { RefreshCcw } from "lucide-react";
import type { MigrationPhase, MigrationState } from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatCard } from "./primitives/StatCard";
import { StatusBadge } from "./primitives/StatusBadge";

interface MigrationProgressProps {
	state: MigrationState | null;
	onRollback?: () => void;
}

const phaseOrder: MigrationPhase[] = [
	"plan",
	"export",
	"convert",
	"import",
	"configure",
	"verify",
	"complete",
];

export function MigrationProgress({
	state,
	onRollback,
}: MigrationProgressProps) {
	const [elapsedLabel, setElapsedLabel] = useState("0s");

	useEffect(() => {
		if (!state) {
			return;
		}

		const update = () => {
			const started = new Date(state.started_at).getTime();
			const ended = state.completed_at
				? new Date(state.completed_at).getTime()
				: Date.now();
			const seconds = Math.max(0, Math.round((ended - started) / 1000));
			setElapsedLabel(`${seconds}s`);
		};

		update();
		const interval = window.setInterval(update, 1000);
		return () => window.clearInterval(interval);
	}, [state]);

	const percentage = useMemo(() => {
		if (!state || state.workloads.length === 0) {
			return 0;
		}

		const scores = state.workloads.map((workload) => {
			if (workload.phase === "complete") {
				return 100;
			}
			if (workload.phase === "failed") {
				return 0;
			}
			const index = Math.max(0, phaseOrder.indexOf(workload.phase));
			return Math.round((index / (phaseOrder.length - 1)) * 100);
		});

		return Math.round(
			scores.reduce((total, score) => total + score, 0) / scores.length,
		);
	}, [state]);

	if (!state) {
		return (
			<InlineNotice
				message="Start a migration to see per-workload execution progress."
				tone="neutral"
			/>
		);
	}

	return (
		<section className="space-y-5">
			<SectionCard
				title="Migration progress"
				description="Track the current phase, checkpoint status, and workload-by-workload execution detail."
				actions={
					state.phase === "failed" && onRollback ? (
						<button
							type="button"
							className="operator-button-danger"
							onClick={onRollback}
						>
							<RefreshCcw className="h-4 w-4" />
							Rollback
						</button>
					) : undefined
				}
			>
				<div className="grid gap-3 md:grid-cols-3">
					<StatCard label="Phase" value={state.phase} />
					<StatCard label="Duration" value={elapsedLabel} />
					<StatCard
						label="Pending approval"
						value={state.pending_approval ? "Yes" : "No"}
						badge={{
							label: state.pending_approval ? "Blocked" : "Clear",
							tone: state.pending_approval ? "warning" : "success",
						}}
					/>
				</div>

				<div className="mt-5 overflow-hidden rounded-full bg-slate-200">
					<div
						className="h-3 rounded-full bg-gradient-to-r from-accent via-steel to-ink transition-all"
						style={{ width: `${percentage}%` }}
					/>
				</div>
				<p className="mt-2 text-sm text-slate-500">{percentage}% complete</p>

				{state.errors && state.errors.length > 0 ? (
					<div className="mt-4 space-y-2">
						{state.errors.map((error, index) => (
							<InlineNotice
								key={`${state.id}-error-${index}`}
								message={error}
								tone="danger"
							/>
						))}
					</div>
				) : null}

				{state.checkpoints && state.checkpoints.length > 0 ? (
					<div className="mt-4 grid gap-3 md:grid-cols-3">
						{state.checkpoints.map((checkpoint) => (
							<StatCard
								key={checkpoint.phase}
								label={checkpoint.phase}
								value={checkpoint.status}
								detail={checkpoint.message}
								badge={{
									label: checkpoint.status,
									tone: checkpointTone(checkpoint.status),
								}}
							/>
						))}
					</div>
				) : null}

				{state.plan && state.plan.waves.length > 0 ? (
					<div className="mt-4">
						<InlineNotice
							tone="info"
							message={`Planned across ${state.plan.waves.length} wave(s) with batch size ${
								state.plan.wave_strategy.size ?? state.plan.waves.length
							}.`}
							title="Runbook"
						/>
					</div>
				) : null}
			</SectionCard>

			<div className="space-y-3">
				{state.workloads.map((workload) => (
					<article
						key={workload.vm.id || workload.vm.name}
						className="list-card"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">{workload.vm.name}</p>
								<p className="mt-1 text-sm text-slate-500">
									{workload.vm.platform} → {workload.target_vm_id || "pending target VM"}
								</p>
							</div>
							<StatusBadge tone={phaseTone(workload.phase)}>
								{workload.phase}
							</StatusBadge>
						</div>
						{workload.error ? (
							<div className="mt-4">
								<InlineNotice message={workload.error} tone="danger" />
							</div>
						) : null}
					</article>
				))}
			</div>
		</section>
	);
}

function phaseTone(
	phase: MigrationPhase,
): "neutral" | "info" | "success" | "warning" | "danger" | "accent" {
	switch (phase) {
		case "plan":
			return "info";
		case "export":
		case "convert":
		case "import":
		case "configure":
			return "accent";
		case "verify":
		case "complete":
			return "success";
		case "failed":
			return "danger";
		case "rolled_back":
			return "neutral";
		default:
			return "neutral";
	}
}

function checkpointTone(
	status: "pending" | "running" | "completed" | "failed",
): "neutral" | "info" | "success" | "warning" | "danger" | "accent" {
	switch (status) {
		case "running":
			return "info";
		case "completed":
			return "success";
		case "failed":
			return "danger";
		default:
			return "neutral";
	}
}
