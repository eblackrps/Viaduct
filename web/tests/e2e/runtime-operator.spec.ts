import { expect, test, type Page } from "@playwright/test";
import { navigateTo } from "./auth";

test("boots the real viaduct runtime and completes the first local session flow", async ({
	page,
}) => {
	const listRequests = new Set<string>();
	page.on("request", (request) => {
		if (request.method() !== "GET") {
			return;
		}
		const url = new URL(request.url());
		if (
			url.pathname === "/api/v1/inventory" ||
			url.pathname === "/api/v1/snapshots" ||
			url.pathname === "/api/v1/migrations" ||
			url.pathname === "/api/v2/inventory" ||
			url.pathname === "/api/v2/snapshots" ||
			url.pathname === "/api/v2/migrations"
		) {
			listRequests.add(url.pathname);
		}
	});

	await ensureRuntimeLocalSession(page);

	const docsResponse = await page.request.get("/api/v1/docs/swagger.json");
	expect(docsResponse.ok()).toBeTruthy();

	await expect
		.poll(async () => {
			if (
				await page
					.getByRole("button", { name: "Create assessment" })
					.isVisible()
					.catch(() => false)
			) {
				return "intake";
			}
			if (
				await page
					.getByRole("heading", { name: "Examples Lab Assessment" })
					.isVisible()
					.catch(() => false)
			) {
				return "workspace";
			}
			return "pending";
		})
		.not.toBe("pending");

	const createWorkspaceButton = page.getByRole("button", {
		name: "Create assessment",
	});
	if (await createWorkspaceButton.isVisible().catch(() => false)) {
		await createWorkspaceButton.click();
		await expect(
			page.getByRole("heading", { name: "Examples Lab Assessment" }),
		).toBeVisible();
	}

	await page.getByRole("button", { name: "Run discovery" }).click();
	await expect(page.getByText("1 snapshot(s) recorded.")).toBeVisible({
		timeout: 30_000,
	});
	await expect(
		page.getByRole("heading", { name: "Workload assessment table" }),
	).toBeVisible();
	const workloadCheckbox = page.getByRole("checkbox", {
		name: "Select ubuntu-web-01",
	});
	await workloadCheckbox.check();
	await expect(workloadCheckbox).toBeChecked();

	const buildGraphButton = page.getByRole("button", { name: "Build graph" });
	await expect(buildGraphButton).toBeVisible();
	await buildGraphButton.click();
	await expect(
		page.getByRole("button", { name: "Run simulation" }),
	).toBeVisible({
		timeout: 30_000,
	});

	await page.getByRole("button", { name: "Run simulation" }).click();
	await expect(page.getByRole("button", { name: "Save plan" })).toBeVisible({
		timeout: 30_000,
	});

	await page.getByRole("button", { name: "Save plan" }).click();
	await expect(page.getByRole("button", { name: "Export report" })).toBeVisible(
		{
			timeout: 30_000,
		},
	);
	await expect(page.getByText(/Saved plan/i)).toBeVisible();

	const downloadPromise = page.waitForEvent("download");
	await page.getByRole("button", { name: "Export report" }).click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toMatch(
		/^examples-lab-assessment-[a-z0-9]{8}\.md$/,
	);
	await expect(page.getByText("Latest export")).toBeVisible({
		timeout: 30_000,
	});

	await navigateTo(page, "Overview");
	await expect(page.getByRole("heading", { name: "Dashboard" })).toBeVisible();

	expect(listRequests.has("/api/v2/inventory")).toBeTruthy();
	expect(listRequests.has("/api/v2/snapshots")).toBeTruthy();
	expect(listRequests.has("/api/v2/migrations")).toBeTruthy();
	expect(listRequests.has("/api/v1/inventory")).toBeFalsy();
	expect(listRequests.has("/api/v1/snapshots")).toBeFalsy();
	expect(listRequests.has("/api/v1/migrations")).toBeFalsy();

	const cookies = await page.context().cookies();
	expect(
		cookies.some((cookie) => cookie.name === "viaduct_dashboard_session"),
	).toBeTruthy();
});

test("keeps the legacy v1 inventory response shape available", async ({
	page,
}) => {
	await ensureRuntimeLocalSession(page);
	await expect
		.poll(async () => {
			const response = await page.request.get("/api/v1/tenants/current");
			return response.status();
		})
		.toBe(200);

	const workspace = await createRuntimeWorkspace(page);
	await runWorkspaceJob(page, workspace.id, "discovery");

	const response = await page.request.get("/api/v1/inventory");
	expect(response.ok()).toBeTruthy();

	const payload = await response.json();
	expect(Array.isArray(payload)).toBeFalsy();
	expect(Array.isArray(payload?.vms)).toBeTruthy();
	expect(payload?.inventory).toBeUndefined();
	expect(payload?.pagination).toBeUndefined();
});

async function ensureRuntimeLocalSession(page: Page) {
	await page.goto("/");
	const bootstrapHeading = page.getByRole("heading", { name: "Get started" });
	const localSessionButton = page.getByRole("button", {
		name: "Start local session",
	});

	await expect
		.poll(
			async () => {
				if (await localSessionButton.isVisible().catch(() => false)) {
					return "bootstrap";
				}
				if (await runtimeShellReady(page)) {
					return "authenticated";
				}
				if (await bootstrapHeading.isVisible().catch(() => false)) {
					return "bootstrap";
				}
				return "pending";
			},
			{
				timeout: 15_000,
				message:
					"expected either the keyless runtime shell or the local session bootstrap",
			},
		)
		.not.toBe("pending");

	if (await localSessionButton.isVisible().catch(() => false)) {
		await localSessionButton.click();
	}
	await expect.poll(() => runtimeShellReady(page)).toBe(true);
}

async function runtimeShellReady(page: Page) {
	return (
		(await page
			.getByRole("navigation", { name: "Primary" })
			.isVisible()
			.catch(() => false)) ||
		(await page
			.getByRole("button", { name: "Open navigation" })
			.isVisible()
			.catch(() => false)) ||
		(await page
			.getByRole("button", { name: "Create assessment" })
			.isVisible()
			.catch(() => false))
	);
}

async function createRuntimeWorkspace(page: Page) {
	const response = await page.request.post("/api/v1/workspaces", {
		data: {
			id: `runtime-inventory-${Date.now()}`,
			name: "Runtime Inventory Shape",
			description: "Runtime smoke inventory shape seed",
			source_connections: [
				{
					id: "lab-kvm",
					name: "Lab KVM",
					platform: "kvm",
					address: "examples/lab/kvm",
				},
			],
			target_assumptions: {
				platform: "proxmox",
				address: "https://pilot-proxmox.local:8006/api2/json",
				default_host: "pve-01",
				default_storage: "local-lvm",
				default_network: "vmbr0",
			},
			plan_settings: {
				name: "runtime-inventory-shape",
				parallel: 2,
				wave_size: 2,
			},
		},
	});
	expect(response.ok()).toBeTruthy();
	return (await response.json()) as { id: string };
}

async function runWorkspaceJob(page: Page, workspaceID: string, type: string) {
	const createResponse = await page.request.post(
		`/api/v1/workspaces/${workspaceID}/jobs`,
		{
			data: {
				type,
				requested_by: "runtime-smoke",
			},
		},
	);
	expect(createResponse.ok()).toBeTruthy();
	const job = (await createResponse.json()) as { id: string };
	await expect
		.poll(async () => {
			const response = await page.request.get(
				`/api/v1/workspaces/${workspaceID}/jobs/${job.id}`,
			);
			if (!response.ok()) {
				return `http-${response.status()}`;
			}
			const current = (await response.json()) as {
				status: string;
				error?: string;
			};
			if (current.status === "failed") {
				return current.error || "failed";
			}
			return current.status;
		})
		.toBe("succeeded");
}
