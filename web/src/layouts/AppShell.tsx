import type { ReactNode } from "react";
import { useState } from "react";
import { Menu } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../app/navigation";
import { SidebarNav } from "../components/navigation/SidebarNav";
import { MobileSidebarDrawer } from "../components/navigation/MobileSidebarDrawer";
import { TopBar } from "../components/navigation/TopBar";
import { ErrorBanner } from "../components/primitives/ErrorBanner";

interface AppShellProps {
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
	error?: string | null;
	children: ReactNode;
}

export function AppShell({
	currentPath,
	tenantId,
	lastDiscoveryAt,
	refreshing,
	onRefresh,
	authSummary,
	onSignOut,
	error,
	children,
}: AppShellProps) {
	const [mobileNavOpen, setMobileNavOpen] = useState(false);

	return (
		<div className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
			<div className="mx-auto grid max-w-[1600px] gap-6 xl:grid-cols-[280px_1fr]">
				{/* Desktop sidebar */}
				<aside className="hidden xl:block">
					<div className="panel p-4 space-y-4">
						{/* Brand */}
						<div className="rounded-xl bg-gradient-to-br from-ink via-steel to-slate-900 px-5 py-4 text-white">
							<p className="operator-kicker !text-slate-300">
								Operator console
							</p>
							<p className="mt-2 font-display text-2xl">Viaduct</p>
							<p className="mt-2 text-xs leading-5 text-slate-300">
								Discover, plan, and execute migrations from one tenant-scoped
								control plane.
							</p>
						</div>

						<SidebarNav groups={navigationGroups} currentPath={currentPath} />
					</div>
				</aside>

				<main className="min-w-0 space-y-5">
					{/* Mobile hamburger + TopBar row */}
					<div className="flex items-start gap-3">
						<button
							type="button"
							aria-label="Open navigation"
							onClick={() => setMobileNavOpen(true)}
							className="xl:hidden mt-1 inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
						>
							<Menu className="h-4 w-4" />
						</button>
						<div className="flex-1 min-w-0">
							<TopBar
								currentPath={currentPath}
								tenantId={tenantId}
								lastDiscoveryAt={lastDiscoveryAt}
								refreshing={refreshing}
								onRefresh={onRefresh}
								authSummary={authSummary}
								onSignOut={onSignOut}
							/>
						</div>
					</div>

					{error && <ErrorBanner message={error} />}

					<div className="space-y-5">{children}</div>
				</main>
			</div>

			<MobileSidebarDrawer
				open={mobileNavOpen}
				onClose={() => setMobileNavOpen(false)}
				currentPath={currentPath}
			/>
		</div>
	);
}
