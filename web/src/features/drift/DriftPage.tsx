import { DriftTimeline } from "../../components/DriftTimeline";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import type { SnapshotMeta } from "../../types";
import { useLifecycleSignals } from "../lifecycle/useLifecycleSignals";

interface DriftPageProps {
	latestSnapshot: SnapshotMeta | null;
	overviewLoading: boolean;
	refreshToken: number;
}

export function DriftPage({
	latestSnapshot,
	overviewLoading,
	refreshToken,
}: DriftPageProps) {
	const {
		drift: report,
		loading,
		errors,
	} = useLifecycleSignals({
		baselineId: latestSnapshot?.id ?? null,
		refreshToken,
		includeDrift: true,
	});
	const error = errors.drift;
	const showLoading = (overviewLoading || loading) && !report;
	const showEmpty = !overviewLoading && !loading && !report && !latestSnapshot;

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Drift"
				title="Drift monitoring"
				description="Review baseline divergence, modified workloads, and policy-driven drift events before approving the next change window."
				badges={[
					{
						label: `${report?.events.length ?? 0} events`,
						tone: (report?.events.length ?? 0) > 0 ? "warning" : "success",
					},
					{
						label: latestSnapshot ? "Baseline available" : "No baseline",
						tone: latestSnapshot ? "info" : "neutral",
					},
				]}
			/>

			{showLoading ? (
				<LoadingState
					title="Loading drift results"
					message="Resolving the latest saved snapshot and comparing it against current inventory to detect baseline movement."
				/>
			) : null}

			{showEmpty ? (
				<EmptyState
					title="No baseline snapshot available"
					message="Save a discovery snapshot before using the drift view so Viaduct has a baseline to compare against."
				/>
			) : null}

			{error && !showLoading && !report && latestSnapshot ? (
				<ErrorState title="Drift comparison unavailable" message={error} />
			) : null}

			{(report || latestSnapshot) && !showLoading ? (
				<>
					<SectionCard
						title="Baseline context"
						description="Current baseline metadata and comparison scope."
					>
						<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
							<StatCard
								label="Baseline"
								value={report?.baseline.id ?? latestSnapshot?.id ?? "Unavailable"}
							/>
							<StatCard
								label="Current sample"
								value={
									report?.current.discovered_at
										? new Date(report.current.discovered_at).toLocaleString()
										: latestSnapshot
											? new Date(latestSnapshot.discovered_at).toLocaleString()
											: "Unavailable"
								}
							/>
							<StatCard
								label="Modified VMs"
								value={String(report?.modified_vms ?? 0)}
							/>
							<StatCard
								label="Policy drift"
								value={String(report?.policy_drifts ?? 0)}
							/>
						</div>
					</SectionCard>

					{report ? (
						<DriftTimeline report={report} />
					) : (
						<EmptyState
							title="Drift report unavailable"
							message="A snapshot exists, but Viaduct did not return a drift comparison payload for the current baseline."
						/>
					)}
				</>
			) : null}
		</div>
	);
}
