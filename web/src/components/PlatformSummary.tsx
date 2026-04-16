import type { DiscoveryResult } from "../types";

interface PlatformSummaryProps {
	inventory: DiscoveryResult | null;
}

export function PlatformSummary({ inventory }: PlatformSummaryProps) {
	const rows = Object.values(
		(inventory?.vms ?? []).reduce<
			Record<
				string,
				{ platform: string; count: number; cpu: number; memory: number }
			>
		>((accumulator, vm) => {
			const item = accumulator[vm.platform] ?? {
				platform: vm.platform,
				count: 0,
				cpu: 0,
				memory: 0,
			};
			item.count += 1;
			item.cpu += vm.cpu_count;
			item.memory += vm.memory_mb;
			accumulator[vm.platform] = item;
			return accumulator;
		}, {}),
	);
	const chartSummary =
		rows.length === 0
			? "Bar chart showing no discovered platforms."
			: `Bar chart showing ${rows.reduce((total, row) => total + row.count, 0)} VMs across ${rows.length} platforms.`;
	const maxCount = rows.reduce(
		(currentMax, row) => Math.max(currentMax, row.count),
		0,
	);

	return (
		<section className="grid gap-5 xl:grid-cols-[1.3fr_1fr]">
			<div className="panel min-w-0 p-5">
				<h2 className="font-display text-2xl text-ink">Platform Totals</h2>
				<p className="mt-1 text-sm text-slate-500">
					A quick capacity snapshot across the discovered estate.
				</p>

				<div className="mt-5 grid gap-4 md:grid-cols-3">
					{rows.map((row) => (
						<article key={row.platform} className="rounded-2xl bg-slate-50 p-4">
							<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
								{row.platform}
							</p>
							<p className="mt-3 font-display text-3xl text-ink">{row.count}</p>
							<p className="mt-1 text-sm text-slate-500">
								{row.cpu} vCPU / {row.memory.toLocaleString()} MB
							</p>
						</article>
					))}
				</div>
			</div>

			<div className="panel min-w-0 h-[320px] p-5">
				<h2 className="font-display text-2xl text-ink">Hypervisor Spread</h2>
				<p className="mt-1 text-sm text-slate-500">
					Compare VM counts side by side before planning a migration wave.
				</p>
				<div
					className="mt-5 h-[220px] space-y-3 overflow-y-auto"
					aria-label={chartSummary}
				>
					{rows.length === 0 ? (
						<p className="rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
							No platform totals are available yet.
						</p>
					) : (
						rows.map((row) => (
							<article
								key={row.platform}
								className="rounded-2xl bg-slate-50 px-4 py-3"
							>
								<div className="flex items-center justify-between gap-3">
									<p className="font-semibold text-ink">{row.platform}</p>
									<p className="text-sm text-slate-600">
										{row.count} VM{row.count === 1 ? "" : "s"}
									</p>
								</div>
								<div aria-hidden="true" className="mt-3 h-2.5 rounded-full bg-slate-200">
									<div
										className="h-full rounded-full bg-ink"
										style={{
											width:
												maxCount === 0
													? "0%"
													: `${Math.max((row.count / maxCount) * 100, 8)}%`,
										}}
									/>
								</div>
								<p className="mt-2 text-xs text-slate-500">
									{row.cpu} vCPU / {row.memory.toLocaleString()} MB
								</p>
							</article>
						))
					)}
				</div>
			</div>
		</section>
	);
}
