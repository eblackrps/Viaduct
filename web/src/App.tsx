import { useEffect, useState } from "react";
import { Activity, Coins, Database, GitBranch, LayoutDashboard, ShieldCheck, Waypoints } from "lucide-react";
import { getCosts, getDrift, getInventory, getPolicies, getRemediation, getSnapshots, getTenantSummary, listMigrations, runSimulation } from "./api";
import { CostComparison } from "./components/CostComparison";
import { DependencyGraph } from "./components/DependencyGraph";
import { DriftTimeline } from "./components/DriftTimeline";
import { InventoryTable } from "./components/InventoryTable";
import { MigrationHistory } from "./components/MigrationHistory";
import { MigrationWizard } from "./components/MigrationWizard";
import { PlatformSummary } from "./components/PlatformSummary";
import { PolicyDashboard } from "./components/PolicyDashboard";
import { RemediationPanel } from "./components/RemediationPanel";
import type { DiscoveryResult, DriftReport, MigrationMeta, Platform, PlatformComparison, PolicyReport, RecommendationReport, SimulationResult, SnapshotMeta, TenantSummary } from "./types";

type View = "dashboard" | "migrate" | "graph" | "lifecycle" | "history";

export default function App() {
  const [view, setView] = useState<View>("dashboard");
  const [inventory, setInventory] = useState<DiscoveryResult | null>(null);
  const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([]);
  const [migrations, setMigrations] = useState<MigrationMeta[]>([]);
  const [costs, setCosts] = useState<PlatformComparison[]>([]);
  const [policies, setPolicies] = useState<PolicyReport | null>(null);
  const [drift, setDrift] = useState<DriftReport | null>(null);
  const [remediation, setRemediation] = useState<RecommendationReport | null>(null);
  const [summary, setSummary] = useState<TenantSummary | null>(null);
  const [simulation, setSimulation] = useState<SimulationResult | null>(null);
  const [simulationLoading, setSimulationLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    try {
      const [inventoryResult, snapshotResult, migrationResult, costResult, policyResult, summaryResult] = await Promise.all([
        getInventory(),
        getSnapshots(),
        listMigrations(),
        getCosts("all"),
        getPolicies(),
        getTenantSummary(),
      ]);

      setInventory(inventoryResult);
      setSnapshots(snapshotResult);
      setMigrations(migrationResult);
      setCosts(Array.isArray(costResult) ? costResult : []);
      setPolicies(policyResult);
      setSummary(summaryResult);

      if (snapshotResult.length > 0) {
        setDrift(await getDrift(snapshotResult[0].id));
        setRemediation(await getRemediation(snapshotResult[0].id));
      } else {
        setDrift(null);
        setRemediation(await getRemediation());
      }

      setError(null);
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleSimulation(targetPlatform: Platform) {
    try {
      setSimulationLoading(true);
      setSimulation(await runSimulation({ target_platform: targetPlatform, include_all: true }));
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSimulationLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-transparent px-4 py-6 md:px-6">
      <div className="mx-auto grid max-w-[1500px] gap-6 xl:grid-cols-[280px_1fr]">
        <aside className="panel p-5">
          <div className="rounded-[2rem] bg-ink px-5 py-6 text-white">
            <p className="font-display text-3xl">Viaduct</p>
            <p className="mt-2 text-sm text-slate-300">Hypervisor-agnostic workload migration and lifecycle intelligence.</p>
          </div>

          <nav className="mt-6 space-y-2">
            {[
              { id: "dashboard", label: "Inventory", icon: LayoutDashboard },
              { id: "migrate", label: "Migrate", icon: Waypoints },
              { id: "graph", label: "Dependency Graph", icon: GitBranch },
              { id: "lifecycle", label: "Lifecycle", icon: Coins },
              { id: "history", label: "History", icon: Activity },
            ].map((item) => {
              const Icon = item.icon;
              const active = view === item.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setView(item.id as View)}
                  className={`flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-left text-sm font-semibold transition ${active ? "bg-accent text-white" : "bg-slate-50 text-slate-700 hover:bg-slate-100"}`}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </button>
              );
            })}
          </nav>

          <div className="mt-6 space-y-4">
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Workloads</p>
              <p className="mt-3 font-display text-3xl text-ink">{summary?.workload_count ?? inventory?.vms.length ?? 0}</p>
            </div>
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Snapshots</p>
              <p className="mt-3 font-display text-3xl text-ink">{summary?.snapshot_count ?? snapshots.length}</p>
            </div>
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Recommendations</p>
              <p className="mt-3 font-display text-3xl text-ink">{summary?.recommendation_count ?? remediation?.recommendations.length ?? 0}</p>
            </div>
          </div>
        </aside>

        <main className="space-y-6">
          <header className="panel flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <p className="font-display text-4xl text-ink">Operate the bridge</p>
              <p className="mt-2 max-w-2xl text-sm text-slate-500">Discover, migrate, compare cost profiles, enforce lifecycle policy, and catch drift before your next cutover window closes.</p>
            </div>
            <div className="flex gap-3">
              <div className="rounded-full bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
                <Database className="mr-2 inline h-4 w-4" />
                REST API + Store
              </div>
              <div className="rounded-full bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
                <ShieldCheck className="mr-2 inline h-4 w-4" />
                Policy + Drift Aware
              </div>
            </div>
          </header>

          {error && <p className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

          {view === "dashboard" && (
            <>
              <PlatformSummary inventory={inventory} />
              <InventoryTable inventory={inventory} />
            </>
          )}

          {view === "migrate" && <MigrationWizard onMigrationChange={refresh} />}
          {view === "graph" && <DependencyGraph />}
          {view === "lifecycle" && (
            <section className="space-y-5">
              <RemediationPanel
                report={remediation}
                simulation={simulation}
                onSimulate={handleSimulation}
                simulationLoading={simulationLoading}
              />
              <CostComparison comparisons={costs} />
              <PolicyDashboard report={policies} />
              <DriftTimeline report={drift} />
            </section>
          )}
          {view === "history" && (
            <section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
              <MigrationHistory migrations={migrations} />

              <div className="panel p-5">
                <p className="font-display text-2xl text-ink">Discovery Snapshots</p>
                <div className="mt-5 space-y-3">
                  {snapshots.map((snapshot) => (
                    <article key={snapshot.id} className="rounded-2xl bg-slate-50 px-4 py-4 text-sm text-slate-600">
                      <p className="font-semibold text-ink">{snapshot.source}</p>
                      <p className="text-slate-500">{snapshot.platform} · {snapshot.vm_count} VMs</p>
                    </article>
                  ))}
                  {snapshots.length === 0 && <p className="text-sm text-slate-500">No discovery snapshots have been saved yet.</p>}
                </div>
              </div>
            </section>
          )}
        </main>
      </div>
    </div>
  );
}
