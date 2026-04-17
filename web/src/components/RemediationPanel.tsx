import type {
	Platform,
	RecommendationReport,
	SimulationResult,
} from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatCard } from "./primitives/StatCard";
import { StatusBadge } from "./primitives/StatusBadge";

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
		<SectionCard
			title="Remediation guidance"
			description="Turn cost, policy, and drift signals into actionable next steps and quick what-if simulations."
			actions={
				<StatusBadge tone="neutral">
					{report?.recommendations.length ?? 0} recommendations
				</StatusBadge>
			}
		>
			<div className="panel-muted px-4 py-4">
				<div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
					<div>
						<p className="font-semibold text-ink">Fleet movement simulation</p>
						<p className="mt-1 text-sm leading-6 text-slate-600">
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
									className="operator-button-secondary"
									onClick={() => onSimulate(platform)}
									disabled={simulationLoading}
								>
									{simulationLoading ? "Running…" : `Simulate ${platform}`}
								</button>
							),
						)}
					</div>
				</div>

				{simulation ? (
					<div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
						<StatCard
							label="Moved"
							value={simulation.moved_vms}
							emphasis="large"
						/>
						<StatCard
							label="Current"
							value={`$${simulation.current_monthly_cost.toFixed(2)}`}
							emphasis="large"
						/>
						<StatCard
							label="Simulated"
							value={`$${simulation.simulated_monthly_cost.toFixed(2)}`}
							emphasis="large"
						/>
						<StatCard
							label="Delta"
							value={`${simulation.monthly_delta <= 0 ? "-" : "+"}$${Math.abs(simulation.monthly_delta).toFixed(2)}`}
							badge={{
								label: simulation.monthly_delta <= 0 ? "Savings" : "Increase",
								tone: simulation.monthly_delta <= 0 ? "success" : "warning",
							}}
							emphasis="large"
						/>
					</div>
				) : null}

				{error ? (
					<InlineNotice message={error} tone="danger" className="mt-4" />
				) : null}
			</div>

			<div className="mt-5 space-y-3">
				{(report?.recommendations ?? []).map((recommendation, index) => (
					<article
						key={`${recommendation.vm.id}-${recommendation.type}-${index}`}
						className="list-card"
					>
						<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div>
								<p className="font-semibold text-ink">
									{recommendation.vm.name}
								</p>
								<p className="mt-1 text-sm text-slate-500">
									{recommendation.type}
									{recommendation.target_platform
										? ` • target ${recommendation.target_platform}`
										: ""}
								</p>
							</div>
							<StatusBadge tone={severityTone(recommendation.severity)}>
								{recommendation.severity}
							</StatusBadge>
						</div>
						<p className="mt-3 text-sm leading-6 text-slate-700">
							{recommendation.summary}
						</p>
						<p className="mt-2 text-sm leading-6 text-slate-600">
							{recommendation.action}
						</p>
						{recommendation.monthly_savings &&
						recommendation.monthly_savings > 0 ? (
							<div className="mt-4">
								<StatusBadge tone="success">
									Estimated savings ${recommendation.monthly_savings.toFixed(2)}
									/mo
								</StatusBadge>
							</div>
						) : null}
					</article>
				))}

				{(report?.recommendations ?? []).length === 0 ? (
					<InlineNotice
						message="No remediation guidance is currently available."
						tone="neutral"
					/>
				) : null}
			</div>
		</SectionCard>
	);
}

function severityTone(
	severity: string,
): "neutral" | "info" | "success" | "warning" | "danger" | "accent" {
	const normalized = severity.toLowerCase();
	if (normalized.includes("critical") || normalized.includes("high")) {
		return "danger";
	}
	if (normalized.includes("warn") || normalized.includes("medium")) {
		return "warning";
	}
	return "info";
}
