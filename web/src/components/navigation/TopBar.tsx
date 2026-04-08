import { Database, RefreshCcw, ShieldCheck } from "lucide-react";
import { StatusBadge, type StatusTone } from "../primitives/StatusBadge";

interface TopBarStatusItem {
  label: string;
  value: string | number;
  tone?: StatusTone;
}

interface TopBarProps {
  tenantId?: string;
  lastDiscoveryAt?: string;
  statusItems: TopBarStatusItem[];
  refreshing: boolean;
  onRefresh: () => void | Promise<void>;
}

export function TopBar({ tenantId, lastDiscoveryAt, statusItems, refreshing, onRefresh }: TopBarProps) {
  return (
    <div className="panel flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between">
      <div>
        <p className="text-xs uppercase tracking-[0.22em] text-slate-500">Tenant Context</p>
        <p className="mt-2 font-semibold text-ink">{tenantId ?? "default"}</p>
        <p className="mt-1 text-sm text-slate-500">
          {lastDiscoveryAt ? `Last discovery ${new Date(lastDiscoveryAt).toLocaleString()}` : "No discovery timestamp has been recorded yet."}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        {statusItems.map((item) => (
          <StatusBadge key={`${item.label}-${item.value}`} tone={item.tone}>
            {item.label}: {item.value}
          </StatusBadge>
        ))}
        <div className="rounded-full bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
          <Database className="mr-2 inline h-4 w-4" />
          REST API + Store
        </div>
        <div className="rounded-full bg-slate-50 px-4 py-2 text-sm font-semibold text-slate-700">
          <ShieldCheck className="mr-2 inline h-4 w-4" />
          Policy + Drift Aware
        </div>
        <button
          type="button"
          onClick={() => void onRefresh()}
          disabled={refreshing}
          className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-400"
        >
          <RefreshCcw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
          {refreshing ? "Refreshing…" : "Refresh"}
        </button>
      </div>
    </div>
  );
}
