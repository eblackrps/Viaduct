import type { DiscoveryResult } from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatCard } from "./primitives/StatCard";

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
	).sort((left, right) => right.count - left.count);
	const chartSummary =
		rows.length === 0
			? "Bar chart showing no discovered platforms."
			: `Bar chart showing ${rows.reduce((total, row) => total + row.count, 0)} VMs across ${rows.length} platforms.`;
	const maxCount = rows.reduce(
		(currentMax, row) => Math.max(currentMax, row.count),
		0,
	);

	return (
		<section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
			<SectionCard
				title="Platform totals"
				description="A quick capacity snapshot across the discovered estate."
			>
				{rows.length === 0 ? (
					<InlineNotice
						message="No discovered platforms are available yet."
						tone="neutral"
					/>
				) : (
					<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
						{rows.map((row) => (
							<StatCard
								key={row.platform}
								label={row.platform}
								value={row.count}
								detail={`${row.cpu} vCPU • ${formatMemory(row.memory)} GB memory`}
								emphasis="large"
							/>
						))}
					</div>
				)}
			</SectionCard>

			<SectionCard
				title="Hypervisor spread"
				description="Compare VM counts side by side before planning a migration wave."
			>
				<div className="space-y-3" aria-label={chartSummary}>
					{rows.length === 0 ? (
						<InlineNotice
							message="No platform totals are available yet."
							tone="neutral"
						/>
					) : (
						rows.map((row) => (
							<article key={row.platform} className="metric-surface">
								<div className="flex items-center justify-between gap-3">
									<p className="font-semibold text-ink">{row.platform}</p>
									<p className="text-sm text-slate-600">
										{row.count} VM{row.count === 1 ? "" : "s"}
									</p>
								</div>
								<div
									aria-hidden="true"
									className="mt-3 h-2.5 overflow-hidden rounded-full bg-slate-200"
								>
									<div
										className="h-full rounded-full bg-gradient-to-r from-steel to-ink"
										style={{
											width:
												maxCount === 0
													? "0%"
													: `${Math.max((row.count / maxCount) * 100, 8)}%`,
										}}
									/>
								</div>
								<p className="mt-2 text-xs text-slate-500">
									{row.cpu} vCPU • {formatMemory(row.memory)} GB memory
								</p>
							</article>
						))
					)}
				</div>
			</SectionCard>
		</section>
	);
}

function formatMemory(memoryMB: number): string {
	return (memoryMB / 1024).toFixed(memoryMB >= 10240 ? 0 : 1);
}
