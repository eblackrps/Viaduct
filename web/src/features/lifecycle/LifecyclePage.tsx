import { getRouteHref } from "../../app/navigation";
import { CostComparison } from "../../components/CostComparison";
import { RemediationPanel } from "../../components/RemediationPanel";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
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
						<a href={getRouteHref("/policy")} className="operator-button-secondary">
							Open policy
						</a>
						<a href={getRouteHref("/drift")} className="operator-button-secondary">
							Open drift
						</a>
					</>
				}
			/>

			{(overviewLoading || loading) && !hasLifecycleData ? (
				<LoadingState
					title="Loading lifecycle signals"
					message="Collecting remediation guidance and platform cost comparisons for the current tenant baseline."
				/>
			) : null}

			{!overviewLoading && !loading && !hasLifecycleData
				? lifecycleError
					? <ErrorState title="Lifecycle data unavailable" message={lifecycleError} />
					: (
							<EmptyState
								title="No lifecycle data available"
								message="Lifecycle guidance will appear here when Viaduct has current cost profiles and remediation recommendations to evaluate."
							/>
						)
				: null}

			{hasLifecycleData ? (
				<>
					<SectionCard
						title="Cross-domain handoff"
						description="Lifecycle decisions stay connected to separate policy and drift views so operators can review enforcement and baseline change before acting."
					>
						<div className="grid gap-3 md:grid-cols-2">
							<InlineNotice
								title="Policy review"
								message="Use the dedicated Policy page to inspect rule-level violations and enforcement posture on the current inventory."
								tone="neutral"
							/>
							<InlineNotice
								title="Drift review"
								message="Use the Drift page to confirm baseline changes and unexpected workload movement before scheduling execution."
								tone="neutral"
							/>
						</div>
					</SectionCard>

					<RemediationPanel
						report={remediation}
						simulation={simulation}
						onSimulate={simulate}
						simulationLoading={simulationLoading}
						error={recommendationError}
					/>
					{errors.costs ? (
						<InlineNotice message={errors.costs} tone="danger" />
					) : null}
					<CostComparison comparisons={costs} />
				</>
			) : null}
		</div>
	);
}
