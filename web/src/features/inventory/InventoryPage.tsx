import { getRouteHref } from "../../app/navigation";
import { InventoryTable } from "../../components/InventoryTable";
import { PlatformSummary } from "../../components/PlatformSummary";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { DiscoveryResult, SnapshotMeta, TenantSummary } from "../../types";
import { saveInventoryPlanningDraft } from "./inventoryPlanningDraft";
import {
  formatTimestamp,
  type InventoryAssessmentRow,
} from "./inventoryModel";
import { useInventoryAssessment } from "./useInventoryAssessment";
import { getPlatformScopeLabel } from "./workloadIdentity";
import { useInventoryWorkspace } from "./useInventoryWorkspace";
import { WorkloadDetailPanel } from "./WorkloadDetailPanel";

interface InventoryPageProps {
  inventory: DiscoveryResult | null;
  summary: TenantSummary | null;
  latestSnapshot: SnapshotMeta | null;
  refreshToken: number;
  loading: boolean;
  error: string | null;
}

export function InventoryPage({
  inventory,
  summary,
  latestSnapshot,
  refreshToken,
  loading,
  error,
}: InventoryPageProps) {
  const assessment = useInventoryAssessment({ inventory, latestSnapshot, refreshToken });
  const workspace = useInventoryWorkspace(assessment.rows);
  const planningBlockedReason = getPlanningBlockedReason(workspace.selectedRows);
  const planningNote = workspace.selectedRows.length > 0
    ? "This action creates a local planning draft and opens the migration planning workspace with the selected workloads already in scope. Viaduct still requires source and target endpoint details before preflight, plan save, and execution because the backend contract is spec-driven."
    : null;
  const selectionScope = getSelectionScopeLabel(workspace.selectedRows);
  const assetCoverage = {
    hosts: inventory?.hosts?.length ?? 0,
    clusters: inventory?.clusters?.length ?? 0,
    networks: inventory?.networks?.length ?? 0,
    datastores: inventory?.datastores?.length ?? 0,
    resourcePools: inventory?.resource_pools?.length ?? 0,
  };

  function handlePreparePlan(rowsToPlan: InventoryAssessmentRow[]) {
    if (rowsToPlan.length === 0) {
      return;
    }

    const [firstRow] = rowsToPlan;
    saveInventoryPlanningDraft({
      version: 1,
      createdAt: new Date().toISOString(),
      sourcePlatform: firstRow.vm.platform,
      workloadKeys: rowsToPlan.map((row) => row.id),
      workloads: rowsToPlan.map((row) => row.vm),
    });
    window.location.hash = getRouteHref("/migrations");
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Inventory"
        title="Fleet inventory and assessment"
        description="Operate from a workload-centric inventory surface with search, dependency context, and migration-planning preparation without losing sight of platform reality."
        badges={[
          { label: `${summary?.workload_count ?? inventory?.vms.length ?? 0} workloads`, tone: "info" },
          { label: `${workspace.summary.highRisk} high risk`, tone: workspace.summary.highRisk > 0 ? "warning" : "success" },
          { label: `${workspace.selectedIds.length} selected`, tone: workspace.selectedIds.length > 0 ? "accent" : "neutral" },
          {
            label: assessment.loading ? "Assessment refreshing" : assessment.error ? "Assessment partial" : "Assessment current",
            tone: assessment.loading ? "info" : assessment.error ? "warning" : "success",
          },
        ]}
        actions={
          <a
            href={getRouteHref("/graph")}
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
          >
            Open dependency graph
          </a>
        }
      />

      {loading && !inventory && (
        <LoadingState
          title="Loading inventory"
          message="Retrieving normalized workload inventory, dependency context, and operator assessment signals from the Viaduct API."
        />
      )}

      {!loading && error && !inventory && <ErrorState title="Inventory unavailable" message={error} />}

      {!loading && !error && !inventory && (
        <EmptyState
          title="No inventory returned"
          message="Run discovery or connect a source platform so Viaduct can populate the workload inventory surface."
        />
      )}

      {inventory && (
        <>
          <SectionCard title="Operational posture" description="Current estate shape and the derived operator assessment across the active tenant inventory.">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
              <MetricCard label="Ready" value={String(workspace.summary.ready)} tone="success" />
              <MetricCard label="Needs review" value={String(workspace.summary.needsReview)} tone="warning" />
              <MetricCard label="Blocked" value={String(workspace.summary.blocked)} tone="danger" />
              <MetricCard label="High risk" value={String(workspace.summary.highRisk)} tone="warning" />
              <MetricCard label="Selected" value={String(workspace.summary.selected)} tone="accent" />
            </div>
          </SectionCard>

          <section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
            <SectionCard title="Discovery context" description="Current inventory source and the latest baseline Viaduct is assessing.">
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <ContextCell label="Source" value={inventory.source?.trim() ? inventory.source : "Mixed latest snapshots"} />
                <ContextCell label="Platform scope" value={inventory.platform?.trim() ? inventory.platform : "Mixed platform inventory"} />
                <ContextCell label="Latest baseline" value={latestSnapshot ? formatTimestamp(latestSnapshot.discovered_at) : "No saved snapshot"} />
                <ContextCell label="Inventory errors" value={String(inventory.errors?.length ?? 0)} />
              </div>
              {inventory.errors && inventory.errors.length > 0 && (
                <div className="mt-4 space-y-2">
                  {inventory.errors.map((item) => (
                    <p key={item} className="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
                      {item}
                    </p>
                  ))}
                </div>
              )}
            </SectionCard>

            <SectionCard title="Asset coverage" description="Normalized infrastructure context shipped with the current inventory contract.">
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                <ContextCell label="Hosts" value={String(assetCoverage.hosts)} />
                <ContextCell label="Clusters" value={String(assetCoverage.clusters)} />
                <ContextCell label="Networks" value={String(assetCoverage.networks)} />
                <ContextCell label="Datastores" value={String(assetCoverage.datastores)} />
                <ContextCell label="Resource pools" value={String(assetCoverage.resourcePools)} />
                <ContextCell label="Snapshot quota free" value={String(summary?.snapshot_quota_free ?? 0)} />
              </div>
            </SectionCard>
          </section>

          <PlatformSummary inventory={inventory} />

          <SectionCard title="Assessment notes" description="Signals that shape readiness and risk in the current operator view.">
            <div className="flex flex-wrap gap-2">
              <StatusBadge tone={assessment.sources.graph ? "success" : "warning"}>
                Graph {assessment.sources.graph ? "ready" : "partial"}
              </StatusBadge>
              <StatusBadge tone={assessment.sources.policies ? "success" : "warning"}>
                Policies {assessment.sources.policies ? "ready" : "partial"}
              </StatusBadge>
              <StatusBadge tone={assessment.sources.remediation ? "success" : "warning"}>
                Remediation {assessment.sources.remediation ? "ready" : "partial"}
              </StatusBadge>
              {assessment.topSignals.length > 0 ? (
                assessment.topSignals.map((signal) => (
                  <StatusBadge key={signal.label} tone="neutral">
                    {signal.label}: {signal.count}
                  </StatusBadge>
                ))
              ) : (
                <StatusBadge tone="success">No derived risk drivers yet</StatusBadge>
              )}
            </div>
            {assessment.loading && (
              <p className="mt-4 rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
                Refreshing dependency and lifecycle assessment signals for the current workload baseline.
              </p>
            )}
            <p className="mt-4 text-sm text-slate-500">
              Readiness in this view is derived from current inventory, policy violations, remediation guidance, and graph relationships. Viaduct does not yet expose a VM-scoped preflight or activity endpoint for a cleaner per-workload readiness contract.
            </p>
            {assessment.error && (
              <p className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
                Inventory data loaded, but some assessment signals are partial: {assessment.error}
              </p>
            )}
          </SectionCard>

          <section className="grid gap-5 xl:grid-cols-[1.55fr_0.95fr]">
            <InventoryTable
              rows={workspace.filteredRows}
              totalCount={assessment.rows.length}
              filteredCount={workspace.filteredRows.length}
              selectedCount={workspace.selectedIds.length}
              hasActiveFilters={workspace.hasActiveFilters}
              loading={loading && assessment.rows.length === 0}
              refreshing={assessment.loading && assessment.rows.length > 0}
              error={error}
              availablePlatforms={workspace.availablePlatforms}
              filters={workspace.filters}
              sortKey={workspace.sortKey}
              sortDirection={workspace.sortDirection}
              activeWorkloadId={workspace.activeRow?.id ?? null}
              selectedIds={workspace.selectedIds}
              onFiltersChange={workspace.updateFilters}
              onSortChange={workspace.changeSort}
              onToggleSelection={workspace.toggleSelection}
              onToggleSelectAllVisible={workspace.toggleSelectAllVisible}
              onClearSelection={workspace.clearSelection}
              onResetFilters={workspace.resetFilters}
              onFocusWorkload={workspace.setActiveWorkloadId}
              actions={
                <>
                  <StatusBadge tone={workspace.visibleSelectedCount > 0 ? "accent" : "neutral"}>
                    {workspace.visibleSelectedCount} visible selected
                  </StatusBadge>
                  {selectionScope && <StatusBadge tone="neutral">{selectionScope}</StatusBadge>}
                  <button
                    type="button"
                    onClick={() => handlePreparePlan(workspace.selectedRows)}
                    disabled={workspace.selectedRows.length === 0 || Boolean(planningBlockedReason)}
                    className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-300"
                  >
                    Open migration plan
                  </button>
                </>
              }
            />

            <WorkloadDetailPanel
              row={workspace.activeRow}
              latestSnapshot={latestSnapshot}
              assessmentErrors={assessment.errors}
              onPreparePlan={(row) => handlePreparePlan([row])}
            />
          </section>

          {(planningBlockedReason || planningNote) && workspace.selectedRows.length > 0 && (
            <p className="rounded-2xl bg-sky-50 px-4 py-3 text-sm text-sky-900">
              {planningBlockedReason ?? planningNote}
            </p>
          )}
        </>
      )}
    </div>
  );
}

function MetricCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "neutral" | "info" | "success" | "warning" | "danger" | "accent";
}) {
  return (
        <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
        <StatusBadge tone={tone}>{label}</StatusBadge>
      </div>
      <p className="mt-3 font-display text-3xl text-ink">{value}</p>
    </div>
  );
}

function ContextCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 font-semibold text-ink">{value}</p>
    </div>
  );
}

function getPlanningBlockedReason(rows: InventoryAssessmentRow[]): string | null {
  if (rows.length === 0) {
    return null;
  }

  const platforms = new Set(rows.map((row) => row.vm.platform));
  if (platforms.size > 1) {
    return "Current migration specs are single-source and single-platform. Narrow the selection to one source platform before preparing a planning draft.";
  }

  return null;
}

function getSelectionScopeLabel(rows: InventoryAssessmentRow[]): string | null {
  if (rows.length === 0) {
    return null;
  }

  const platforms = Array.from(new Set(rows.map((row) => row.vm.platform)));
  if (platforms.length === 1) {
    return `${rows.length} workload(s) from ${getPlatformScopeLabel(platforms[0])}`;
  }

  return `${rows.length} workload(s) across ${platforms.length} platforms`;
}
