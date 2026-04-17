import { PolicyDashboard } from "../../components/PolicyDashboard";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import { useLifecycleSignals } from "../lifecycle/useLifecycleSignals";

interface PolicyPageProps {
	refreshToken: number;
}

export function PolicyPage({ refreshToken }: PolicyPageProps) {
	const {
		policies: report,
		loading,
		errors,
	} = useLifecycleSignals({ refreshToken, includePolicies: true });
	const error = errors.policies;

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Policy"
				title="Policy controls"
				description="Inspect evaluated rules, rule-level violations, and the current enforcement posture across discovered workloads."
				badges={[
					{
						label: `${report?.policies.length ?? 0} policies`,
						tone: "neutral",
					},
					{
						label: `${report?.violations.length ?? 0} violations`,
						tone: (report?.violations.length ?? 0) > 0 ? "warning" : "success",
					},
				]}
			/>

			{loading && !report ? (
				<LoadingState
					title="Loading policy results"
					message="Evaluating lifecycle policies and loading the latest compliance posture for the current tenant."
				/>
			) : null}

			{!loading && !report
				? error
					? <ErrorState title="Policy evaluation unavailable" message={error} />
					: (
							<EmptyState
								title="No policy report available"
								message="The policy engine has not returned an evaluation result for the current tenant yet."
							/>
						)
				: null}

			{report ? (
				<>
					<SectionCard
						title="Evaluation context"
						description="High-level policy execution metadata for the current report."
					>
						<div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
							<StatCard label="Policies" value={report.policies.length} />
							<StatCard label="Violations" value={report.violations.length} />
							<StatCard label="Waived" value={report.waived_violations} />
							<StatCard
								label="Evaluated"
								value={new Date(report.evaluated_at).toLocaleString()}
							/>
						</div>
					</SectionCard>
					<PolicyDashboard report={report} />
				</>
			) : null}
		</div>
	);
}
