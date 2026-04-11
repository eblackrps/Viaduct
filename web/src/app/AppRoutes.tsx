import type { ReactNode } from "react";
import { AuthBootstrapScreen } from "../features/auth/AuthBootstrapScreen";
import { useAuthBootstrap } from "../features/auth/useAuthBootstrap";
import { DashboardPage } from "../features/dashboard/DashboardPage";
import { DriftPage } from "../features/drift/DriftPage";
import { GraphPage } from "../features/graph/GraphPage";
import { InventoryPage } from "../features/inventory/InventoryPage";
import { LifecyclePage } from "../features/lifecycle/LifecyclePage";
import { MigrationsPage } from "../features/migrations/MigrationsPage";
import { PolicyPage } from "../features/policy/PolicyPage";
import { ReportsPage } from "../features/reports/ReportsPage";
import { SettingsPage } from "../features/settings/SettingsPage";
import { WorkspacePage } from "../features/workspaces/WorkspacePage";
import { AppShell } from "../layouts/AppShell";
import { getNavigationItem, type AppRoutePath } from "./navigation";
import { useHashRoute } from "./useHashRoute";
import { useOperatorOverview, type OperatorOverviewState } from "./useOperatorOverview";

function renderRoute(path: AppRoutePath, overview: OperatorOverviewState): ReactNode {
  const inventoryError = joinMessages(overview.errors.inventory, overview.errors.summary);

  switch (path) {
    case "/workspaces":
      return <WorkspacePage />;
    case "/dashboard":
      return (
        <DashboardPage
          inventory={overview.inventory}
          migrations={overview.migrations}
          migrationError={overview.errors.migrations}
          summary={overview.summary}
          latestSnapshot={overview.latestSnapshot}
          loading={overview.loading}
          refreshToken={overview.refreshToken}
        />
      );
    case "/inventory":
      return (
        <InventoryPage
          inventory={overview.inventory}
          summary={overview.summary}
          latestSnapshot={overview.latestSnapshot}
          refreshToken={overview.refreshToken}
          loading={overview.loading}
          error={inventoryError}
        />
      );
    case "/migrations":
      return (
        <MigrationsPage
          inventory={overview.inventory}
          migrations={overview.migrations}
          snapshots={overview.snapshots}
          summary={overview.summary}
          latestSnapshot={overview.latestSnapshot}
          refreshToken={overview.refreshToken}
          loading={overview.loading}
          migrationError={overview.errors.migrations}
          snapshotError={overview.errors.snapshots}
          onMigrationChange={overview.refresh}
        />
      );
    case "/lifecycle":
      return (
        <LifecyclePage
          summary={overview.summary}
          latestSnapshot={overview.latestSnapshot}
          overviewLoading={overview.loading}
          refreshToken={overview.refreshToken}
        />
      );
    case "/policy":
      return <PolicyPage refreshToken={overview.refreshToken} />;
    case "/drift":
      return (
        <DriftPage
          latestSnapshot={overview.latestSnapshot}
          overviewLoading={overview.loading}
          refreshToken={overview.refreshToken}
        />
      );
    case "/reports":
      return (
        <ReportsPage
          migrations={overview.migrations}
          snapshots={overview.snapshots}
          loading={overview.loading}
          migrationError={overview.errors.migrations}
          snapshotError={overview.errors.snapshots}
        />
      );
    case "/settings":
      return <SettingsPage summary={overview.summary} />;
    case "/graph":
      return <GraphPage />;
    default:
      return null;
  }
}

function AuthenticatedAppRoutes() {
  const { path } = useHashRoute();
  const overview = useOperatorOverview();
  const currentRoute = getNavigationItem(path);

  return (
    <AppShell
      currentPath={currentRoute.path}
      tenantId={overview.summary?.tenant_id}
      lastDiscoveryAt={overview.summary?.last_discovery_at ?? overview.latestSnapshot?.discovered_at}
      statusItems={[
        { label: "Workloads", value: overview.summary?.workload_count ?? overview.inventory?.vms.length ?? 0, tone: "info" },
        { label: "Active", value: overview.summary?.active_migrations ?? 0, tone: "accent" },
        { label: "Approvals", value: overview.summary?.pending_approvals ?? 0, tone: "warning" },
        { label: "Failed", value: overview.summary?.failed_migrations ?? 0, tone: (overview.summary?.failed_migrations ?? 0) > 0 ? "danger" : "neutral" },
      ]}
      refreshing={overview.refreshing}
      onRefresh={overview.refresh}
      error={overview.error}
    >
      {renderRoute(currentRoute.path, overview)}
    </AppShell>
  );
}

export function AppRoutes() {
  const auth = useAuthBootstrap();

  if (auth.status !== "authenticated") {
    return <AuthBootstrapScreen auth={auth} />;
  }

  return <AuthenticatedAppRoutes />;
}

function joinMessages(...values: Array<string | null | undefined>): string | null {
  const filtered = values.filter((value): value is string => Boolean(value));
  return filtered.length > 0 ? filtered.join(" ") : null;
}
