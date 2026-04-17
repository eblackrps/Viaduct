import { getRouteHref } from "../../app/navigation";
import { MigrationHistory } from "../../components/MigrationHistory";
import { PlatformSummary } from "../../components/PlatformSummary";
import { EmptyState } from "../../components/primitives/EmptyState";
import { InlineNotice } from "../../components/primitives/InlineNotice";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatCard } from "../../components/primitives/StatCard";
import {
	StatusBadge,
	type StatusTone,
} from "../../components/primitives/StatusBadge";
import type {
	DiscoveryResult,
	MigrationMeta,
	SnapshotMeta,
	TenantSummary,
} from "../../types";
import { useLifecycleSignals } from "../lifecycle/useLifecycleSignals";

interface DashboardPageProps {
	inventory: DiscoveryResult | null;
	migrations: MigrationMeta[];
	migrationError?: string;
	summary: TenantSummary | null;
	latestSnapshot: SnapshotMeta | null;
	loading: boolean;
	refreshToken: number;
}

export function DashboardPage({
	inventory,
	migrations,
	migrationError,
	summary,
	latestSnapshot,
	loading,
	refreshToken,
}: DashboardPageProps) {
	const {
		drift,
		policies,
		remediation,
		loading: postureLoading,
		errors,
	} = useLifecycleSignals({
		baselineId: latestSnapshot?.id ?? null,
		refreshToken,
		includePolicies: true,
		includeDrift: true,
		includeRemediation: true,
	});
	const platformCounts =
		summary?.platform_counts ?? summarizePlatforms(inventory);
	const recentMigrations = migrations.slice(0, 4);
	const postureError = [errors.policies, errors.drift, errors.remediation]
		.filter(Boolean)
		.join(" ");
	const metricCards = [
		{
			label: "Workloads",
			value: summary?.workload_count ?? inventory?.vms.length ?? 0,
			tone: "info" as StatusTone,
			badgeLabel: "Inventory",
			detail: "Normalized VMs currently visible to the operator surface.",
		},
		{
			label: "Active migrations",
			value: summary?.active_migrations ?? countActiveMigrations(migrations),
			tone: "accent" as StatusTone,
			badgeLabel: "In flight",
			detail: "Persisted migrations still in plan or execution phases.",
		},
		{
			label: "Pending approvals",
			value: summary?.pending_approvals ?? 0,
			tone: "warning" as StatusTone,
			badgeLabel: "Gate",
			detail: "Saved plans waiting for an explicit approval gate to clear.",
		},
		{
			label: "Recommendations",
			value:
				summary?.recommendation_count ??
				remediation?.recommendations.length ??
				0,
			tone: "success" as StatusTone,
			badgeLabel: "Lifecycle",
			detail: "Lifecycle moves currently suggested by cost and policy posture.",
		},
	];

	return (
		<div className="space-y-6">
			<PageHeader
				eyebrow="Overview"
				title="Operational dashboard"
				description="Monitor tenant posture, current execution state, and workload distribution before drilling into specific migration or governance tasks."
				badges={[
					{
						label: `${migrations.length} migrations recorded`,
						tone: "neutral",
					},
					{
						label: `${Object.keys(platformCounts).length} platforms observed`,
						tone: "info",
					},
				]}
				actions={
					<a
						href={getRouteHref("/migrations")}
						className="operator-button-secondary"
					>
						Open migrations
					</a>
				}
			/>

			{loading && !inventory && !summary ? (
				<LoadingState
					title="Loading dashboard"
					message="Collecting inventory, migration history, lifecycle posture, and tenant summary data from the Viaduct operator API."
				/>
			) : null}

			{!loading && !inventory && !summary ? (
				<EmptyState
					title="No operator data available"
					message="Connect the dashboard to a tenant inventory source or run a discovery cycle to populate the operational dashboard."
				/>
			) : null}

			{inventory || summary ? (
				<>
					<section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
						{metricCards.map((card) => (
							<StatCard
								key={card.label}
								label={card.label}
								value={card.value}
								detail={card.detail}
								badge={{ label: card.badgeLabel, tone: card.tone }}
								emphasis="large"
							/>
						))}
					</section>

					{inventory ? (
						<PlatformSummary inventory={inventory} />
					) : (
						<EmptyState
							title="Inventory snapshot unavailable"
							message="The tenant summary loaded, but inventory data is not currently available to populate platform distribution."
						/>
					)}

					<section className="grid gap-5 xl:grid-cols-[1.15fr_0.85fr]">
						<MigrationHistory
							migrations={recentMigrations}
							error={migrationError}
						/>

						<SectionCard
							title="Operational posture"
							description="Current cross-domain signals that affect planning, execution, and remediation."
						>
							<div className="space-y-4">
								{postureLoading && !policies && !drift && !remediation ? (
									<InlineNotice
										message="Loading policy, drift, and remediation posture for this tenant."
										tone="neutral"
									/>
								) : null}
								{postureError ? (
									<InlineNotice message={postureError} tone="warning" />
								) : null}
								<div className="grid gap-3 md:grid-cols-3">
									<SignalRow
										label="Policy posture"
										value={`${policies?.non_compliant_vms ?? 0} flagged / ${policies?.compliant_vms ?? 0} compliant`}
										badgeTone={
											(policies?.non_compliant_vms ?? 0) > 0
												? "warning"
												: "success"
										}
									/>
									<SignalRow
										label="Drift posture"
										value={`${drift?.events.length ?? 0} events / ${drift?.modified_vms ?? 0} modified`}
										badgeTone={
											(drift?.events.length ?? 0) > 0 ? "warning" : "success"
										}
									/>
									<SignalRow
										label="Latest snapshot"
										value={
											latestSnapshot
												? `${latestSnapshot.source} • ${new Date(latestSnapshot.discovered_at).toLocaleString()}`
												: "No snapshot recorded"
										}
										badgeTone={latestSnapshot ? "info" : "neutral"}
									/>
								</div>
								<div>
									<p className="operator-kicker">Platform spread</p>
									<div className="mt-3 flex flex-wrap gap-2">
										{Object.entries(platformCounts).map(([platform, count]) => (
											<StatusBadge key={platform} tone="neutral">
												{platform}: {count}
											</StatusBadge>
										))}
										{Object.keys(platformCounts).length === 0 ? (
											<StatusBadge tone="neutral">
												No platform counts yet
											</StatusBadge>
										) : null}
									</div>
								</div>
								<div className="panel-muted px-4 py-4">
									<p className="font-semibold text-ink">Dependency analysis</p>
									<p className="mt-2 text-sm leading-6 text-slate-600">
										Open the dedicated analysis view to inspect workload
										relationships across storage, network, and backup
										dependencies.
									</p>
									<a
										href={getRouteHref("/graph")}
										className="operator-button-secondary mt-4"
									>
										Open dependency graph
									</a>
								</div>
							</div>
						</SectionCard>
					</section>
				</>
			) : null}
		</div>
	);
}

const toneLabel: Record<StatusTone, string> = {
	success: "Healthy",
	warning: "Attention",
	danger: "Critical",
	info: "Active",
	neutral: "None",
	accent: "Active",
};

function SignalRow({
	label,
	value,
	badgeTone,
}: {
	label: string;
	value: string;
	badgeTone: StatusTone;
}) {
	return (
		<StatCard
			label={label}
			value={value}
			badge={{ label: toneLabel[badgeTone], tone: badgeTone }}
		/>
	);
}

function summarizePlatforms(
	inventory: DiscoveryResult | null,
): Record<string, number> {
	return (inventory?.vms ?? []).reduce<Record<string, number>>(
		(accumulator, vm) => {
			accumulator[vm.platform] = (accumulator[vm.platform] ?? 0) + 1;
			return accumulator;
		},
		{},
	);
}

function countActiveMigrations(migrations: MigrationMeta[]): number {
	return migrations.filter(
		(migration) =>
			!["complete", "failed", "rolled_back"].includes(migration.phase),
	).length;
}
