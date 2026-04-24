import { expect, test } from "@playwright/test";
import { navigateTo } from "./auth";

test("boots the real viaduct runtime and completes the first local operator flow", async ({
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

	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Get started" }),
	).toBeVisible();
	await expect(
		page.getByRole("button", { name: "Start local session" }),
	).toBeVisible();

	const docsResponse = await page.request.get("/api/v1/docs/swagger.json");
	expect(docsResponse.ok()).toBeTruthy();

	await page.getByRole("button", { name: "Start local session" }).click();
	await expect
		.poll(async () => {
			if (
				await page
					.getByRole("button", { name: "Create workspace" })
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
		name: "Create workspace",
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
	await expect(
		page.getByRole("button", { name: "Export report" }),
	).toBeVisible({
		timeout: 30_000,
	});
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
	await expect(
		page.getByRole("heading", { name: "Operational dashboard" }),
	).toBeVisible();

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

test("keeps the legacy v1 inventory response shape available", async ({ page }) => {
	await page.goto("/");

	const localSessionButton = page.getByRole("button", {
		name: "Start local session",
	});
	if (await localSessionButton.isVisible()) {
		await localSessionButton.click();
	}
	await expect
		.poll(async () => {
			const response = await page.request.get("/api/v1/tenants/current");
			return response.status();
		})
		.toBe(200);

	const response = await page.request.get("/api/v1/inventory");
	expect(response.ok()).toBeTruthy();

	const payload = await response.json();
	expect(Array.isArray(payload)).toBeFalsy();
	expect(Array.isArray(payload?.vms)).toBeTruthy();
	expect(payload?.inventory).toBeUndefined();
	expect(payload?.pagination).toBeUndefined();
});
