import {
	getRouteHref,
	type AppRoutePath,
	type NavigationGroup,
} from "../../app/navigation";

interface SidebarNavProps {
	groups: NavigationGroup[];
	currentPath: AppRoutePath;
	onNavigate?: () => void;
}

export function SidebarNav({
	groups,
	currentPath,
	onNavigate,
}: SidebarNavProps) {
	return (
		<nav aria-label="Primary" className="space-y-5">
			{groups.map((group) => (
				<div key={group.label}>
					<p className="operator-kicker px-2">{group.label}</p>
					<div className="mt-3 space-y-2">
						{group.items.map((item) => {
							const Icon = item.icon;
							const active = currentPath === item.path;

							return (
								<a
									key={item.path}
									href={getRouteHref(item.path)}
									aria-current={active ? "page" : undefined}
									onClick={onNavigate}
									className={`group block rounded-xl border px-3.5 py-3.5 transition duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:ring-offset-2 focus-visible:ring-offset-paper ${
										active
											? "border-ink bg-ink text-white shadow-[0_18px_30px_rgba(15,23,42,0.22)]"
											: "border-transparent bg-transparent text-slate-700 hover:border-slate-200/80 hover:bg-white/70"
									}`}
								>
									<span className="flex items-start gap-3">
										<span
											className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md ${
												active
													? "bg-white/10 text-white"
													: "bg-slate-100/90 text-slate-600 transition group-hover:bg-white group-hover:text-ink"
											}`}
										>
											<Icon className="h-4 w-4" />
										</span>
										<span className="min-w-0">
											<span className="block text-sm font-semibold">
												{item.label}
											</span>
											<span
												className={`mt-1 block text-xs leading-5 ${
													active ? "text-slate-300" : "text-slate-500"
												}`}
											>
												{item.title}
											</span>
										</span>
									</span>
								</a>
							);
						})}
					</div>
				</div>
			))}
		</nav>
	);
}
