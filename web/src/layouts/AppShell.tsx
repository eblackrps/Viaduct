import type { ReactNode } from "react";
import { useEffect, useId, useRef, useState } from "react";
import { Menu } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../app/navigation";
import { MobileSidebarDrawer } from "../components/navigation/MobileSidebarDrawer";
import { SidebarNav } from "../components/navigation/SidebarNav";
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
	const [restoreNavFocus, setRestoreNavFocus] = useState(false);
	const navigationDrawerID = useId();
	const mobileNavButtonRef = useRef<HTMLButtonElement | null>(null);
	const shellContentRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		const shellContent = shellContentRef.current;
		if (!shellContent) {
			return;
		}

		if (mobileNavOpen) {
			shellContent.setAttribute("inert", "");
			return () => {
				shellContent.removeAttribute("inert");
			};
		}

		shellContent.removeAttribute("inert");
	}, [mobileNavOpen]);

	useEffect(() => {
		if (mobileNavOpen || !restoreNavFocus) {
			return;
		}

		mobileNavButtonRef.current?.focus();
		setRestoreNavFocus(false);
	}, [mobileNavOpen, restoreNavFocus]);

	function closeMobileNav() {
		setRestoreNavFocus(true);
		setMobileNavOpen(false);
	}

	return (
		<div className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
			<div
				ref={shellContentRef}
				className="mx-auto grid max-w-[1600px] gap-6 2xl:grid-cols-[304px_minmax(0,1fr)]"
			>
				<aside className="hidden 2xl:block">
					<div className="sticky top-6 space-y-4">
						<div className="panel px-4 py-4">
							<div className="rounded-[26px] bg-gradient-to-br from-ink via-steel to-slate-900 px-5 py-5 text-white shadow-[0_20px_38px_rgba(15,23,42,0.24)]">
								<p className="operator-kicker !text-slate-300">
									Operator console
								</p>
								<p className="mt-3 font-display text-[2rem] leading-none tracking-[-0.04em]">
									Viaduct
								</p>
								<p className="mt-3 text-sm leading-6 text-slate-200">
									Workspace-first migration operations with tenant-scoped
									discovery, planning, execution, and governance in one place.
								</p>
								<div className="mt-5 flex flex-wrap gap-2">
									<span className="rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-semibold text-white/90">
										Workspace-first
									</span>
									<span className="rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-semibold text-white/90">
										Tenant-scoped
									</span>
								</div>
							</div>

							<div className="mt-4">
								<SidebarNav
									groups={navigationGroups}
									currentPath={currentPath}
								/>
							</div>
						</div>

						<div className="panel-muted px-4 py-4 text-sm text-slate-600">
							<p className="operator-kicker">Operator Flow</p>
							<p className="mt-2 font-semibold text-ink">
								Intake, discover, inspect, simulate, plan, execute, report.
							</p>
							<p className="mt-2 leading-6">
								The dashboard keeps operator evidence attached to the same
								workspace instead of scattering it across disconnected screens.
							</p>
						</div>
					</div>
				</aside>

				<main className="min-w-0 space-y-6">
					<div className="flex items-start gap-3">
						<button
							ref={mobileNavButtonRef}
							type="button"
							aria-label="Open navigation"
							aria-controls={navigationDrawerID}
							aria-expanded={mobileNavOpen}
							onClick={() => setMobileNavOpen(true)}
							className="operator-button-secondary 2xl:hidden h-11 w-11 shrink-0 rounded-full px-0"
						>
							<Menu className="h-4 w-4" />
						</button>
						<div className="min-w-0 flex-1">
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

					<div className="space-y-6">{children}</div>
				</main>
			</div>

			<MobileSidebarDrawer
				drawerID={navigationDrawerID}
				open={mobileNavOpen}
				onClose={closeMobileNav}
				currentPath={currentPath}
			/>
		</div>
	);
}
