import { getRouteHref } from "../../app/navigation";
import { MigrationHistory } from "../../components/MigrationHistory";
import { PlatformSummary } from "../../components/PlatformSummary";
import { EmptyState } from "../../components/primitives/EmptyState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge, type StatusTone } from "../../components/primitives/StatusBadge";
import type { DiscoveryResult, MigrationMeta, SnapshotMeta, TenantSummary } from "../../types";
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
  const { drift, policies, remediation, loading: postureLoading, errors } = useLifecycleSignals({
    baselineId: latestSnapshot?.id ?? null,
    refreshToken,
    includePolicies: true,
    includeDrift: true,
    includeRemediation: true,
  });
  const platformCounts = summary?.platform_counts ?? summarizePlatforms(inventory);
  const recentMigrations = migrations.slice(0, 4);
  const postureError = [errors.policies, errors.drift, errors.remediation].filter(Boolean).join(" ");
  const metricCards = [
    { label: "Workloads", value: summary?.workload_count ?? inventory?.vms.length ?? 0, tone: "info" as StatusTone },
    { label: "Active migrations", value: summary?.active_migrations ?? countActiveMigrations(migrations), tone: "accent" as StatusTone },
    { label: "Pending approvals", value: summary?.pending_approvals ?? 0, tone: "warning" as StatusTone },
    { label: "Recommendations", value: summary?.recommendation_count ?? remediation?.recommendations.length ?? 0, tone: "success" as StatusTone },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Overview"
        title="Operational dashboard"
        description="Monitor tenant posture, current execution state, and workload distribution before drilling into specific migration or governance tasks."
        badges={[
          { label: `${migrations.length} migrations recorded`, tone: "neutral" },
          { label: `${Object.keys(platformCounts).length} platforms observed`, tone: "info" },
        ]}
        actions={
          <a
            href={getRouteHref("/migrations")}
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
          >
            Open migrations
          </a>
        }
      />

      {loading && !inventory && !summary && (
        <LoadingState
          title="Loading dashboard"
          message="Collecting inventory, migration history, lifecycle posture, and tenant summary data from the Viaduct operator API."
        />
      )}

      {!loading && !inventory && !summary && (
        <EmptyState
          title="No operator data available"
          message="Connect the dashboard to a tenant inventory source or run a discovery cycle to populate the operational dashboard."
        />
      )}

      {(inventory || summary) && (
        <>
          <section className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
            {metricCards.map((card) => (
              <SectionCard key={card.label} className="p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-500">{card.label}</p>
                <p className="mt-3 font-display text-3xl text-ink">{card.value}</p>
                <div className="mt-3">
                  <StatusBadge tone={card.tone}>{card.label}</StatusBadge>
                </div>
              </SectionCard>
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
            <MigrationHistory migrations={recentMigrations} error={migrationError} />

            <SectionCard
              title="Operational posture"
              description="Current cross-domain signals that affect planning, execution, and remediation."
            >
              <div className="space-y-4 text-sm text-slate-600">
                {postureLoading && !policies && !drift && !remediation && (
                  <p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-500">
                    Loading policy, drift, and remediation posture for this tenant.
                  </p>
                )}
                {postureError && (
                  <p className="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800">{postureError}</p>
                )}
                <SignalRow
                  label="Policy posture"
                  value={`${policies?.non_compliant_vms ?? 0} flagged / ${policies?.compliant_vms ?? 0} compliant`}
                  badgeTone={(policies?.non_compliant_vms ?? 0) > 0 ? "warning" : "success"}
                />
                <SignalRow
                  label="Drift posture"
                  value={`${drift?.events.length ?? 0} events / ${drift?.modified_vms ?? 0} modified`}
                  badgeTone={(drift?.events.length ?? 0) > 0 ? "warning" : "success"}
                />
                <SignalRow
                  label="Latest snapshot"
                  value={latestSnapshot ? `${latestSnapshot.source} • ${new Date(latestSnapshot.discovered_at).toLocaleString()}` : "No snapshot recorded"}
                  badgeTone={latestSnapshot ? "info" : "neutral"}
                />
                <div>
                  <p className="text-xs uppercase tracking-[0.18em] text-slate-500">Platform spread</p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    {Object.entries(platformCounts).map(([platform, count]) => (
                      <StatusBadge key={platform} tone="neutral">
                        {platform}: {count}
                      </StatusBadge>
                    ))}
                    {Object.keys(platformCounts).length === 0 && <StatusBadge tone="neutral">No platform counts yet</StatusBadge>}
                  </div>
                </div>
                <div className="rounded-2xl bg-slate-50 px-4 py-4">
                  <p className="font-semibold text-ink">Dependency analysis</p>
                  <p className="mt-2 text-sm text-slate-500">
                    Open the dedicated analysis view to inspect workload relationships across storage, network, and backup dependencies.
                  </p>
                  <a
                    href={getRouteHref("/graph")}
                    className="mt-3 inline-flex rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
                  >
                    Open dependency graph
                  </a>
                </div>
              </div>
            </SectionCard>
          </section>
        </>
      )}
    </div>
  );
}

function SignalRow({ label, value, badgeTone }: { label: string; value: string; badgeTone: StatusTone }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <div className="flex items-center justify-between gap-3">
        <p className="font-semibold text-ink">{label}</p>
        <StatusBadge tone={badgeTone}>{label}</StatusBadge>
      </div>
      <p className="mt-2 text-slate-500">{value}</p>
    </div>
  );
}

function summarizePlatforms(inventory: DiscoveryResult | null): Record<string, number> {
  return (inventory?.vms ?? []).reduce<Record<string, number>>((accumulator, vm) => {
    accumulator[vm.platform] = (accumulator[vm.platform] ?? 0) + 1;
    return accumulator;
  }, {});
}

function countActiveMigrations(migrations: MigrationMeta[]): number {
  return migrations.filter((migration) => !["complete", "failed", "rolled_back"].includes(migration.phase)).length;
}
