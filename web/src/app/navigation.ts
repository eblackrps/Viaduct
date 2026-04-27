import {
	FileText,
	FolderKanban,
	GitCompare,
	LayoutDashboard,
	Network,
	Server,
	Settings,
	ShieldCheck,
	TrendingUp,
	Waypoints,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

export type AppRoutePath =
	| "/workspaces"
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

export const defaultRoute: AppRoutePath = "/workspaces";

export const navigationGroups: NavigationGroup[] = [
	{
		label: "Operate",
		items: [
			{
				path: "/workspaces",
				label: "Assessments",
				title: "Assessments",
				description:
					"Start an assessment, run discovery, review dependencies, save plans, and export reports.",
				icon: FolderKanban,
			},
			{
				path: "/dashboard",
				label: "Overview",
				title: "Dashboard",
				description:
					"Review tenant status, migration work, and lifecycle signals in one place.",
				icon: LayoutDashboard,
			},
			{
				path: "/inventory",
				label: "Inventory",
				title: "Fleet Inventory",
				description:
					"Inspect discovered workloads, placement signals, and platform distribution.",
				icon: Server,
			},
			{
				path: "/migrations",
				label: "Migrations",
				title: "Migrations",
				description: "Plan, validate, execute, and review workload migrations.",
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
				description:
					"Review remediation guidance and cost status for the current workload baseline.",
				icon: TrendingUp,
			},
			{
				path: "/policy",
				label: "Policy",
				title: "Policy Controls",
				description:
					"Inspect compliance rules, violations, and enforcement status across current inventory.",
				icon: ShieldCheck,
			},
			{
				path: "/drift",
				label: "Drift",
				title: "Drift Monitoring",
				description:
					"Track workload divergence from discovery baselines before execution windows close.",
				icon: GitCompare,
			},
		],
	},
	{
		label: "Observe",
		items: [
			{
				path: "/reports",
				label: "Reports",
				title: "Reports and history",
				description:
					"Review migration records, discovery snapshots, and exportable reports.",
				icon: FileText,
			},
			{
				path: "/graph",
				label: "Dependency Graph",
				title: "Dependency Graph",
				description:
					"Inspect workload relationships across network, storage, and backup dependencies.",
				icon: Network,
			},
		],
	},
	{
		label: "Admin",
		items: [
			{
				path: "/settings",
				label: "Settings",
				title: "Settings",
				description:
					"Inspect tenant context, sign-in state, and dashboard settings.",
				icon: Settings,
			},
		],
	},
];

const navigationItems = navigationGroups.flatMap((group) => group.items);
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
	return (
		navigationItems.find((item) => item.path === normalized) ??
		navigationItems[0]
	);
}
