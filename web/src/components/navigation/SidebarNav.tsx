import { getRouteHref, type AppRoutePath, type NavigationGroup } from "../../app/navigation";

interface SidebarNavProps {
  groups: NavigationGroup[];
  currentPath: AppRoutePath;
}

export function SidebarNav({ groups, currentPath }: SidebarNavProps) {
  return (
    <div className="space-y-5">
      <nav aria-label="Primary" className="space-y-6">
        {groups.map((group) => (
          <div key={group.label}>
            <p className="operator-kicker px-3">{group.label}</p>
            <div className="mt-3 space-y-2">
              {group.items.map((item) => {
                const Icon = item.icon;
                const active = currentPath === item.path;
                return (
                  <a
                    key={item.path}
                    href={getRouteHref(item.path)}
                    aria-current={active ? "page" : undefined}
                    className={`flex items-start gap-3 rounded-[1.4rem] border px-4 py-3 transition ${
                      active
                        ? "border-ink bg-ink text-white shadow-panel"
                        : "border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50"
                    }`}
                  >
                    <span
                      className={`mt-0.5 inline-flex h-10 w-10 items-center justify-center rounded-2xl ${
                        active ? "bg-white/10 text-white" : "bg-slate-100 text-slate-600"
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                    </span>
                    <span className="min-w-0">
                      <span className="block text-sm font-semibold">{item.label}</span>
                      <span className={`mt-1 block text-xs leading-5 ${active ? "text-slate-200" : "text-slate-500"}`}>
                        {item.description}
                      </span>
                    </span>
                  </a>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      <section className="panel-muted p-4">
        <p className="operator-kicker">Operator path</p>
        <p className="mt-2 text-sm font-semibold text-ink">Start with authentication, then stay inside one workspace record.</p>
        <p className="mt-2 text-sm text-slate-600">
          The recommended path is create workspace, discover, inspect, simulate, save a plan, and export a report.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <a
            href="https://github.com/eblackrps/Viaduct/blob/main/QUICKSTART.md"
            className="operator-button-secondary px-3 py-2 text-xs"
          >
            Quickstart
          </a>
          <a
            href="https://github.com/eblackrps/Viaduct/blob/main/docs/operations/pilot-workspace-flow.md"
            className="operator-button-secondary px-3 py-2 text-xs"
          >
            Workspace guide
          </a>
        </div>
      </section>
    </div>
  );
}
