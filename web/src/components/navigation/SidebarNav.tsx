import { getRouteHref, type AppRoutePath, type NavigationGroup } from "../../app/navigation";

interface SidebarNavProps {
  groups: NavigationGroup[];
  currentPath: AppRoutePath;
}

export function SidebarNav({ groups, currentPath }: SidebarNavProps) {
  return (
    <nav className="mt-6 space-y-6">
      {groups.map((group) => (
        <div key={group.label}>
          <p className="px-3 text-xs uppercase tracking-[0.22em] text-slate-500">{group.label}</p>
          <div className="mt-3 space-y-2">
            {group.items.map((item) => {
              const Icon = item.icon;
              const active = currentPath === item.path;
              return (
                <a
                  key={item.path}
                  href={getRouteHref(item.path)}
                  aria-current={active ? "page" : undefined}
                  className={`flex items-center gap-3 rounded-2xl px-4 py-3 text-sm font-semibold transition ${
                    active ? "bg-accent text-white" : "bg-slate-50 text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </a>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );
}
