import type { PlatformComparison } from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatCard } from "./primitives/StatCard";
import { StatusBadge } from "./primitives/StatusBadge";

interface CostComparisonProps {
	comparisons: PlatformComparison[];
}

export function CostComparison({ comparisons }: CostComparisonProps) {
	return (
		<SectionCard
			title="Cost modeling"
			description="Compare each workload across the available platform pricing profiles."
			actions={
				<StatusBadge tone="neutral">
					{comparisons.length} workload{comparisons.length === 1 ? "" : "s"}{" "}
					compared
				</StatusBadge>
			}
		>
			<div className="space-y-3">
				{comparisons.map((comparison) => (
					<article
						key={comparison.vm.id || comparison.vm.name}
						className="list-card"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">{comparison.vm.name}</p>
								<p className="mt-1 text-sm text-slate-500">
									Current {comparison.vm.platform} • Cheapest{" "}
									{comparison.cheapest_platform}
								</p>
							</div>
							<StatusBadge tone="success">
								Save ${comparison.monthly_savings.toFixed(2)}/mo
							</StatusBadge>
						</div>

						<div className="mt-4 grid gap-3 md:grid-cols-3">
							{Object.entries(comparison.cost_by_platform).map(
								([platform, cost]) => (
									<StatCard
										key={platform}
										label={platform}
										value={`$${cost.monthly_total.toFixed(2)}`}
										detail={`$${cost.annual_total.toFixed(2)} annual`}
										emphasis="large"
									/>
								),
							)}
						</div>
					</article>
				))}

				{comparisons.length === 0 ? (
					<InlineNotice
						message="No cost comparisons are available yet."
						tone="neutral"
					/>
				) : null}
			</div>
		</SectionCard>
	);
}
