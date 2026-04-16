import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { MigrationPlan } from "../../types";

interface MigrationPlanSummaryProps {
	title: string;
	description: string;
	plan: MigrationPlan;
}

export function MigrationPlanSummary({
	title,
	description,
	plan,
}: MigrationPlanSummaryProps) {
	return (
		<SectionCard title={title} description={description}>
			<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
				<Metric
					label="Generated"
					value={new Date(plan.generated_at).toLocaleString()}
				/>
				<Metric label="Workloads" value={`${plan.total_workloads}`} />
				<Metric label="Waves" value={`${plan.waves.length}`} />
				<Metric
					label="Approval"
					value={
						plan.requires_approval
							? plan.approval_satisfied
								? "Satisfied"
								: "Required"
							: "Not required"
					}
				/>
			</div>

			<div className="mt-4 flex flex-wrap gap-2">
				<StatusBadge
					tone={plan.wave_strategy.dependency_aware ? "accent" : "neutral"}
				>
					{plan.wave_strategy.dependency_aware
						? "Dependency-aware waves"
						: "Fixed batch waves"}
				</StatusBadge>
				<StatusBadge tone="info">
					Wave size {plan.wave_strategy.size ?? plan.waves.length}
				</StatusBadge>
				{plan.window?.not_before && (
					<StatusBadge tone="neutral">
						Window opens {new Date(plan.window.not_before).toLocaleString()}
					</StatusBadge>
				)}
				{plan.window?.not_after && (
					<StatusBadge tone="neutral">
						Window closes {new Date(plan.window.not_after).toLocaleString()}
					</StatusBadge>
				)}
			</div>

			<div className="mt-5 space-y-3">
				{plan.waves.map((wave) => (
					<article
						key={wave.index}
						className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600"
					>
						<div className="flex flex-wrap items-center justify-between gap-3">
							<div>
								<p className="font-semibold text-ink">Wave {wave.index}</p>
								<p className="text-slate-500">
									{wave.reason === "dependency_wave"
										? "Ordered by dependencies"
										: "Ordered by batch size"}
								</p>
							</div>
							<StatusBadge tone={wave.dependency_aware ? "accent" : "neutral"}>
								{wave.dependency_aware ? "Dependency-aware" : "Batch"}
							</StatusBadge>
						</div>

						<div className="mt-3 flex flex-wrap gap-2">
							{wave.workloads.map((workload) => (
								<StatusBadge
									key={`${wave.index}:${workload.vm_id}`}
									tone="neutral"
								>
									{workload.name}
								</StatusBadge>
							))}
						</div>
					</article>
				))}

				{plan.waves.length === 0 && (
					<p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
						No waves were generated because the current selectors did not match
						any workloads during planning.
					</p>
				)}
			</div>
		</SectionCard>
	);
}

function Metric({ label, value }: { label: string; value: string }) {
	return (
		<div className="rounded-2xl bg-slate-50 px-4 py-4">
			<p className="text-xs uppercase tracking-[0.18em] text-slate-500">
				{label}
			</p>
			<p className="mt-2 text-sm font-semibold text-ink">{value}</p>
		</div>
	);
}
