import type {
	Platform,
	RecommendationReport,
	SimulationResult,
} from "../types";

interface RemediationPanelProps {
	report: RecommendationReport | null;
	simulation: SimulationResult | null;
	onSimulate: (targetPlatform: Platform) => void;
	simulationLoading?: boolean;
	error?: string | null;
}

export function RemediationPanel({
	report,
	simulation,
	onSimulate,
	simulationLoading,
	error,
}: RemediationPanelProps) {
	return (
		<section className="panel p-5">
			<div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
				<div>
					<p className="font-display text-2xl text-ink">Remediation Guidance</p>
					<p className="text-sm text-slate-500">
						Turn cost, policy, and drift signals into actionable next steps and
						quick what-if simulations.
					</p>
				</div>
				<p className="text-sm font-semibold text-slate-500">
					{report?.recommendations.length ?? 0} recommendations
				</p>
			</div>

			<div className="mt-5 rounded-2xl bg-slate-50 p-4">
				<div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
					<div>
						<p className="font-semibold text-ink">Fleet Movement Simulation</p>
						<p className="text-sm text-slate-500">
							Simulate moving the current fleet to a target platform before
							scheduling the next migration wave.
						</p>
					</div>
					<div className="flex flex-wrap gap-2">
						{(["proxmox", "vmware", "hyperv", "kvm", "nutanix"] as const).map(
							(platform) => (
								<button
									key={platform}
									type="button"
									className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700"
									onClick={() => onSimulate(platform)}
									disabled={simulationLoading}
								>
									{simulationLoading ? "Running…" : `Simulate ${platform}`}
								</button>
							),
						)}
					</div>
				</div>

				{simulation && (
					<div className="mt-4 grid gap-3 md:grid-cols-4">
						<div className="rounded-2xl bg-white px-4 py-3">
							<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
								Moved
							</p>
							<p className="mt-2 font-display text-2xl text-ink">
								{simulation.moved_vms}
							</p>
						</div>
						<div className="rounded-2xl bg-white px-4 py-3">
							<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
								Current
							</p>
							<p className="mt-2 font-display text-2xl text-ink">
								${simulation.current_monthly_cost.toFixed(2)}
							</p>
						</div>
						<div className="rounded-2xl bg-white px-4 py-3">
							<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
								Simulated
							</p>
							<p className="mt-2 font-display text-2xl text-ink">
								${simulation.simulated_monthly_cost.toFixed(2)}
							</p>
						</div>
						<div className="rounded-2xl bg-white px-4 py-3">
							<p className="text-xs uppercase tracking-[0.22em] text-slate-500">
								Delta
							</p>
							<p
								className={`mt-2 font-display text-2xl ${simulation.monthly_delta <= 0 ? "text-emerald-700" : "text-amber-700"}`}
							>
								{simulation.monthly_delta <= 0 ? "-" : "+"}$
								{Math.abs(simulation.monthly_delta).toFixed(2)}
							</p>
						</div>
					</div>
				)}
				{error && (
					<p className="mt-4 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
						{error}
					</p>
				)}
			</div>

			<div className="mt-5 space-y-3">
				{(report?.recommendations ?? []).map((recommendation, index) => (
					<article
						key={`${recommendation.vm.id}-${recommendation.type}-${index}`}
						className="rounded-2xl border border-slate-200 bg-white px-4 py-4"
					>
						<div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
							<div>
								<p className="font-semibold text-ink">
									{recommendation.vm.name}
								</p>
								<p className="text-sm text-slate-500">
									{recommendation.type}
									{recommendation.target_platform
										? ` · target ${recommendation.target_platform}`
										: ""}
								</p>
							</div>
							<span className="rounded-full bg-ink px-3 py-1 text-xs font-semibold text-white">
								{recommendation.severity}
							</span>
						</div>
						<p className="mt-3 text-sm text-slate-700">
							{recommendation.summary}
						</p>
						<p className="mt-2 text-sm text-slate-500">
							{recommendation.action}
						</p>
						{recommendation.monthly_savings &&
							recommendation.monthly_savings > 0 && (
								<p className="mt-2 text-sm font-semibold text-emerald-700">
									Estimated savings: $
									{recommendation.monthly_savings.toFixed(2)}/mo
								</p>
							)}
					</article>
				))}

				{(report?.recommendations ?? []).length === 0 && (
					<p className="rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
						No remediation guidance is currently available.
					</p>
				)}
			</div>
		</section>
	);
}
