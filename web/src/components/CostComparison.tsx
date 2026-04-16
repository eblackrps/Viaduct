import type { PlatformComparison } from "../types";

interface CostComparisonProps {
	comparisons: PlatformComparison[];
}

export function CostComparison({ comparisons }: CostComparisonProps) {
	return (
		<section className="panel p-5">
			<div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
				<div>
					<p className="font-display text-2xl text-ink">Cost Modeling</p>
					<p className="text-sm text-slate-500">
						Compare each workload across the available platform pricing
						profiles.
					</p>
				</div>
				<p className="text-sm font-semibold text-slate-500">
					{comparisons.length} workloads compared
				</p>
			</div>

			<div className="mt-5 space-y-3">
				{comparisons.map((comparison) => (
					<article
						key={comparison.vm.id || comparison.vm.name}
						className="rounded-2xl bg-slate-50 p-4"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
							<div>
								<p className="font-semibold text-ink">{comparison.vm.name}</p>
								<p className="text-sm text-slate-500">
									Current: {comparison.vm.platform} · Cheapest:{" "}
									{comparison.cheapest_platform}
								</p>
							</div>
							<div className="rounded-full bg-emerald-100 px-4 py-2 text-sm font-semibold text-emerald-700">
								Save ${comparison.monthly_savings.toFixed(2)}/mo
							</div>
						</div>

						<div className="mt-4 grid gap-3 md:grid-cols-3">
							{Object.entries(comparison.cost_by_platform).map(
								([platform, cost]) => (
									<div
										key={platform}
										className="rounded-2xl border border-slate-200 bg-white px-4 py-3"
									>
										<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
											{platform}
										</p>
										<p className="mt-2 font-display text-2xl text-ink">
											${cost.monthly_total.toFixed(2)}
										</p>
										<p className="text-sm text-slate-500">
											${cost.annual_total.toFixed(2)} annual
										</p>
									</div>
								),
							)}
						</div>
					</article>
				))}

				{comparisons.length === 0 && (
					<p className="rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
						No cost comparisons are available yet.
					</p>
				)}
			</div>
		</section>
	);
}
