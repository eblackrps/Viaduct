export interface OperatorRouteExpectation {
	label: string;
	heading: string | RegExp;
	readyHeading?: string | RegExp;
	snapshotName: string;
}

export const operatorRoutes: OperatorRouteExpectation[] = [
	{
		label: "Assessments",
		heading: "E2E Lab Workspace",
		readyHeading: "Notes and exports",
		snapshotName: "pilot-workspaces",
	},
	{
		label: "Overview",
		heading: "Dashboard",
		snapshotName: "overview",
	},
	{
		label: "Inventory",
		heading: "Fleet inventory and assessment",
		readyHeading: "Workload assessment table",
		snapshotName: "inventory",
	},
	{
		label: "Migration Ops",
		heading: "Migrations",
		snapshotName: "migrations",
	},
	{
		label: "Lifecycle",
		heading: "Lifecycle optimization",
		snapshotName: "lifecycle",
	},
	{
		label: "Policy",
		heading: "Policy controls",
		snapshotName: "policy",
	},
	{
		label: "Drift",
		heading: "Drift monitoring",
		snapshotName: "drift",
	},
	{
		label: "Reports",
		heading: "Reports and history",
		snapshotName: "reports",
	},
	{
		label: "Dependency Graph",
		heading: "Dependency graph",
		snapshotName: "dependency-graph",
	},
	{
		label: "Settings",
		heading: "Settings",
		snapshotName: "settings",
	},
];
