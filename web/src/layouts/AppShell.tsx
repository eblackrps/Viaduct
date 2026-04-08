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
  error,
  children,
}: AppShellProps) {
  return (
    <div className="min-h-screen bg-transparent px-4 py-6 md:px-6">
      <div className="mx-auto grid max-w-[1550px] gap-6 xl:grid-cols-[300px_1fr]">
        <aside className="panel p-5">
          <div className="rounded-[2rem] bg-ink px-5 py-6 text-white">
            <p className="font-display text-3xl">Viaduct</p>
            <p className="mt-2 text-sm text-slate-300">Hypervisor-agnostic workload migration and lifecycle management.</p>
          </div>

          <SidebarNav groups={navigationGroups} currentPath={currentPath} />
        </aside>

        <main className="space-y-6">
          <TopBar
            tenantId={tenantId}
            lastDiscoveryAt={lastDiscoveryAt}
            statusItems={statusItems}
            refreshing={refreshing}
            onRefresh={onRefresh}
          />

          {error && <p className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</p>}

          <div className="rounded-[2rem] border border-white/60 bg-white/35 p-6 backdrop-blur-sm">{children}</div>
        </main>
      </div>
    </div>
  );
}
