import { useEffect, useMemo, useState } from "react";
import { createMigration, executeMigration, getInventory, getMigrationState, rollbackMigration, runPreflight } from "../api";
import { MigrationProgress } from "./MigrationProgress";
import type { DiscoveryResult, MigrationSpec, MigrationState, Platform, PreflightReport } from "../types";

const steps = ["Source", "Select VMs", "Target", "Network Mapping", "Review", "Pre-flight", "Execute"];

interface MigrationWizardProps {
  onMigrationChange?: () => void;
}

export function MigrationWizard({ onMigrationChange }: MigrationWizardProps) {
  const [step, setStep] = useState(0);
  const [sourcePlatform, setSourcePlatform] = useState<Platform>("vmware");
  const [sourceAddress, setSourceAddress] = useState("");
  const [inventory, setInventory] = useState<DiscoveryResult | null>(null);
  const [selectedVMs, setSelectedVMs] = useState<string[]>([]);
  const [selectionSearch, setSelectionSearch] = useState("");
  const [targetPlatform, setTargetPlatform] = useState<Platform>("proxmox");
  const [targetAddress, setTargetAddress] = useState("");
  const [defaultHost, setDefaultHost] = useState("pve-01");
  const [defaultStorage, setDefaultStorage] = useState("local-lvm");
  const [networkMap, setNetworkMap] = useState<Record<string, string>>({});
  const [approvalRequired, setApprovalRequired] = useState(false);
  const [approvedBy, setApprovedBy] = useState("");
  const [scheduledStart, setScheduledStart] = useState("");
  const [waveSize, setWaveSize] = useState(2);
  const [preflight, setPreflight] = useState<PreflightReport | null>(null);
  const [migrationState, setMigrationState] = useState<MigrationState | null>(null);
  const [migrationID, setMigrationID] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedWorkloads = useMemo(
    () => (inventory?.vms ?? []).filter((vm) => selectedVMs.includes(vm.id)),
    [inventory?.vms, selectedVMs],
  );
  const filteredWorkloads = useMemo(
    () => (inventory?.vms ?? []).filter((vm) => vm.name.toLowerCase().includes(selectionSearch.trim().toLowerCase())),
    [inventory?.vms, selectionSearch],
  );
  const sourceNetworks = Array.from(new Set(selectedWorkloads.flatMap((vm) => vm.nics.map((nic) => nic.network)).filter(Boolean)));

  useEffect(() => {
    if (!migrationID || step !== 6) {
      return;
    }

    const interval = window.setInterval(() => {
      getMigrationState(migrationID)
        .then((state) => {
          setMigrationState(state);
          onMigrationChange?.();
        })
        .catch((err: Error) => setError(err.message));
    }, 2000);

    return () => window.clearInterval(interval);
  }, [migrationID, onMigrationChange, step]);

  async function discoverSource() {
    setLoading(true);
    setError(null);
    try {
      const result = await getInventory(sourcePlatform);
      setInventory(result);
      setSelectedVMs(result.vms.map((vm) => vm.id));
      setStep(1);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function buildSpec(): MigrationSpec {
    return {
      name: "dashboard-migration",
      source: {
        address: sourceAddress || "source.local",
        platform: sourcePlatform,
      },
      target: {
        address: targetAddress || "target.local",
        platform: targetPlatform,
        default_host: defaultHost,
        default_storage: defaultStorage,
      },
      workloads: selectedWorkloads.map((vm) => ({
        match: {
          name_pattern: vm.name,
        },
        overrides: {
          target_host: defaultHost,
          target_storage: defaultStorage,
          network_map: networkMap,
        },
      })),
      options: {
        parallel: 2,
        shutdown_source: true,
        verify_boot: true,
        window: scheduledStart ? { not_before: new Date(scheduledStart).toISOString() } : undefined,
        approval: approvalRequired
          ? {
              required: true,
              approved_by: approvedBy || undefined,
              approved_at: approvedBy ? new Date().toISOString() : undefined,
            }
          : undefined,
        waves: {
          size: waveSize,
          dependency_aware: true,
        },
      },
    };
  }

  async function handlePreflight() {
    setLoading(true);
    setError(null);
    try {
      const report = await runPreflight(buildSpec());
      setPreflight(report);
      setStep(5);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleExecute() {
    setLoading(true);
    setError(null);
    try {
      const planned = await createMigration(buildSpec());
      setMigrationID(planned.id);
      setMigrationState(planned);
      await executeMigration(planned.id);
      setStep(6);
      onMigrationChange?.();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleRollback() {
    if (!migrationID) {
      return;
    }
    setLoading(true);
    try {
      await rollbackMigration(migrationID);
      const state = await getMigrationState(migrationID);
      setMigrationState(state);
      onMigrationChange?.();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="space-y-5">
      <div className="panel p-5">
        <div className="flex flex-wrap gap-3">
          {steps.map((label, index) => (
            <div key={label} className={`rounded-full px-4 py-2 text-sm font-semibold ${index <= step ? "bg-ink text-white" : "bg-slate-100 text-slate-500"}`}>
              {index + 1}. {label}
            </div>
          ))}
        </div>

        {error && <p className="mt-4 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

        {step === 0 && (
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Source Platform</span>
              <select className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={sourcePlatform} onChange={(event) => setSourcePlatform(event.target.value as Platform)}>
                <option value="vmware">VMware</option>
                <option value="proxmox">Proxmox</option>
                <option value="hyperv">Hyper-V</option>
              </select>
            </label>
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Source Address</span>
              <input className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={sourceAddress} onChange={(event) => setSourceAddress(event.target.value)} placeholder="vcsa.lab.local" />
            </label>
            <div className="md:col-span-2">
              <button type="button" className="rounded-full bg-accent px-5 py-3 text-sm font-semibold text-white" onClick={discoverSource} disabled={loading}>
                {loading ? "Discovering…" : "Discover"}
              </button>
            </div>
          </div>
        )}

        {step === 1 && (
          <div className="mt-6 space-y-4">
            <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <input className="rounded-2xl border border-slate-200 px-4 py-3 text-sm md:w-72" placeholder="Filter discovered VMs" value={selectionSearch} onChange={(event) => setSelectionSearch(event.target.value)} />
              <button type="button" className="rounded-full border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700" onClick={() => setSelectedVMs(filteredWorkloads.map((vm) => vm.id))}>
                Select All
              </button>
            </div>
            <div className="grid gap-3">
              {filteredWorkloads.map((vm) => (
                <label key={vm.id} className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm">
                  <input
                    type="checkbox"
                    checked={selectedVMs.includes(vm.id)}
                    onChange={() =>
                      setSelectedVMs((current) =>
                        current.includes(vm.id) ? current.filter((item) => item !== vm.id) : [...current, vm.id],
                      )
                    }
                  />
                  <span className="font-semibold text-ink">{vm.name}</span>
                  <span className="text-slate-500">{vm.platform}</span>
                </label>
              ))}
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Target Platform</span>
              <select className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={targetPlatform} onChange={(event) => setTargetPlatform(event.target.value as Platform)}>
                <option value="proxmox">Proxmox</option>
                <option value="vmware">VMware</option>
                <option value="hyperv">Hyper-V</option>
              </select>
            </label>
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Target Address</span>
              <input className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={targetAddress} onChange={(event) => setTargetAddress(event.target.value)} placeholder="pve.lab.local" />
            </label>
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Default Host</span>
              <input className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={defaultHost} onChange={(event) => setDefaultHost(event.target.value)} />
            </label>
            <label className="space-y-2 text-sm">
              <span className="font-semibold text-ink">Default Storage</span>
              <input className="w-full rounded-2xl border border-slate-200 px-4 py-3" value={defaultStorage} onChange={(event) => setDefaultStorage(event.target.value)} />
            </label>
          </div>
        )}

        {step === 3 && (
          <div className="mt-6 space-y-4">
            {sourceNetworks.map((network) => (
              <label key={network} className="grid gap-2 text-sm md:grid-cols-[1fr_1fr] md:items-center">
                <span className="font-semibold text-ink">{network}</span>
                <select
                  className="rounded-2xl border border-slate-200 px-4 py-3"
                  value={networkMap[network] ?? ""}
                  onChange={(event) => setNetworkMap((current) => ({ ...current, [network]: event.target.value }))}
                >
                  <option value="">Choose target network</option>
                  <option value="vmbr0">vmbr0</option>
                  <option value="vmbr1">vmbr1</option>
                  <option value="External">External</option>
                </select>
              </label>
            ))}
          </div>
        )}

        {step === 4 && (
          <div className="mt-6 space-y-4 text-sm text-slate-600">
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="font-semibold text-ink">Source</p>
              <p>{sourcePlatform} · {sourceAddress || "source.local"}</p>
            </div>
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="font-semibold text-ink">Target</p>
              <p>{targetPlatform} · {targetAddress || "target.local"} · host {defaultHost} · storage {defaultStorage}</p>
            </div>
            <div className="rounded-3xl bg-slate-50 p-4">
              <p className="font-semibold text-ink">Selected Workloads</p>
              <ul className="mt-2 flex flex-wrap gap-2">
                {selectedWorkloads.map((vm) => (
                  <li key={vm.id} className="rounded-full bg-white px-3 py-1">{vm.name}</li>
                ))}
              </ul>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <label className="rounded-3xl bg-slate-50 p-4">
                <span className="font-semibold text-ink">Scheduled Start</span>
                <input className="mt-3 w-full rounded-2xl border border-slate-200 px-4 py-3" type="datetime-local" value={scheduledStart} onChange={(event) => setScheduledStart(event.target.value)} />
              </label>
              <label className="rounded-3xl bg-slate-50 p-4">
                <span className="font-semibold text-ink">Wave Size</span>
                <input className="mt-3 w-full rounded-2xl border border-slate-200 px-4 py-3" type="number" min={1} value={waveSize} onChange={(event) => setWaveSize(Number(event.target.value) || 1)} />
              </label>
            </div>
            <div className="rounded-3xl bg-slate-50 p-4">
              <label className="flex items-center gap-3 font-semibold text-ink">
                <input type="checkbox" checked={approvalRequired} onChange={(event) => setApprovalRequired(event.target.checked)} />
                Require approval before execution
              </label>
              {approvalRequired && (
                <input className="mt-3 w-full rounded-2xl border border-slate-200 px-4 py-3" value={approvedBy} onChange={(event) => setApprovedBy(event.target.value)} placeholder="Approver name" />
              )}
            </div>
          </div>
        )}

        {step === 5 && (
          <div className="mt-6 space-y-3">
            {preflight?.checks.map((check) => (
              <div key={check.name} className="rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <p className="font-semibold text-ink">{check.name}</p>
                  <span className={`rounded-full px-3 py-1 text-xs font-semibold ${check.status === "pass" ? "bg-emerald-100 text-emerald-700" : check.status === "warn" ? "bg-amber-100 text-amber-700" : "bg-rose-100 text-rose-700"}`}>
                    {check.status}
                  </span>
                </div>
                <p className="mt-2 text-slate-500">{check.message}</p>
              </div>
            ))}
            {preflight?.plan && (
              <div className="rounded-3xl bg-slate-50 p-4 text-sm text-slate-600">
                <p className="font-semibold text-ink">Execution Runbook</p>
                <p className="mt-2">{preflight.plan.total_workloads} workloads across {preflight.plan.waves.length} wave(s).</p>
                <div className="mt-3 space-y-2">
                  {preflight.plan.waves.map((wave) => (
                    <div key={wave.index} className="rounded-2xl bg-white px-4 py-3">
                      <p className="font-semibold text-ink">Wave {wave.index}</p>
                      <p className="text-slate-500">{wave.workloads.map((item) => item.name).join(", ")}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {preflight && (
              <button
                type="button"
                className="rounded-full bg-ink px-5 py-3 text-sm font-semibold text-white disabled:bg-slate-300"
                onClick={handleExecute}
                disabled={!preflight.can_proceed || loading}
              >
                {loading ? "Starting…" : "Execute Migration"}
              </button>
            )}
          </div>
        )}

        {step === 6 && <div className="mt-6"><MigrationProgress state={migrationState} onRollback={handleRollback} /></div>}

        {step < 5 && (
          <div className="mt-6 flex justify-between">
            <button type="button" className="rounded-full border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700" onClick={() => setStep((value) => Math.max(0, value - 1))} disabled={step === 0}>
              Back
            </button>
            {step < 4 && (
              <button type="button" className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white" onClick={() => setStep((value) => Math.min(4, value + 1))} disabled={step === 1 && selectedVMs.length === 0}>
                Next
              </button>
            )}
            {step === 4 && (
              <button type="button" className="rounded-full bg-accent px-4 py-2 text-sm font-semibold text-white" onClick={handlePreflight} disabled={loading}>
                {loading ? "Running…" : "Run Pre-flight"}
              </button>
            )}
          </div>
        )}
      </div>
    </section>
  );
}
