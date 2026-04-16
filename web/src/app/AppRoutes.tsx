import { useEffect, type ReactNode } from "react";
import { requestManager } from "../api";
import { AuthBootstrapScreen } from "../features/auth/AuthBootstrapScreen";
import {
	useAuthBootstrap,
	type AuthBootstrapState,
} from "../features/auth/useAuthBootstrap";
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
import {
	getDashboardAuthPersistence,
	getDashboardAuthSession,
} from "../runtimeAuth";
import { getNavigationItem, type AppRoutePath } from "./navigation";
import { useHashRoute } from "./useHashRoute";
import {
	useOperatorOverview,
	type OperatorOverviewState,
} from "./useOperatorOverview";

function renderRoute(
	path: AppRoutePath,
	overview: OperatorOverviewState,
	options: {
		authSourceLabel: string;
		authPersistenceLabel: string;
		onForgetRuntimeKey?: (() => void) | undefined;
	},
): ReactNode {
	const inventoryError = joinMessages(
		overview.errors.inventory,
		overview.errors.summary,
	);

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
					inventoryPagination={overview.inventoryPagination}
					inventoryPage={overview.inventoryPage}
					summary={overview.summary}
					latestSnapshot={overview.latestSnapshot}
					refreshToken={overview.refreshToken}
					loading={overview.loading}
					error={inventoryError}
					onInventoryPageChange={overview.setInventoryPage}
				/>
			);
		case "/migrations":
			return (
				<MigrationsPage
					inventory={overview.inventory}
					migrations={overview.migrations}
					migrationsPagination={overview.migrationsPagination}
					migrationsPage={overview.migrationsPage}
					snapshots={overview.snapshots}
					snapshotsPagination={overview.snapshotsPagination}
					snapshotsPage={overview.snapshotsPage}
					summary={overview.summary}
					latestSnapshot={overview.latestSnapshot}
					refreshToken={overview.refreshToken}
					loading={overview.loading}
					migrationError={overview.errors.migrations}
					snapshotError={overview.errors.snapshots}
					onMigrationChange={overview.refresh}
					onMigrationsPageChange={overview.setMigrationsPage}
					onSnapshotsPageChange={overview.setSnapshotsPage}
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
					migrationsPagination={overview.migrationsPagination}
					migrationsPage={overview.migrationsPage}
					snapshots={overview.snapshots}
					snapshotsPagination={overview.snapshotsPagination}
					snapshotsPage={overview.snapshotsPage}
					loading={overview.loading}
					migrationError={overview.errors.migrations}
					snapshotError={overview.errors.snapshots}
					onMigrationsPageChange={overview.setMigrationsPage}
					onSnapshotsPageChange={overview.setSnapshotsPage}
				/>
			);
		case "/settings":
			return (
				<SettingsPage
					summary={overview.summary}
					authSourceLabel={options.authSourceLabel}
					authPersistenceLabel={options.authPersistenceLabel}
					onForgetRuntimeKey={options.onForgetRuntimeKey}
				/>
			);
		case "/graph":
			return <GraphPage />;
		default:
			return null;
	}
}

function AuthenticatedAppRoutes({ auth }: { auth: AuthBootstrapState }) {
	const { path } = useHashRoute();
	const overview = useOperatorOverview();
	const currentRoute = getNavigationItem(path);
	const authSession = getDashboardAuthSession();
	const authPersistence = getDashboardAuthPersistence();
	const localSingleUserMode =
		authSession.mode === "none" &&
		auth.currentTenant?.auth_method === "default-fallback";
	const authSourceLabel = localSingleUserMode
		? "Local single-user session"
		: authSession.mode === "service-account"
			? "Service-account key"
			: authSession.mode === "tenant"
				? "Tenant key"
				: "No runtime credential";
	const authPersistenceLabel = localSingleUserMode
		? "No browser credential required on this local runtime"
		: authPersistence === "local"
			? "Remembered in this browser"
			: authPersistence === "session"
				? "Session-only browser session"
				: authPersistence === "environment"
					? "Provided by environment configuration"
					: "No stored credential";

	useEffect(() => {
		return () => {
			requestManager.cancelAll();
		};
	}, [path]);

	return (
		<AppShell
			currentPath={currentRoute.path}
			tenantId={overview.summary?.tenant_id}
			lastDiscoveryAt={
				overview.summary?.last_discovery_at ??
				overview.latestSnapshot?.discovered_at
			}
			refreshing={overview.refreshing}
			onRefresh={overview.refresh}
			authSummary={{
				modeLabel: authSourceLabel,
				persistenceLabel: authPersistenceLabel,
			}}
			onSignOut={
				authSession.source === "runtime" && authSession.mode !== "none"
					? auth.signOut
					: undefined
			}
			error={overview.error}
		>
			{renderRoute(currentRoute.path, overview, {
				authSourceLabel,
				authPersistenceLabel,
				onForgetRuntimeKey:
					authSession.source === "runtime" && authSession.mode !== "none"
						? auth.signOut
						: undefined,
			})}
		</AppShell>
	);
}

export function AppRoutes() {
	const auth = useAuthBootstrap();

	if (auth.status !== "authenticated") {
		return <AuthBootstrapScreen auth={auth} />;
	}

	return <AuthenticatedAppRoutes auth={auth} />;
}

function joinMessages(
	...values: Array<string | null | undefined>
): string | null {
	const filtered = values.filter((value): value is string => Boolean(value));
	return filtered.length > 0 ? filtered.join(" ") : null;
}
