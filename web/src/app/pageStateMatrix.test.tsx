import * as axe from "axe-core";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../api";
import { DashboardPage } from "../features/dashboard/DashboardPage";
import { DriftPage } from "../features/drift/DriftPage";
import { GraphPage } from "../features/graph/GraphPage";
import { InventoryPage } from "../features/inventory/InventoryPage";
import { LifecyclePage } from "../features/lifecycle/LifecyclePage";
import {
	useLifecycleSignals,
	type LifecycleSignalState,
} from "../features/lifecycle/useLifecycleSignals";
import { MigrationsPage } from "../features/migrations/MigrationsPage";
import { PolicyPage } from "../features/policy/PolicyPage";
import { ReportsPage } from "../features/reports/ReportsPage";
import { SettingsPage } from "../features/settings/SettingsPage";
import { useSettingsData } from "../features/settings/useSettingsData";
import { WorkspacePage } from "../features/workspaces/WorkspacePage";
import type {
	AboutResponse,
	CurrentTenant,
	DependencyGraph,
	DiscoveryResult,
	GraphNode,
	PilotWorkspace,
	SnapshotMeta,
	TenantSummary,
} from "../types";

vi.mock("../features/lifecycle/useLifecycleSignals", () => ({
	useLifecycleSignals: vi.fn(),
}));

vi.mock("../features/settings/useSettingsData", () => ({
	useSettingsData: vi.fn(),
}));

vi.mock("../api", async () => {
	const actual = await vi.importActual<typeof api>("../api");
	return {
		...actual,
		createWorkspace: vi.fn(),
		createWorkspaceJob: vi.fn(),
		dashboardAuthMode: vi.fn(() => "tenant"),
		deleteWorkspace: vi.fn(),
		downloadReport: vi.fn(),
		exportWorkspaceReport: vi.fn(),
		getGraph: vi.fn(),
		getSnapshot: vi.fn(),
		getWorkspace: vi.fn(),
		hasApiKeyConfigured: vi.fn(() => true),
		listWorkspaceJobs: vi.fn(),
		listWorkspaces: vi.fn(),
		updateWorkspace: vi.fn(),
	};
});

const mockUseLifecycleSignals = vi.mocked(useLifecycleSignals);
const mockUseSettingsData = vi.mocked(useSettingsData);
const mockGetGraph = vi.mocked(api.getGraph);
const mockListWorkspaces = vi.mocked(api.listWorkspaces);

const sampleSnapshot = {
	id: "snap-1",
	source: "examples/lab/kvm",
	platform: "kvm",
	vm_count: 1,
	discovered_at: "2026-04-21T12:00:00Z",
} as SnapshotMeta;

const sampleInventory = {
	source: "examples/lab/kvm",
	platform: "kvm",
	vms: [
		{
			id: "vm-1",
			name: "ubuntu-web-01",
			platform: "kvm",
			power_state: "on",
			cpu_count: 2,
			memory_mb: 4096,
			disks: [],
			nics: [],
			host: "kvm-node-01",
		},
	],
	networks: [],
	datastores: [],
	hosts: [],
	clusters: [],
	resource_pools: [],
	errors: [],
} as DiscoveryResult;

const sampleSummary = {
	tenant_id: "tenant-a",
	workload_count: 1,
	snapshot_count: 1,
	active_migrations: 1,
	completed_migrations: 0,
	failed_migrations: 0,
	pending_approvals: 0,
	recommendation_count: 1,
	platform_counts: { kvm: 1 },
	snapshot_quota_free: 7,
	migration_quota_free: 7,
} as TenantSummary;

const sampleAbout = {
	name: "Viaduct",
	api_version: "v1",
	version: "3.2.1-dev",
	commit: "abc1234",
	built_at: "2026-04-21T12:00:00Z",
	go_version: "go1.26.0",
	plugin_protocol: "v1",
	supported_platforms: ["kvm", "vmware"],
	supported_permissions: ["inventory.read", "reports.read"],
	store_backend: "memory",
	persistent_store: false,
	local_operator_session_enabled: true,
} as AboutResponse;

const sampleCurrentTenant = {
	tenant_id: "tenant-a",
	name: "Tenant A",
	active: true,
	role: "admin",
	permissions: ["inventory.read", "migration.manage"],
	auth_method: "service-account",
	service_account_name: "dashboard-e2e",
	service_account_count: 1,
} as CurrentTenant;

const sampleWorkspace = {
	id: "workspace-1",
	name: "Examples Lab Assessment",
	description: "Seeded workspace",
	status: "draft",
	source_connections: [],
	selected_workload_ids: [],
	target_assumptions: {},
	plan_settings: {},
	notes: [],
} as unknown as PilotWorkspace;

const sampleGraph = {
	nodes: [
		{
			id: "vm-1",
			label: "ubuntu-web-01",
			type: "vm",
			platform: "kvm",
			metadata: {},
		},
	] satisfies GraphNode[],
	edges: [],
} as DependencyGraph;

const baseLifecycleState: LifecycleSignalState = {
	costs: [],
	policies: null,
	drift: null,
	remediation: null,
	simulation: null,
	loading: false,
	refreshing: false,
	simulationLoading: false,
	errors: {},
	refresh: vi.fn().mockResolvedValue(undefined),
	simulate: vi.fn().mockResolvedValue(undefined),
};

beforeEach(() => {
	mockUseLifecycleSignals.mockReturnValue(baseLifecycleState);
	mockUseSettingsData.mockReturnValue({
		about: sampleAbout,
		currentTenant: sampleCurrentTenant,
		loading: false,
		errors: {},
	});
	mockGetGraph.mockResolvedValue(sampleGraph);
	mockListWorkspaces.mockResolvedValue([sampleWorkspace]);
});

describe("page state matrix", () => {
	it("renders dashboard loading and empty states accessibly", async () => {
		const loadingView = render(
			<DashboardPage
				inventory={null}
				migrations={[]}
				summary={null}
				latestSnapshot={null}
				loading
				refreshToken={0}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Loading dashboard" }),
		).toBeVisible();
		await expectNoActionableViolations(loadingView.container);
		loadingView.unmount();

		const emptyView = render(
			<DashboardPage
				inventory={null}
				migrations={[]}
				summary={null}
				latestSnapshot={null}
				loading={false}
				refreshToken={0}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "No dashboard data available" }),
		).toBeVisible();
		await expectNoActionableViolations(emptyView.container);
	});

	it("renders inventory loading, error, and empty states accessibly", async () => {
		const loadingView = render(
			<InventoryPage
				inventory={null}
				inventoryPagination={null}
				inventoryPage={1}
				summary={null}
				latestSnapshot={null}
				refreshToken={0}
				loading
				error={null}
				onInventoryPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Loading inventory" }),
		).toBeVisible();
		await expectNoActionableViolations(loadingView.container);
		loadingView.unmount();

		const errorView = render(
			<InventoryPage
				inventory={null}
				inventoryPagination={null}
				inventoryPage={1}
				summary={null}
				latestSnapshot={null}
				refreshToken={0}
				loading={false}
				error="Forced inventory failure"
				onInventoryPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Inventory unavailable" }),
		).toBeVisible();
		await expectNoActionableViolations(errorView.container);
		errorView.unmount();

		const emptyView = render(
			<InventoryPage
				inventory={null}
				inventoryPagination={null}
				inventoryPage={1}
				summary={null}
				latestSnapshot={null}
				refreshToken={0}
				loading={false}
				error={null}
				onInventoryPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "No inventory returned" }),
		).toBeVisible();
		await expectNoActionableViolations(emptyView.container);
	});

	it("renders migrations loading, error, and empty states accessibly", async () => {
		const loadingView = render(
			<MigrationsPage
				inventory={null}
				migrations={[]}
				migrationsPagination={null}
				migrationsPage={1}
				snapshots={[]}
				snapshotsPagination={null}
				snapshotsPage={1}
				summary={sampleSummary}
				latestSnapshot={null}
				refreshToken={0}
				loading
				onMigrationChange={vi.fn()}
				onMigrationsPageChange={vi.fn()}
				onSnapshotsPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Loading migration operations" }),
		).toBeVisible();
		await expectNoActionableViolations(loadingView.container);
		loadingView.unmount();

		const errorView = render(
			<MigrationsPage
				inventory={null}
				migrations={[]}
				migrationsPagination={null}
				migrationsPage={1}
				snapshots={[]}
				snapshotsPagination={null}
				snapshotsPage={1}
				summary={sampleSummary}
				latestSnapshot={null}
				refreshToken={0}
				loading={false}
				migrationError="Forced migration failure"
				onMigrationChange={vi.fn()}
				onMigrationsPageChange={vi.fn()}
				onSnapshotsPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Migration history unavailable" }),
		).toBeVisible();
		await expectNoActionableViolations(errorView.container);
		errorView.unmount();

		const emptyView = render(
			<MigrationsPage
				inventory={sampleInventory}
				migrations={[]}
				migrationsPagination={null}
				migrationsPage={1}
				snapshots={[]}
				snapshotsPagination={null}
				snapshotsPage={1}
				summary={sampleSummary}
				latestSnapshot={sampleSnapshot}
				refreshToken={0}
				loading={false}
				onMigrationChange={vi.fn()}
				onMigrationsPageChange={vi.fn()}
				onSnapshotsPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "No migration activity recorded" }),
		).toBeVisible();
		await expectNoActionableViolations(emptyView.container);
	});

	it("renders lifecycle, policy, and drift states accessibly", async () => {
		mockUseLifecycleSignals.mockReturnValue({
			...baseLifecycleState,
			loading: true,
		});
		const lifecycleLoading = render(
			<LifecyclePage
				summary={sampleSummary}
				latestSnapshot={sampleSnapshot}
				overviewLoading={false}
				refreshToken={0}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Loading lifecycle signals" }),
		).toBeVisible();
		await expectNoActionableViolations(lifecycleLoading.container);
		lifecycleLoading.unmount();

		mockUseLifecycleSignals.mockReturnValue({
			...baseLifecycleState,
			errors: { policies: "Forced policy failure" },
		});
		const policyError = render(<PolicyPage refreshToken={0} />);
		expect(
			screen.getByRole("heading", { name: "Policy evaluation unavailable" }),
		).toBeVisible();
		await expectNoActionableViolations(policyError.container);
		policyError.unmount();

		mockUseLifecycleSignals.mockReturnValue({
			...baseLifecycleState,
			drift: null,
		});
		const driftEmpty = render(
			<DriftPage
				latestSnapshot={null}
				overviewLoading={false}
				refreshToken={0}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "No baseline snapshot available" }),
		).toBeVisible();
		await expectNoActionableViolations(driftEmpty.container);
	});

	it("renders reports and settings states accessibly", async () => {
		const reportsLoading = render(
			<ReportsPage
				migrations={[]}
				migrationsPagination={null}
				migrationsPage={1}
				snapshots={[]}
				snapshotsPagination={null}
				snapshotsPage={1}
				loading
				onMigrationsPageChange={vi.fn()}
				onSnapshotsPageChange={vi.fn()}
			/>,
		);
		expect(
			screen.getByRole("heading", { name: "Loading reports surface" }),
		).toBeVisible();
		await expectNoActionableViolations(reportsLoading.container);
		reportsLoading.unmount();

		mockUseSettingsData.mockReturnValue({
			about: null,
			currentTenant: null,
			loading: false,
			errors: {
				about: {
					message: "Unable to load build metadata.",
					technicalDetails: [],
				},
			},
		});
		const settingsError = render(
			<SettingsPage
				summary={sampleSummary}
				authSourceLabel="Service account key"
				authPersistenceLabel="Remembered in this browser"
			/>,
		);
		expect(
			screen.getByRole("heading", {
				name: "Settings unavailable",
			}),
		).toBeVisible();
		await expectNoActionableViolations(settingsError.container);
	});

	it("renders workspace loading, error, and empty states accessibly", async () => {
		mockListWorkspaces.mockImplementation(
			() => new Promise<PilotWorkspace[]>(() => undefined),
		);
		const workspaceLoading = render(<WorkspacePage />);
		expect(
			await screen.findByRole("heading", { name: "Loading assessments" }),
		).toBeVisible();
		await expectNoActionableViolations(workspaceLoading.container);
		workspaceLoading.unmount();

		mockListWorkspaces.mockRejectedValueOnce(
			new Error("forced workspace failure"),
		);
		const workspaceError = render(<WorkspacePage />);
		expect(
			await screen.findByRole("heading", {
				name: "Assessments unavailable",
			}),
		).toBeVisible();
		await expectNoActionableViolations(workspaceError.container);
		workspaceError.unmount();

		mockListWorkspaces.mockResolvedValueOnce([]);
		const workspaceEmpty = render(<WorkspacePage />);
		expect(
			await screen.findByRole("heading", {
				name: "Create the first assessment",
			}),
		).toBeVisible();
		await expectNoActionableViolations(workspaceEmpty.container);
	});

	it("renders graph loading, error, and empty states accessibly", async () => {
		mockGetGraph.mockImplementation(
			() => new Promise<DependencyGraph>(() => undefined),
		);
		const graphLoading = render(<GraphPage />);
		expect(
			await screen.findByRole("heading", { name: "Loading dependency graph" }),
		).toBeVisible();
		await expectNoActionableViolations(graphLoading.container);
		graphLoading.unmount();

		mockGetGraph.mockRejectedValueOnce(new Error("forced graph failure"));
		const graphError = render(<GraphPage />);
		expect(
			await screen.findByRole("heading", {
				name: "Dependency graph unavailable",
			}),
		).toBeVisible();
		await expectNoActionableViolations(graphError.container);
		graphError.unmount();

		mockGetGraph.mockResolvedValueOnce({ nodes: [], edges: [] });
		const graphEmpty = render(<GraphPage />);
		expect(
			await screen.findByRole("heading", {
				name: "No dependency nodes match the current scope",
			}),
		).toBeVisible();
		await expectNoActionableViolations(graphEmpty.container);
	});
});

async function expectNoActionableViolations(container: HTMLElement) {
	const results = await axe.run(container, {
		runOnly: {
			type: "tag",
			values: ["wcag2a", "wcag2aa"],
		},
		rules: {
			"color-contrast": { enabled: false },
		},
	});
	const actionableViolations = results.violations.filter((violation) =>
		["serious", "critical"].includes(violation.impact ?? ""),
	);
	expect(actionableViolations).toEqual([]);
}
