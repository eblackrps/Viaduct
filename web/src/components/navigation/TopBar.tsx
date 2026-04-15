import { KeyRound, LogOut, RefreshCcw } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../../app/navigation";
import { StatusBadge } from "../primitives/StatusBadge";

interface TopBarProps {
  currentPath: AppRoutePath;
  tenantId?: string;
  lastDiscoveryAt?: string;
  refreshing: boolean;
  onRefresh: () => void | Promise<void>;
  authSummary?: {
    modeLabel: string;
    persistenceLabel: string;
  };
  onSignOut?: (() => void) | undefined;
}

export function TopBar({
  currentPath,
  tenantId,
  lastDiscoveryAt,
  refreshing,
  onRefresh,
  authSummary,
  onSignOut,
}: TopBarProps) {
  const currentItem = navigationGroups.flatMap((g) => g.items).find((item) => item.path === currentPath);

  return (
    <div className="panel p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge tone="accent">Operator console</StatusBadge>
            <StatusBadge tone="neutral">{tenantId ?? "default tenant"}</StatusBadge>
            {authSummary && <StatusBadge tone="info">{authSummary.modeLabel}</StatusBadge>}
          </div>
          <p className="text-base font-semibold text-ink">{currentItem?.title ?? "Operator Console"}</p>
          <p className="text-xs text-slate-500 leading-5 max-w-xl">
            {currentItem?.description ?? "Operate Viaduct from a shared, tenant-scoped control plane."}
          </p>
          {lastDiscoveryAt && (
            <p className="text-xs text-slate-400">
              Last discovery: {new Date(lastDiscoveryAt).toLocaleString()}
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2 sm:shrink-0">
          {authSummary && (
            <div className="panel-muted inline-flex items-center gap-1.5 px-3 py-2 text-xs font-semibold text-slate-600">
              <KeyRound className="h-3.5 w-3.5 text-slate-400" />
              {authSummary.persistenceLabel}
            </div>
          )}
          {onSignOut && (
            <button type="button" onClick={onSignOut} className="operator-button-secondary text-xs px-3 py-2">
              <LogOut className="h-3.5 w-3.5" />
              Sign out
            </button>
          )}
          <button
            type="button"
            onClick={() => void onRefresh()}
            disabled={refreshing}
            className="operator-button-secondary text-xs px-3 py-2"
          >
            <RefreshCcw className={`h-3.5 w-3.5 ${refreshing ? "animate-spin" : ""}`} />
            {refreshing ? "Refreshing…" : "Refresh"}
          </button>
        </div>
      </div>
    </div>
  );
}
