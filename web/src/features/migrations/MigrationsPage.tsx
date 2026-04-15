import { useEffect, useMemo, useState } from "react";
import { getRouteHref } from "../../app/navigation";
import { DiscoverySnapshotsPanel } from "../../components/DiscoverySnapshotsPanel";
import { MigrationHistory } from "../../components/MigrationHistory";
import { MigrationWizard } from "../../components/MigrationWizard";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { DiscoveryResult, MigrationMeta, SnapshotMeta, TenantSummary } from "../../types";
import {
  clearInventoryPlanningDraft,
  loadInventoryPlanningDraft,
  type InventoryPlanningDraft,
} from "../inventory/inventoryPlanningDraft";
import { summarizeInventoryRows } from "../inventory/inventoryModel";
import { useInventoryAssessment } from "../inventory/useInventoryAssessment";
import {
  countPersistedMigrationListStatuses,
  describeMigrationPhase,
  getPersistedMigrationListPresentation,
  getPersistedMigrationListStatus,
  getWorkflowStatusPresentation,
} from "./migrationStatus";

interface MigrationsPageProps {
  inventory: DiscoveryResult | null;
  migrations: MigrationMeta[];
  snapshots: SnapshotMeta[];
  summary: TenantSummary | null;
  latestSnapshot: SnapshotMeta | null;
  refreshToken: number;
  loading: boolean;
  migrationError?: string;
  snapshotError?: string;
  onMigrationChange: () => void | Promise<void>;
}

export function MigrationsPage({
  inventory,
  migrations,
  snapshots,
  summary,
  latestSnapshot,
  refreshToken,
  loading,
  migrationError,
  snapshotError,
  onMigrationChange,
}: MigrationsPageProps) {
  const [planningDraft, setPlanningDraft] = useState<InventoryPlanningDraft | null>(null);
  const historyError = [migrationError, snapshotError].filter(Boolean).join(" ");
  const assessment = useInventoryAssessment({ inventory, latestSnapshot, refreshToken });
  const queueCounts = useMemo(() => countPersistedMigrationListStatuses(migrations), [migrations]);

  useEffect(() => {
    setPlanningDraft(loadInventoryPlanningDraft());
  }, []);

  const planningRows = useMemo(() => {
    if (!planningDraft) {
      return [];
    }

    const selectedKeys = new Set(planningDraft.workloadKeys);
    return assessment.rows.filter((row) => selectedKeys.has(row.id));
  }, [assessment.rows, planningDraft]);

  const planningSummary = useMemo(
    () => summarizeInventoryRows(planningRows, planningRows.length),
    [planningRows],
  );
  const dependencySummary = useMemo(
    () => ({
      networks: planningRows.reduce((total, row) => total + row.dependencies.networks.length, 0),
      datastores: planningRows.reduce((total, row) => total + row.dependencies.datastores.length, 0),
      backups: planningRows.reduce((total, row) => total + row.dependencies.backups.length, 0),
    }),
    [planningRows],
  );
  const unmatchedDraftCount = Math.max(0, (planningDraft?.workloads.length ?? 0) - planningRows.length);

  function handleClearPlanningDraft() {
    clearInventoryPlanningDraft();
    setPlanningDraft(null);
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Execution"
        title="Migrations"
        description="Move from selected inventory into validated migration plans, save execution-ready plan state, and keep running jobs visible in the same operational surface."
        badges={[
          { label: `${summary?.active_migrations ?? 0} active`, tone: "accent" },
          { label: `${summary?.pending_approvals ?? 0} pending approval`, tone: "warning" },
          { label: `${migrations.length} recorded`, tone: "neutral" },
        ]}
        actions={
          <>
            <a
              href={getRouteHref("/inventory")}
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
            >
              Review inventory
            </a>
            <a
              href={getRouteHref("/reports")}
              className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
            >
              Open reports
            </a>
          </>
        }
      />

      <section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <SectionCard
          title="Planning intake"
          description="Inventory selections arrive here as a local dashboard draft. The draft guides planning, but it is not a persisted backend migration record until you save a migration plan."
          actions={
            planningDraft ? (
              <button
                type="button"
                onClick={handleClearPlanningDraft}
                className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50"
              >
                Clear draft
              </button>
            ) : undefined
          }
        >
          {planningDraft ? (
            <div className="space-y-4">
              <div className="flex flex-wrap gap-2">
                <StatusBadge tone="accent">Imported from inventory</StatusBadge>
                <StatusBadge tone="info">{planningDraft.sourcePlatform}</StatusBadge>
                <StatusBadge tone="neutral">{planningDraft.workloads.length} workload(s)</StatusBadge>
                <StatusBadge tone={planningSummary.highRisk > 0 ? "warning" : "success"}>
                  {planningSummary.highRisk} high risk
                </StatusBadge>
              </div>

              <div className="grid gap-3 md:grid-cols-4">
                <PostureMetric label="Ready" value={planningSummary.ready} tone="success" />
                <PostureMetric label="Needs review" value={planningSummary.needsReview} tone="warning" />
                <PostureMetric label="Blocked" value={planningSummary.blocked} tone="danger" />
                <PostureMetric label="High risk" value={planningSummary.highRisk} tone="warning" />
              </div>

              <div className="grid gap-3 md:grid-cols-3">
                <ContextCell label="Network relationships" value={`${dependencySummary.networks}`} />
                <ContextCell label="Storage relationships" value={`${dependencySummary.datastores}`} />
                <ContextCell label="Backup relationships" value={`${dependencySummary.backups}`} />
              </div>

              <p className="rounded-2xl bg-sky-50 px-4 py-3 text-sm text-sky-950">
                The workload list came from the inventory route. Viaduct still requires source and target endpoint details plus a saved plan state before execution can start.
              </p>

              {assessment.loading && (
                <p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
                  Refreshing dependency and lifecycle signals for the imported workloads.
                </p>
              )}

              {assessment.error && (
                <p className="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
                  Imported workloads are visible, but some readiness context is partial: {assessment.error}
                </p>
              )}

              {unmatchedDraftCount > 0 && (
                <p className="rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-900">
                  {unmatchedDraftCount} imported workload(s) were not matched back to the current inventory assessment. The draft is still available, but risk and dependency context may be incomplete until the latest inventory lines up again.
                </p>
              )}
            </div>
          ) : (
        <div className="rounded-2xl border border-dashed border-slate-300 px-5 py-6 text-sm text-slate-600">
              <p className="font-semibold text-ink">No inventory draft is active.</p>
              <p className="mt-2">
                Start from the inventory route when you want a selected workload set to carry directly into migration planning.
              </p>
            </div>
          )}
        </SectionCard>

        <SectionCard
          title="Execution posture"
          description="Live migration activity from the persisted store. Saved plans stay in plan phase until you explicitly execute them."
        >
          <div className="grid gap-3 md:grid-cols-2">
            <PostureMetric label="Saved plans" value={queueCounts.planned} tone="neutral" />
            <PostureMetric label="Running" value={queueCounts.running} tone="accent" />
            <PostureMetric label="Pending approvals" value={summary?.pending_approvals ?? 0} tone="warning" />
            <PostureMetric label="Failed" value={queueCounts.failed} tone={queueCounts.failed > 0 ? "danger" : "neutral"} />
          </div>

          <div className="mt-5 space-y-3">
            {migrations.slice(0, 3).map((migration) => {
              const status = getPersistedMigrationListPresentation(getPersistedMigrationListStatus(migration.phase));

              return (
                <article key={migration.id} className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p className="font-semibold text-ink">{migration.spec_name}</p>
                      <p className="text-slate-500">{migration.id}</p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
                      <StatusBadge tone="neutral">{describeMigrationPhase(migration.phase)}</StatusBadge>
                    </div>
                  </div>
                  <p className="mt-3 text-slate-500">
                    Started {new Date(migration.started_at).toLocaleString()} and updated {new Date(migration.updated_at).toLocaleString()}.
                  </p>
                </article>
              );
            })}

            {migrations.length === 0 && !loading && !migrationError && (
              <p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
                No persisted migration runs have been recorded yet.
              </p>
            )}
          </div>
        </SectionCard>
      </section>

      <SectionCard
        title="Operational states"
        description="Draft, ready, warning, and blocked are detailed planning states inside the workspace. Persisted migration history only exposes plan and phase metadata from the list API."
      >
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {(["draft", "ready", "warning", "blocked", "running", "failed", "completed"] as const).map((statusKey) => {
            const status = getWorkflowStatusPresentation(statusKey);

            return (
              <div key={statusKey} className="rounded-2xl bg-slate-50 px-4 py-4">
                <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
                <p className="mt-3 text-sm text-slate-600">{status.description}</p>
              </div>
            );
          })}
        </div>
      </SectionCard>

      <MigrationWizard
        planningDraft={planningDraft}
        onPlanningDraftCleared={handleClearPlanningDraft}
        onMigrationChange={onMigrationChange}
      />

      {loading && migrations.length === 0 && snapshots.length === 0 && !planningDraft && (
        <LoadingState
          title="Loading migration operations"
          message="Retrieving migration history and saved discovery baselines so operators can validate the current runbook context."
        />
      )}

      {historyError && migrations.length === 0 && snapshots.length === 0 && !loading && (
        <ErrorState title="Migration history unavailable" message={historyError} />
      )}

      <section className="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <MigrationHistory migrations={migrations.slice(0, 6)} loading={loading} error={migrationError} />
        <DiscoverySnapshotsPanel snapshots={snapshots.slice(0, 6)} loading={loading} error={snapshotError} />
      </section>
    </div>
  );
}

function PostureMetric({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
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
      <p className="mt-2 text-sm font-semibold text-ink">{value}</p>
    </div>
  );
}
