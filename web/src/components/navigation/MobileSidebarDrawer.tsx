import { X } from "lucide-react";
import { navigationGroups, type AppRoutePath } from "../../app/navigation";
import { SidebarNav } from "./SidebarNav";

interface MobileSidebarDrawerProps {
	open: boolean;
	onClose: () => void;
	currentPath: AppRoutePath;
}

export function MobileSidebarDrawer({
	open,
	onClose,
	currentPath,
}: MobileSidebarDrawerProps) {
	return (
		<>
			{/* Backdrop */}
			{open && (
				<div
					aria-hidden="true"
					className="fixed inset-0 z-40 bg-ink/40 backdrop-blur-sm xl:hidden"
					onClick={onClose}
				/>
			)}

			{/* Drawer */}
			<aside
				aria-label="Navigation"
				className={`fixed inset-y-0 left-0 z-50 flex w-72 flex-col gap-4 bg-white p-4 shadow-xl transition-transform duration-200 xl:hidden ${
					open ? "translate-x-0" : "-translate-x-full"
				}`}
			>
				<div className="flex items-center justify-between">
					<div>
						<p className="operator-kicker">Operator console</p>
						<p className="mt-0.5 font-display text-lg text-ink">Viaduct</p>
					</div>
					<button
						type="button"
						aria-label="Close navigation"
						onClick={onClose}
						className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50"
					>
						<X className="h-4 w-4" />
					</button>
				</div>

				<div className="flex-1 overflow-y-auto">
					<SidebarNav groups={navigationGroups} currentPath={currentPath} />
				</div>
			</aside>
		</>
	);
}
