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
	await expect(
		page.getByRole("heading", { name: "Start the workspace-first operator flow" }),
	).toBeVisible();

	await page.getByRole("button", { name: "Create workspace" }).click();
	await expect(
		page.getByRole("heading", { name: "Examples Lab Assessment" }),
	).toBeVisible();

	await page.getByRole("button", { name: "Run discovery" }).click();
	await expect(page.getByText("1 snapshot(s) recorded.")).toBeVisible({
		timeout: 30_000,
	});
	await expect(
		page.getByRole("heading", { name: "Workload assessment table" }),
	).toBeVisible();

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
