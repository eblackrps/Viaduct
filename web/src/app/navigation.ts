import {
  Activity,
  Coins,
  Database,
  FileText,
  GitBranch,
  LayoutDashboard,
  Settings,
  ShieldCheck,
  Waypoints,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

export type AppRoutePath =
  | "/dashboard"
  | "/inventory"
  | "/migrations"
  | "/lifecycle"
  | "/policy"
  | "/drift"
  | "/reports"
  | "/settings"
  | "/graph";

export interface NavigationItem {
  path: AppRoutePath;
  label: string;
  title: string;
  description: string;
  icon: LucideIcon;
}

export interface NavigationGroup {
  label: string;
  items: NavigationItem[];
}

export const defaultRoute: AppRoutePath = "/dashboard";

export const navigationGroups: NavigationGroup[] = [
  {
    label: "Operate",
    items: [
      {
        path: "/dashboard",
        label: "Dashboard",
        title: "Operational Dashboard",
        description: "Review tenant posture, migration flow, and lifecycle signals in one operational surface.",
        icon: LayoutDashboard,
      },
      {
        path: "/inventory",
        label: "Inventory",
        title: "Fleet Inventory",
        description: "Inspect discovered workloads, placement signals, and platform distribution across the estate.",
        icon: Database,
      },
      {
        path: "/migrations",
        label: "Migrations",
        title: "Migration Operations",
        description: "Plan, validate, execute, and review workload migrations without leaving the operational context.",
        icon: Waypoints,
      },
    ],
  },
  {
    label: "Govern",
    items: [
      {
        path: "/lifecycle",
        label: "Lifecycle",
        title: "Lifecycle Optimization",
        description: "Review remediation guidance and cost posture for the current workload baseline.",
        icon: Coins,
      },
      {
        path: "/policy",
        label: "Policy",
        title: "Policy Controls",
        description: "Inspect compliance rules, violations, and enforcement posture across current inventory.",
        icon: ShieldCheck,
      },
      {
        path: "/drift",
        label: "Drift",
        title: "Drift Monitoring",
        description: "Track workload divergence from discovery baselines before execution windows close.",
        icon: Activity,
      },
    ],
  },
  {
    label: "Admin",
    items: [
      {
        path: "/reports",
        label: "Reports",
        title: "Reports and History",
        description: "Review historical migration records, discovery snapshots, and export-ready operator reports.",
        icon: FileText,
      },
      {
        path: "/settings",
        label: "Settings",
        title: "Workspace Settings",
        description: "Inspect tenant context, operator connection assumptions, and dashboard runtime configuration.",
        icon: Settings,
      },
    ],
  },
  {
    label: "Analysis",
    items: [
      {
        path: "/graph",
        label: "Dependency Graph",
        title: "Dependency Graph",
        description: "Inspect workload relationships across network, storage, and backup dependencies.",
        icon: GitBranch,
      },
    ],
  },
];

export const navigationItems = navigationGroups.flatMap((group) => group.items);
const knownPaths = new Set(navigationItems.map((item) => item.path));

export function getRouteHref(path: AppRoutePath): string {
  return `#${path}`;
}

export function normalizeRoutePath(value: string): AppRoutePath {
  const trimmed = value.trim().replace(/^#/, "");
  if (trimmed === "") {
    return defaultRoute;
  }

  const candidate = trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
  if (knownPaths.has(candidate as AppRoutePath)) {
    return candidate as AppRoutePath;
  }

  return defaultRoute;
}

export function getNavigationItem(path: string): NavigationItem {
  const normalized = normalizeRoutePath(path);
  return navigationItems.find((item) => item.path === normalized) ?? navigationItems[0];
}
