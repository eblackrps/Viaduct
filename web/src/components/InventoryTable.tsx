import { useDeferredValue, useEffect, useState } from "react";
import { ArrowDownUp, Search } from "lucide-react";
import { getInventory } from "../api";
import type { DiscoveryResult, PowerState, VirtualMachine } from "../types";

type SortKey = "name" | "platform" | "power_state" | "cpu_count" | "memory_mb" | "host" | "cluster";

interface InventoryTableProps {
  inventory?: DiscoveryResult | null;
}

export function InventoryTable({ inventory }: InventoryTableProps) {
  const [localInventory, setLocalInventory] = useState<DiscoveryResult | null>(inventory ?? null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [platformFilter, setPlatformFilter] = useState("all");
  const [powerFilter, setPowerFilter] = useState<PowerState | "all">("all");
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const deferredSearch = useDeferredValue(search);

  useEffect(() => {
    setLocalInventory(inventory ?? null);
  }, [inventory]);

  useEffect(() => {
    if (inventory) {
      return;
    }

    let cancelled = false;
    setLoading(true);
    getInventory()
      .then((result) => {
        if (!cancelled) {
          setLocalInventory(result);
          setError(null);
        }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setError(err.message);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [inventory]);

  const rows = [...(localInventory?.vms ?? [])]
    .filter((vm) => {
      if (platformFilter !== "all" && vm.platform !== platformFilter) {
        return false;
      }
      if (powerFilter !== "all" && vm.power_state !== powerFilter) {
        return false;
      }

      const text = deferredSearch.trim().toLowerCase();
      if (!text) {
        return true;
      }

      return [
        vm.name,
        vm.platform,
        vm.power_state,
        String(vm.cpu_count),
        String(vm.memory_mb),
        vm.host,
        vm.cluster ?? "",
      ]
        .join(" ")
        .toLowerCase()
        .includes(text);
    })
    .sort((left, right) => {
      const leftValue = String(left[sortKey] ?? "").toLowerCase();
      const rightValue = String(right[sortKey] ?? "").toLowerCase();
      const result = leftValue.localeCompare(rightValue, undefined, { numeric: true });
      return sortDirection === "asc" ? result : -result;
    });

  const platforms = Array.from(new Set((localInventory?.vms ?? []).map((vm) => vm.platform)));

  function toggleSort(nextKey: SortKey) {
    if (sortKey === nextKey) {
      setSortDirection((value) => (value === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(nextKey);
    setSortDirection("asc");
  }

  return (
    <section className="panel overflow-hidden p-5">
      <div className="mb-5 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="font-display text-2xl text-ink">Inventory View</p>
          <p className="text-sm text-slate-500">Search, sort, and compare discovered workloads across hypervisors.</p>
        </div>
        <div className="flex flex-col gap-3 md:flex-row">
          <label className="flex items-center gap-2 rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-500">
            <Search className="h-4 w-4" />
            <input
              className="w-48 border-none bg-transparent outline-none"
              placeholder="Search inventory"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
          <select
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-700"
            value={platformFilter}
            onChange={(event) => setPlatformFilter(event.target.value)}
          >
            <option value="all">All Platforms</option>
            {platforms.map((platform) => (
              <option key={platform} value={platform}>
                {platform}
              </option>
            ))}
          </select>
          <select
            className="rounded-full border border-slate-200 bg-white px-4 py-2 text-sm text-slate-700"
            value={powerFilter}
            onChange={(event) => setPowerFilter(event.target.value as PowerState | "all")}
          >
            <option value="all">All States</option>
            <option value="on">On</option>
            <option value="off">Off</option>
            <option value="suspended">Suspended</option>
          </select>
        </div>
      </div>

      {loading && <p className="text-sm text-slate-500">Loading inventory…</p>}
      {error && <p className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

      <div className="overflow-x-auto">
        <table className="min-w-full border-separate border-spacing-y-2 text-left">
          <thead>
            <tr className="text-xs uppercase tracking-[0.22em] text-slate-500">
              {[
                ["name", "Name"],
                ["platform", "Platform"],
                ["power_state", "State"],
                ["cpu_count", "CPU"],
                ["memory_mb", "Memory"],
                ["host", "Host"],
                ["cluster", "Cluster"],
              ].map(([key, label]) => (
                <th key={key} className="px-3 py-2">
                  <button
                    type="button"
                    className="flex items-center gap-2"
                    onClick={() => toggleSort(key as SortKey)}
                  >
                    {label}
                    <ArrowDownUp className="h-3.5 w-3.5" />
                  </button>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((vm) => (
              <tr key={vm.id || vm.name} className="rounded-2xl bg-slate-50/80 text-sm text-slate-700">
                <td className="rounded-l-2xl px-3 py-3 font-semibold text-ink">{vm.name}</td>
                <td className="px-3 py-3">
                  <span className={`rounded-full px-3 py-1 text-xs font-semibold ${platformClass(vm.platform)}`}>{vm.platform}</span>
                </td>
                <td className="px-3 py-3">
                  <span className="inline-flex items-center gap-2">
                    <span className={`h-2.5 w-2.5 rounded-full ${stateDotClass(vm.power_state)}`} />
                    {vm.power_state}
                  </span>
                </td>
                <td className="px-3 py-3">{vm.cpu_count}</td>
                <td className="px-3 py-3">{vm.memory_mb.toLocaleString()} MB</td>
                <td className="px-3 py-3">{vm.host || "Unknown"}</td>
                <td className="rounded-r-2xl px-3 py-3">{vm.cluster || "Unassigned"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {!loading && rows.length === 0 && (
        <p className="mt-4 rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
          No workloads match the current filters.
        </p>
      )}
    </section>
  );
}

function platformClass(platform: VirtualMachine["platform"]) {
  switch (platform) {
    case "vmware":
      return "bg-emerald-100 text-emerald-700";
    case "proxmox":
      return "bg-orange-100 text-orange-700";
    case "hyperv":
      return "bg-sky-100 text-sky-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

function stateDotClass(state: PowerState) {
  switch (state) {
    case "on":
      return "bg-emerald-500";
    case "off":
      return "bg-rose-500";
    case "suspended":
      return "bg-amber-500";
    default:
      return "bg-slate-400";
  }
}
