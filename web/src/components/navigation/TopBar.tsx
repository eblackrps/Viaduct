import { Database, KeyRound, LogOut, RefreshCcw, ShieldCheck } from "lucide-react";
import { StatusBadge, type StatusTone } from "../primitives/StatusBadge";

interface TopBarStatusItem {
  label: string;
  value: string | number;
  tone?: StatusTone;
}

interface TopBarProps {
  currentTitle: string;
  currentDescription: string;
  tenantId?: string;
  lastDiscoveryAt?: string;
  statusItems: TopBarStatusItem[];
  refreshing: boolean;
  onRefresh: () => void | Promise<void>;
  authSummary?: {
    modeLabel: string;
    persistenceLabel: string;
  };
  onSignOut?: (() => void) | undefined;
}

export function TopBar({
  currentTitle,
  currentDescription,
  tenantId,
  lastDiscoveryAt,
  statusItems,
  refreshing,
  onRefresh,
  authSummary,
  onSignOut,
}: TopBarProps) {
  return (
    <div className="panel overflow-hidden p-5">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge tone="accent">Operator console</StatusBadge>
            <StatusBadge tone="neutral">{tenantId ?? "default tenant"}</StatusBadge>
            {authSummary ? <StatusBadge tone="info">{authSummary.modeLabel}</StatusBadge> : null}
          </div>
          <div>
            <p className="operator-kicker">Current surface</p>
            <p className="mt-2 font-display text-3xl text-ink">{currentTitle}</p>
            <p className="mt-2 max-w-3xl text-sm text-slate-600">{currentDescription}</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {authSummary ? (
            <div className="panel-muted inline-flex items-center gap-2 px-4 py-3 text-sm font-semibold text-slate-700">
              <KeyRound className="h-4 w-4 text-slate-500" />
              {authSummary.persistenceLabel}
            </div>
          ) : null}
          <div className="panel-muted inline-flex items-center gap-2 px-4 py-3 text-sm font-semibold text-slate-700">
            <Database className="h-4 w-4 text-slate-500" />
            REST API + shared store
          </div>
          <div className="panel-muted inline-flex items-center gap-2 px-4 py-3 text-sm font-semibold text-slate-700">
            <ShieldCheck className="h-4 w-4 text-slate-500" />
            Tenant-scoped visibility
          </div>
          {onSignOut ? (
            <button type="button" onClick={onSignOut} className="operator-button-secondary">
              <LogOut className="h-4 w-4" />
              Forget browser key
            </button>
          ) : null}
          <button type="button" onClick={() => void onRefresh()} disabled={refreshing} className="operator-button-secondary">
            <RefreshCcw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
            {refreshing ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(0,1.15fr)_repeat(4,minmax(0,0.8fr))]">
        <div className="metric-surface">
          <p className="operator-kicker">Tenant context</p>
          <p className="mt-2 text-base font-semibold text-ink">{tenantId ?? "default"}</p>
          <p className="mt-2 text-sm text-slate-600">
            {lastDiscoveryAt
              ? `Last discovery ${new Date(lastDiscoveryAt).toLocaleString()}`
              : "No discovery timestamp has been recorded yet."}
          </p>
        </div>
        {statusItems.map((item) => (
          <div key={`${item.label}-${item.value}`} className="metric-surface">
            <p className="operator-kicker">{item.label}</p>
            <p className="mt-3 font-display text-3xl text-ink">{item.value}</p>
            <div className="mt-3">
              <StatusBadge tone={item.tone}>{item.label}</StatusBadge>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
