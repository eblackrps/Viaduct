import { useEffect, useMemo, useState } from "react";
import {
  createWorkspace,
  createWorkspaceJob,
  describeError,
  exportWorkspaceReport,
  getSnapshot,
  getWorkspace,
  listWorkspaceJobs,
  listWorkspaces,
  updateWorkspace,
  type ErrorDisplay,
} from "../../api";
import { DependencyGraph } from "../../components/DependencyGraph";
import { InventoryTable } from "../../components/InventoryTable";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type {
  DependencyGraph as DependencyGraphModel,
  DiscoveryResult,
  Platform,
  PilotWorkspace,
  WorkspaceJob,
  WorkspaceJobType,
  WorkspaceNote,
  WorkspaceSnapshot,
} from "../../types";
import { buildInventoryAssessmentRows } from "../inventory/inventoryModel";
import { useInventoryWorkspace } from "../inventory/useInventoryWorkspace";
import { WorkloadDetailPanel } from "../inventory/WorkloadDetailPanel";

interface WorkspacePageState {
  workspaces: PilotWorkspace[];
  selectedWorkspace: PilotWorkspace | null;
  jobs: WorkspaceJob[];
  inventory: DiscoveryResult | null;
  loading: boolean;
  refreshing: boolean;
  error: ErrorDisplay | null;
  actionError: ErrorDisplay | null;
}

export function WorkspacePage() {
  const [state, setState] = useState<WorkspacePageState>({
    workspaces: [],
    selectedWorkspace: null,
    jobs: [],
    inventory: null,
    loading: true,
    refreshing: false,
    error: null,
    actionError: null,
  });
  const [selectedWorkspaceID, setSelectedWorkspaceID] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: "Examples Lab Assessment",
    description: "Assessment-to-pilot workspace",
    sourceName: "Lab KVM",
    sourcePlatform: "kvm",
    sourceAddress: "examples/lab/kvm",
    sourceCredentialRef: "lab-kvm",
    targetPlatform: "proxmox",
    targetAddress: "https://pilot-proxmox.local:8006/api2/json",
    defaultHost: "pve-node-01",
    defaultStorage: "local-lvm",
    defaultNetwork: "vmbr0",
  });
  const [noteDraft, setNoteDraft] = useState("");
  const [settingsDraft, setSettingsDraft] = useState<Pick<PilotWorkspace, "target_assumptions" | "plan_settings">>({
    target_assumptions: {},
    plan_settings: {},
  });
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  async function refreshWorkspaces(preferredWorkspaceID?: string) {
    setState((current) => ({ ...current, refreshing: !current.loading }));
    const workspaceResult = await Promise.allSettled([listWorkspaces()]);
    const result = workspaceResult[0];
    if (result.status === "rejected") {
      setState((current) => ({
        ...current,
        workspaces: [],
        selectedWorkspace: null,
        jobs: [],
        inventory: null,
        loading: false,
        refreshing: false,
        error: describeError(result.reason, {
          scope: "pilot workspaces",
          fallback: "Unable to load pilot workspaces.",
        }),
      }));
      return;
    }

    const workspaces = result.value;
    const nextWorkspaceID = preferredWorkspaceID ?? selectedWorkspaceID ?? workspaces[0]?.id ?? null;
    setSelectedWorkspaceID(nextWorkspaceID);
    setState((current) => ({
      ...current,
      workspaces,
      loading: false,
      refreshing: false,
      error: null,
    }));
  }

  async function refreshWorkspaceDetail(workspaceID: string) {
    setState((current) => ({ ...current, refreshing: true }));
    const [workspaceResult, jobsResult] = await Promise.allSettled([getWorkspace(workspaceID), listWorkspaceJobs(workspaceID)]);
    if (workspaceResult.status === "rejected") {
      setState((current) => ({
        ...current,
        selectedWorkspace: null,
        jobs: [],
        inventory: null,
        refreshing: false,
        error: describeError(workspaceResult.reason, {
          scope: "workspace state",
          fallback: "Unable to load workspace state.",
        }),
      }));
      return;
    }

    const workspace = workspaceResult.value;
    let inventory: DiscoveryResult | null = null;
    try {
      inventory = await loadWorkspaceInventory(workspace.snapshots ?? []);
    } catch (reason) {
      setState((current) => ({
        ...current,
        actionError: describeError(reason, {
          scope: "workspace snapshots",
          fallback: "Unable to load workspace snapshots.",
        }),
      }));
    }
    setSettingsDraft({
      target_assumptions: workspace.target_assumptions ?? {},
      plan_settings: workspace.plan_settings ?? {},
    });
    setState((current) => ({
      ...current,
      selectedWorkspace: workspace,
      jobs: jobsResult.status === "fulfilled" ? jobsResult.value : [],
      inventory,
      refreshing: false,
      error: null,
      actionError:
        jobsResult.status === "rejected"
          ? describeError(jobsResult.reason, {
              scope: "workspace jobs",
              fallback: "Unable to load workspace jobs.",
            })
          : current.actionError,
    }));
  }

  useEffect(() => {
    void refreshWorkspaces();
  }, []);

  useEffect(() => {
    if (!selectedWorkspaceID) {
      return;
    }
    void refreshWorkspaceDetail(selectedWorkspaceID);
  }, [selectedWorkspaceID]);

  useEffect(() => {
    if (!state.selectedWorkspace || !state.jobs.some((job) => job.status === "queued" || job.status === "running")) {
      return;
    }
    const handle = window.setTimeout(() => {
      void refreshWorkspaceDetail(state.selectedWorkspace!.id);
    }, 1500);
    return () => window.clearTimeout(handle);
  }, [state.jobs, state.selectedWorkspace]);

  const graph = state.selectedWorkspace?.graph?.raw_json ?? null;
  const simulation = state.selectedWorkspace?.simulation?.raw_json;
  const rows = useMemo(
    () =>
      buildInventoryAssessmentRows(
        state.inventory?.vms ?? [],
        graph as DependencyGraphModel | null,
        simulation?.policy_report ?? null,
        simulation?.recommendation_report ?? null,
        {
          graph: Boolean(graph),
          policies: Boolean(simulation?.policy_report),
          remediation: Boolean(simulation?.recommendation_report),
        },
        state.inventory?.discovered_at,
      ),
    [graph, simulation, state.inventory],
  );
  const savedSelection = useMemo(() => state.selectedWorkspace?.selected_workload_ids ?? [], [state.selectedWorkspace?.selected_workload_ids]);
  const inventoryWorkspace = useInventoryWorkspace(rows, savedSelection);
  const latestSnapshot = state.selectedWorkspace?.snapshots?.[0]
    ? {
        id: state.selectedWorkspace.snapshots[0].snapshot_id,
        source: state.selectedWorkspace.snapshots[0].source,
        platform: state.selectedWorkspace.snapshots[0].platform,
        vm_count: state.selectedWorkspace.snapshots[0].vm_count,
        discovered_at: state.selectedWorkspace.snapshots[0].discovered_at,
      }
    : null;

  async function handleCreateWorkspace() {
    setCreating(true);
    try {
      const created = await createWorkspace({
        name: createForm.name,
        description: createForm.description,
        status: "draft",
        source_connections: [
          {
            id: "source-lab",
            name: createForm.sourceName,
            platform: createForm.sourcePlatform as Platform,
            address: createForm.sourceAddress,
            credential_ref: createForm.sourceCredentialRef,
          },
        ],
        target_assumptions: {
          platform: createForm.targetPlatform as Platform,
          address: createForm.targetAddress,
          default_host: createForm.defaultHost,
          default_storage: createForm.defaultStorage,
          default_network: createForm.defaultNetwork,
        },
        plan_settings: {
          name: `${createForm.name.toLowerCase().replace(/\s+/g, "-")}-plan`,
          parallel: 2,
          verify_boot: true,
          approval_required: true,
          wave_size: 2,
          dependency_aware: true,
        },
      });
      setState((current) => ({ ...current, actionError: null }));
      await refreshWorkspaces(created.id);
    } catch (reason) {
      setState((current) => ({
        ...current,
        actionError: describeError(reason, {
          scope: "workspace creation",
          fallback: "Unable to create the pilot workspace.",
        }),
      }));
    } finally {
      setCreating(false);
    }
  }

  async function handleWorkspaceUpdate(payload: Partial<PilotWorkspace>, loadingKey: string) {
    if (!state.selectedWorkspace) {
      return;
    }
    setActionLoading(loadingKey);
    try {
      await updateWorkspace(state.selectedWorkspace.id, payload);
      if (loadingKey === "notes") {
        setNoteDraft("");
      }
      setState((current) => ({ ...current, actionError: null }));
      await refreshWorkspaceDetail(state.selectedWorkspace.id);
    } catch (reason) {
      setState((current) => ({
        ...current,
        actionError: describeError(reason, {
          scope: "workspace update",
          fallback: "Unable to save workspace updates.",
        }),
      }));
    } finally {
      setActionLoading(null);
    }
  }

  async function handleJob(type: WorkspaceJobType) {
    if (!state.selectedWorkspace) {
      return;
    }
    setActionLoading(type);
    try {
      await createWorkspaceJob(state.selectedWorkspace.id, {
        type,
        selected_workload_ids: type === "simulation" || type === "plan" ? inventoryWorkspace.selectedIds : undefined,
      });
      setState((current) => ({ ...current, actionError: null }));
      await refreshWorkspaceDetail(state.selectedWorkspace.id);
    } catch (reason) {
      setState((current) => ({
        ...current,
        actionError: describeError(reason, {
          scope: `${type} job`,
          fallback: `Unable to start the ${type} job.`,
        }),
      }));
    } finally {
      setActionLoading(null);
    }
  }

  async function handleExportReport() {
    if (!state.selectedWorkspace) {
      return;
    }
    setActionLoading("report");
    try {
      const result = await exportWorkspaceReport(state.selectedWorkspace.id, "markdown");
      const url = window.URL.createObjectURL(result.blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = result.filename;
      link.click();
      window.URL.revokeObjectURL(url);
      await refreshWorkspaceDetail(state.selectedWorkspace.id);
    } catch (reason) {
      setState((current) => ({
        ...current,
        actionError: describeError(reason, {
          scope: "report export",
          fallback: "Unable to export the pilot report.",
        }),
      }));
    } finally {
      setActionLoading(null);
    }
  }

  if (state.loading) {
    return <LoadingState title="Loading pilot workspaces" message="Retrieving saved pilot assessments and the latest workspace-first operator flow." />;
  }

  if (state.error && state.workspaces.length === 0) {
    return (
      <ErrorState
        title="Pilot workspaces unavailable"
        message={state.error.message}
        technicalDetails={state.error.technicalDetails}
        actions={[
          <button key="retry" type="button" onClick={() => void refreshWorkspaces()} className="rounded-full border border-rose-200 bg-white px-4 py-2 text-sm font-semibold text-rose-700 transition hover:bg-rose-50">
            Retry
          </button>,
        ]}
      />
    );
  }

  if (state.workspaces.length === 0) {
    return (
      <div className="space-y-6">
        <EmptyState
          title="Create the first pilot workspace"
          message="Start with a persisted assessment workspace that ties discovery, graphing, simulation, saved plans, approvals, notes, and report export together."
        />
        <SectionCard title="Workspace intake" description="The local lab defaults are prefilled so a fresh clone can get to the first successful report flow quickly.">
          <div className="grid gap-3 md:grid-cols-2">
            <LabeledInput label="Workspace name" value={createForm.name} onChange={(value) => setCreateForm((current) => ({ ...current, name: value }))} />
            <LabeledInput label="Description" value={createForm.description} onChange={(value) => setCreateForm((current) => ({ ...current, description: value }))} />
            <LabeledInput label="Source name" value={createForm.sourceName} onChange={(value) => setCreateForm((current) => ({ ...current, sourceName: value }))} />
            <LabeledInput label="Source address" value={createForm.sourceAddress} onChange={(value) => setCreateForm((current) => ({ ...current, sourceAddress: value }))} />
            <LabeledInput label="Credential ref" value={createForm.sourceCredentialRef} onChange={(value) => setCreateForm((current) => ({ ...current, sourceCredentialRef: value }))} />
            <LabeledInput label="Target address" value={createForm.targetAddress} onChange={(value) => setCreateForm((current) => ({ ...current, targetAddress: value }))} />
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <button type="button" onClick={() => void handleCreateWorkspace()} disabled={creating} className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:opacity-60">
              {creating ? "Creating..." : "Create workspace"}
            </button>
          </div>
        </SectionCard>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Pilot Workspace"
        title={state.selectedWorkspace?.name ?? "Pilot Workspace"}
        description="Create, discover, inspect, simulate, save, and export from one persisted operator workspace instead of bouncing between disconnected surfaces."
        badges={[
          { label: `${state.workspaces.length} workspace(s)`, tone: "neutral" },
          { label: state.selectedWorkspace?.status ?? "draft", tone: "info" },
          { label: `${state.selectedWorkspace?.snapshots?.length ?? 0} snapshot(s)`, tone: "neutral" },
        ]}
        actions={
          <>
            <select value={selectedWorkspaceID ?? ""} onChange={(event) => setSelectedWorkspaceID(event.target.value)} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-700">
              {state.workspaces.map((workspace) => (
                <option key={workspace.id} value={workspace.id}>{workspace.name}</option>
              ))}
            </select>
            <button type="button" onClick={() => void refreshWorkspaceDetail(selectedWorkspaceID ?? "")} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              Refresh
            </button>
          </>
        }
      />

      {state.actionError && <ErrorState title="Workspace action blocked" message={state.actionError.message} technicalDetails={state.actionError.technicalDetails} />}

      <section className="grid gap-5 xl:grid-cols-[0.9fr_1.1fr]">
        <SectionCard title="Assessment intake" description="Keep source connections, target assumptions, and plan controls in one operator-owned document.">
          <div className="grid gap-3 md:grid-cols-2">
            <LabeledInput label="Target address" value={settingsDraft.target_assumptions?.address ?? ""} onChange={(value) => setSettingsDraft((current) => ({ ...current, target_assumptions: { ...current.target_assumptions, address: value } }))} />
            <LabeledInput label="Default host" value={settingsDraft.target_assumptions?.default_host ?? ""} onChange={(value) => setSettingsDraft((current) => ({ ...current, target_assumptions: { ...current.target_assumptions, default_host: value } }))} />
            <LabeledInput label="Default storage" value={settingsDraft.target_assumptions?.default_storage ?? ""} onChange={(value) => setSettingsDraft((current) => ({ ...current, target_assumptions: { ...current.target_assumptions, default_storage: value } }))} />
            <LabeledInput label="Default network" value={settingsDraft.target_assumptions?.default_network ?? ""} onChange={(value) => setSettingsDraft((current) => ({ ...current, target_assumptions: { ...current.target_assumptions, default_network: value } }))} />
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <button type="button" onClick={() => void handleWorkspaceUpdate(settingsDraft, "settings")} className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800">
              {actionLoading === "settings" ? "Saving..." : "Save assumptions"}
            </button>
            <button type="button" onClick={() => void handleJob("discovery")} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              {actionLoading === "discovery" ? "Running discovery..." : "Run discovery"}
            </button>
            <button type="button" onClick={() => void handleJob("graph")} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              {actionLoading === "graph" ? "Building graph..." : "Build graph"}
            </button>
            <button type="button" onClick={() => void handleJob("simulation")} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              {actionLoading === "simulation" ? "Running simulation..." : "Run simulation"}
            </button>
            <button type="button" onClick={() => void handleJob("plan")} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              {actionLoading === "plan" ? "Saving plan..." : "Save plan"}
            </button>
            <button type="button" onClick={() => void handleExportReport()} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              {actionLoading === "report" ? "Exporting..." : "Export report"}
            </button>
          </div>
        </SectionCard>

        <SectionCard title="Readiness and jobs" description="Persisted background jobs keep discovery, graphing, simulation, and planning reproducible.">
          <div className="grid gap-3 md:grid-cols-2">
            <Metric label="Readiness" value={state.selectedWorkspace?.readiness?.status ?? "pending"} />
            <Metric label="Selected workloads" value={String(state.selectedWorkspace?.readiness?.selected_workload_count ?? inventoryWorkspace.selectedIds.length)} />
            <Metric label="Policy issues" value={String(state.selectedWorkspace?.readiness?.policy_violation_count ?? 0)} />
            <Metric label="Recommendations" value={String(state.selectedWorkspace?.readiness?.recommendation_count ?? 0)} />
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            {(state.jobs.length === 0 ? [{ id: "none", status: "neutral", message: "No jobs recorded yet." }] : state.jobs.slice(0, 6)).map((job) => (
              <StatusBadge key={job.id} tone={job.status === "failed" ? "danger" : job.status === "succeeded" ? "success" : "warning"}>
                {"type" in job ? `${job.type}: ${job.status}` : job.message}
              </StatusBadge>
            ))}
          </div>
        </SectionCard>
      </section>

      {rows.length === 0 ? (
        <EmptyState title="No discovered workloads yet" message="Run discovery for this workspace to persist a snapshot baseline before inspection, simulation, or plan generation." />
      ) : (
        <section className="grid gap-5 xl:grid-cols-[1.35fr_0.85fr]">
          <InventoryTable
            rows={inventoryWorkspace.filteredRows}
            totalCount={rows.length}
            filteredCount={inventoryWorkspace.filteredRows.length}
            selectedCount={inventoryWorkspace.selectedIds.length}
            hasActiveFilters={inventoryWorkspace.hasActiveFilters}
            loading={state.refreshing}
            refreshing={state.refreshing}
            availablePlatforms={inventoryWorkspace.availablePlatforms}
            filters={inventoryWorkspace.filters}
            sortKey={inventoryWorkspace.sortKey}
            sortDirection={inventoryWorkspace.sortDirection}
            activeWorkloadId={inventoryWorkspace.activeRow?.id ?? null}
            selectedIds={inventoryWorkspace.selectedIds}
            actions={
              <button type="button" onClick={() => void handleWorkspaceUpdate({ selected_workload_ids: inventoryWorkspace.selectedIds }, "selection")} className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800">
                {actionLoading === "selection" ? "Saving selection..." : "Save selection"}
              </button>
            }
            onFiltersChange={inventoryWorkspace.updateFilters}
            onSortChange={inventoryWorkspace.changeSort}
            onToggleSelection={inventoryWorkspace.toggleSelection}
            onToggleSelectAllVisible={inventoryWorkspace.toggleSelectAllVisible}
            onClearSelection={inventoryWorkspace.clearSelection}
            onResetFilters={inventoryWorkspace.resetFilters}
            onFocusWorkload={inventoryWorkspace.setActiveWorkloadId}
          />
          <WorkloadDetailPanel row={inventoryWorkspace.activeRow} latestSnapshot={latestSnapshot} onPreparePlan={(row) => inventoryWorkspace.replaceSelection([row.id])} />
        </section>
      )}

      <section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
        <DependencyGraph graph={graph as DependencyGraphModel | null} />
        <SectionCard title="Notes and exports" description="Keep operator commentary and pilot artifacts attached to the same workspace.">
          <textarea value={noteDraft} onChange={(event) => setNoteDraft(event.target.value)} className="min-h-32 w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-ink outline-none transition focus:border-sky-400" placeholder="Add operator notes, pilot caveats, or approval context." />
          <div className="mt-4 flex flex-wrap gap-2">
            <button type="button" onClick={() => void handleWorkspaceUpdate({ notes: [...(state.selectedWorkspace?.notes ?? []), { id: "", kind: "operator", author: state.selectedWorkspace?.tenant_id ?? "operator", body: noteDraft, created_at: "" } as WorkspaceNote] }, "notes")} className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800" disabled={noteDraft.trim() === ""}>
              {actionLoading === "notes" ? "Saving note..." : "Save note"}
            </button>
            <button type="button" onClick={() => void handleExportReport()} className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50">
              Download pilot report
            </button>
          </div>
          <div className="mt-4 space-y-2">
            {(state.selectedWorkspace?.reports ?? []).slice(0, 4).map((report) => (
              <div key={report.id} className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-700">
                {report.file_name} • {report.exported_at}
              </div>
            ))}
          </div>
        </SectionCard>
      </section>
    </div>
  );
}

function LabeledInput({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="block">
      <span className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</span>
      <input value={value} onChange={(event) => onChange(event.target.value)} className="mt-2 w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-ink outline-none transition focus:border-sky-400" />
    </label>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-3">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 text-sm font-semibold text-ink">{value}</p>
    </div>
  );
}

async function loadWorkspaceInventory(snapshots: WorkspaceSnapshot[]): Promise<DiscoveryResult | null> {
  if (snapshots.length === 0) {
    return null;
  }
  const results = await Promise.all(snapshots.map((snapshot) => getSnapshot(snapshot.snapshot_id)));
  return mergeDiscoveryResults(results);
}

function mergeDiscoveryResults(results: DiscoveryResult[]): DiscoveryResult {
  return results.reduce<DiscoveryResult>(
    (merged, current) => ({
      source: merged.source || current.source,
      platform: merged.platform || current.platform,
      vms: [...merged.vms, ...(current.vms ?? [])],
      networks: [...(merged.networks ?? []), ...(current.networks ?? [])],
      datastores: [...(merged.datastores ?? []), ...(current.datastores ?? [])],
      hosts: [...(merged.hosts ?? []), ...(current.hosts ?? [])],
      clusters: [...(merged.clusters ?? []), ...(current.clusters ?? [])],
      resource_pools: [...(merged.resource_pools ?? []), ...(current.resource_pools ?? [])],
      discovered_at: merged.discovered_at ?? current.discovered_at,
      errors: [...(merged.errors ?? []), ...(current.errors ?? [])],
      duration: (merged.duration ?? 0) + (current.duration ?? 0),
    }),
    { vms: [] },
  );
}
