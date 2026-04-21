import { useId, useRef } from "react";
import { X } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../../app/navigation";
import { SidebarNav } from "./SidebarNav";
import { useFocusTrap } from "./useFocusTrap";

interface MobileSidebarDrawerProps {
	drawerID: string;
	open: boolean;
	onClose: () => void;
	currentPath: AppRoutePath;
}

export function MobileSidebarDrawer({
	drawerID,
	open,
	onClose,
	currentPath,
}: MobileSidebarDrawerProps) {
	const drawerRef = useRef<HTMLElement | null>(null);
	const titleID = useId();
	useFocusTrap(drawerRef, open, onClose);

	if (!open) {
		return null;
	}

	return (
		<>
			<div
				aria-hidden="true"
				className="fixed inset-0 z-40 bg-ink/35 backdrop-blur-sm lg:hidden"
				onClick={onClose}
			/>

			<aside
				id={drawerID}
				ref={drawerRef}
				role="dialog"
				aria-modal="true"
				aria-labelledby={titleID}
				tabIndex={-1}
				className="fixed inset-y-0 left-0 z-50 flex w-[88vw] max-w-[320px] flex-col gap-4 bg-transparent p-4 lg:hidden"
			>
				<div className="panel flex h-full min-h-0 flex-col px-4 py-4">
					<p id={titleID} className="sr-only">
						Primary navigation
					</p>
					<div className="rounded-2xl bg-gradient-to-br from-ink via-steel to-slate-900 px-4 py-4 text-white">
						<div className="flex items-start justify-between gap-3">
							<div>
								<p className="operator-kicker !text-slate-300">
									Operator console
								</p>
								<p className="mt-2 font-display text-xl tracking-[-0.03em]">
									Viaduct
								</p>
							</div>
							<button
								type="button"
								aria-label="Close navigation"
								onClick={onClose}
								className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-white/15 bg-white/10 text-white transition hover:bg-white/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:ring-offset-2 focus-visible:ring-offset-ink"
							>
								<X className="h-4 w-4" />
							</button>
						</div>
						<p className="mt-3 text-sm leading-6 text-slate-200">
							Discover, plan, execute, and govern from the same control plane.
						</p>
					</div>

					<div className="mt-4 flex-1 overflow-y-auto">
						<SidebarNav
							groups={navigationGroups}
							currentPath={currentPath}
							onNavigate={onClose}
						/>
					</div>
				</div>
			</aside>
		</>
	);
}
