import {
	getRouteHref,
	type AppRoutePath,
	type NavigationGroup,
} from "../../app/navigation";

interface SidebarNavProps {
	groups: NavigationGroup[];
	currentPath: AppRoutePath;
}

export function SidebarNav({ groups, currentPath }: SidebarNavProps) {
	return (
		<nav aria-label="Primary" className="space-y-5">
			{groups.map((group) => (
				<div key={group.label}>
					<p className="operator-kicker px-2">{group.label}</p>
					<div className="mt-2 space-y-1">
						{group.items.map((item) => {
							const Icon = item.icon;
							const active = currentPath === item.path;
							return (
								<a
									key={item.path}
									href={getRouteHref(item.path)}
									aria-current={active ? "page" : undefined}
									className={`flex items-center gap-3 rounded-xl border px-3 py-2.5 transition ${
										active
											? "border-ink bg-ink text-white shadow-panel"
											: "border-transparent text-slate-700 hover:border-slate-200 hover:bg-slate-50"
									}`}
								>
									<span
										className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
											active
												? "bg-white/10 text-white"
												: "bg-slate-100 text-slate-600"
										}`}
									>
										<Icon className="h-4 w-4" />
									</span>
									<span className="text-sm font-semibold">{item.label}</span>
								</a>
							);
						})}
					</div>
				</div>
			))}
		</nav>
	);
}
