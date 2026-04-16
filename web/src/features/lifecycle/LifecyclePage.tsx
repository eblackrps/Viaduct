import { getRouteHref } from "../../app/navigation";
import { CostComparison } from "../../components/CostComparison";
import { RemediationPanel } from "../../components/RemediationPanel";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import type { SnapshotMeta, TenantSummary } from "../../types";
import { useLifecycleSignals } from "./useLifecycleSignals";

interface LifecyclePageProps {
	summary: TenantSummary | null;
	latestSnapshot: SnapshotMeta | null;
	overviewLoading: boolean;
	refreshToken: number;
}

export function LifecyclePage({
	summary,
	latestSnapshot,
	overviewLoading,
	refreshToken,
}: LifecyclePageProps) {
	const {
		costs,
		remediation,
		simulation,
		loading,
		simulationLoading,
		errors,
		simulate,
	} = useLifecycleSignals({
		baselineId: latestSnapshot?.id ?? null,
		refreshToken,
		includeCosts: true,
		includeRemediation: true,
	});
	const hasLifecycleData = costs.length > 0 || remediation !== null;
	const lifecycleError = [errors.costs, errors.remediation, errors.simulation]
		.filter(Boolean)
		.join(" ");
	const recommendationError = [errors.remediation, errors.simulation]
		.filter(Boolean)
		.join(" ");

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Lifecycle"
				title="Lifecycle optimization"
				description="Use Viaduct remediation guidance and cost comparisons to prioritize fleet movement before committing a migration wave."
				badges={[
					{
						label: `${summary?.recommendation_count ?? remediation?.recommendations.length ?? 0} recommendations`,
						tone: "success",
					},
					{ label: `${costs.length} cost models`, tone: "info" },
				]}
				actions={
					<>
						<a
							href={getRouteHref("/policy")}
							className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
						>
							Open policy
						</a>
						<a
							href={getRouteHref("/drift")}
							className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
						>
							Open drift
						</a>
					</>
				}
			/>

			{(overviewLoading || loading) && !hasLifecycleData && (
				<LoadingState
					title="Loading lifecycle signals"
					message="Collecting remediation guidance and platform cost comparisons for the current tenant baseline."
				/>
			)}

			{!overviewLoading &&
				!loading &&
				!hasLifecycleData &&
				(lifecycleError ? (
					<ErrorState
						title="Lifecycle data unavailable"
						message={lifecycleError}
					/>
				) : (
					<EmptyState
						title="No lifecycle data available"
						message="Lifecycle guidance will appear here when Viaduct has current cost profiles and remediation recommendations to evaluate."
					/>
				))}

			{hasLifecycleData && (
				<>
					<SectionCard
						title="Cross-domain handoff"
						description="Lifecycle decisions stay connected to separate policy and drift views so operators can review enforcement and baseline change before acting."
					>
						<div className="grid gap-3 md:grid-cols-2">
							<div className="rounded-2xl bg-slate-50 px-4 py-4">
								<p className="font-semibold text-ink">Policy review</p>
								<p className="mt-2 text-sm text-slate-500">
									Use the dedicated Policy page to inspect rule-level violations
									and enforcement posture on the current inventory.
								</p>
							</div>
							<div className="rounded-2xl bg-slate-50 px-4 py-4">
								<p className="font-semibold text-ink">Drift review</p>
								<p className="mt-2 text-sm text-slate-500">
									Use the Drift page to confirm baseline changes and unexpected
									workload movement before scheduling execution.
								</p>
							</div>
						</div>
					</SectionCard>

					<RemediationPanel
						report={remediation}
						simulation={simulation}
						onSimulate={simulate}
						simulationLoading={simulationLoading}
						error={recommendationError}
					/>
					{errors.costs && (
						<p className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
							{errors.costs}
						</p>
					)}
					<CostComparison comparisons={costs} />
				</>
			)}
		</div>
	);
}
