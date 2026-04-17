import type { MigrationPlan } from "../../types";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";

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
				<StatCard
					label="Generated"
					value={new Date(plan.generated_at).toLocaleString()}
				/>
				<StatCard label="Workloads" value={plan.total_workloads} />
				<StatCard label="Waves" value={plan.waves.length} />
				<StatCard
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

			<div className="mt-5 flex flex-wrap gap-2">
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
				{plan.window?.not_before ? (
					<StatusBadge tone="neutral">
						Window opens {new Date(plan.window.not_before).toLocaleString()}
					</StatusBadge>
				) : null}
				{plan.window?.not_after ? (
					<StatusBadge tone="neutral">
						Window closes {new Date(plan.window.not_after).toLocaleString()}
					</StatusBadge>
				) : null}
			</div>

			<div className="mt-5 space-y-3">
				{plan.waves.map((wave) => (
					<article
						key={wave.index}
						className="list-card text-sm text-slate-600"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">Wave {wave.index}</p>
								<p className="mt-1 text-sm text-slate-500">
									{wave.reason === "dependency_wave"
										? "Ordered by dependencies"
										: "Ordered by batch size"}
								</p>
							</div>
							<StatusBadge tone={wave.dependency_aware ? "accent" : "neutral"}>
								{wave.dependency_aware ? "Dependency-aware" : "Batch"}
							</StatusBadge>
						</div>

						<div className="mt-4 flex flex-wrap gap-2">
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

				{plan.waves.length === 0 ? (
					<InlineNotice
						message="No waves were generated because the current selectors did not match any workloads during planning."
						tone="neutral"
					/>
				) : null}
			</div>
		</SectionCard>
	);
}
