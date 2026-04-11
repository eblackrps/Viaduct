import type { ReactNode } from "react";
import { navigationGroups, type AppRoutePath } from "../app/navigation";
import { SidebarNav } from "../components/navigation/SidebarNav";
import { TopBar } from "../components/navigation/TopBar";
import type { StatusTone } from "../components/primitives/StatusBadge";

interface AppShellStatusItem {
  label: string;
  value: string | number;
  tone?: StatusTone;
}

interface AppShellProps {
  currentPath: AppRoutePath;
  tenantId?: string;
  lastDiscoveryAt?: string;
  statusItems: AppShellStatusItem[];
  refreshing: boolean;
  onRefresh: () => void | Promise<void>;
  authSummary?: {
    modeLabel: string;
    persistenceLabel: string;
  };
  onSignOut?: (() => void) | undefined;
  error?: string | null;
  children: ReactNode;
}

export function AppShell({
  currentPath,
  tenantId,
  lastDiscoveryAt,
  statusItems,
  refreshing,
  onRefresh,
  authSummary,
  onSignOut,
  error,
  children,
}: AppShellProps) {
  const currentItem = navigationGroups.flatMap((group) => group.items).find((item) => item.path === currentPath);

  return (
    <div className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
      <div className="mx-auto grid max-w-[1600px] gap-6 xl:grid-cols-[320px_1fr]">
        <aside className="space-y-5">
          <div className="panel overflow-hidden p-5">
            <div className="rounded-[1.8rem] bg-gradient-to-br from-ink via-steel to-slate-900 px-6 py-6 text-white">
              <p className="operator-kicker !text-slate-300">Viaduct operator console</p>
              <p className="mt-3 font-display text-4xl">Viaduct</p>
              <p className="mt-3 max-w-xs text-sm leading-6 text-slate-200">
                Discover mixed estates, map dependencies, plan controlled migrations, and keep operator handoff data in one tenant-scoped system.
              </p>
            </div>

            <div className="mt-5 grid gap-3">
              <div className="panel-muted p-4">
                <p className="operator-kicker">Default flow</p>
                <p className="mt-2 text-sm font-semibold text-ink">Authenticate, create a workspace, discover, inspect, simulate, plan, and export.</p>
              </div>
              <div className="panel-muted p-4">
                <p className="operator-kicker">Shared truth</p>
                <p className="mt-2 text-sm text-slate-600">The dashboard stays aligned with the same persisted API and store-backed model used by the CLI and packaged release.</p>
              </div>
            </div>
          </div>

          <div className="panel p-5">
            <SidebarNav groups={navigationGroups} currentPath={currentPath} />
          </div>
        </aside>

        <main className="space-y-6">
          <TopBar
            currentTitle={currentItem?.title ?? "Operator Console"}
            currentDescription={currentItem?.description ?? "Operate Viaduct from a shared, tenant-scoped control plane."}
            tenantId={tenantId}
            lastDiscoveryAt={lastDiscoveryAt}
            statusItems={statusItems}
            refreshing={refreshing}
            onRefresh={onRefresh}
            authSummary={authSummary}
            onSignOut={onSignOut}
          />

          {error && <p className="panel-muted border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

          <div className="panel p-6 lg:p-7">{children}</div>
        </main>
      </div>
    </div>
  );
}
