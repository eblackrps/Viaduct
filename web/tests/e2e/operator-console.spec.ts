import { expect, test } from "@playwright/test";
import { login, navigateTo } from "./auth";

test("authenticates with the seeded service-account flow", async ({ page }) => {
	await login(page);

	await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();
	await expect(
		page.getByRole("heading", { name: "E2E Lab Workspace" }),
	).toBeVisible();

	const cookies = await page.context().cookies();
	expect(
		cookies.some((cookie) => cookie.name === "viaduct_dashboard_session"),
	).toBeTruthy();
});

test("loads the operational dashboard from the seeded API", async ({
	page,
}) => {
	await login(page);

	await navigateTo(page, "Overview");

	await expect(
		page.getByRole("heading", { name: "Operational dashboard" }),
	).toBeVisible();
	await expect(page.getByText("Workloads")).toBeVisible();
	await expect(page.getByText("Recommendations")).toBeVisible();
	await expect(page.getByText("1 platforms observed")).toBeVisible();
});

test("supports keyboard dismissal and focus restoration for collapsed navigation", async ({
	page,
}) => {
	await login(page);

	const openNavigationButton = page.getByRole("button", {
		name: "Open navigation",
	});
	await expect(openNavigationButton).toHaveAttribute("aria-expanded", "false");
	await openNavigationButton.click();

	const navigationDrawer = page.getByRole("dialog", {
		name: "Primary navigation",
	});
	await expect(navigationDrawer).toBeVisible();
	await expect(openNavigationButton).toHaveAttribute("aria-expanded", "true");
	await expect(
		page.getByRole("button", { name: "Close navigation" }),
	).toBeFocused();

	await page.keyboard.press("Escape");

	await expect(navigationDrawer).toBeHidden();
	await expect(openNavigationButton).toBeFocused();
});

test("renders seeded inventory rows in the workload table", async ({
	page,
}) => {
	await login(page);

	await navigateTo(page, "Inventory");
	await expect(
		page.getByText("Assessment current", { exact: true }),
	).toBeVisible();
	const inventoryTable = page.getByRole("table");
	const inventorySection = page
		.locator("section")
		.filter({
			has: page.getByRole("heading", { name: "Workload assessment table" }),
		})
		.first();
	const inventoryPlanButton = inventorySection
		.getByRole("button", { name: "Open migration plan" })
		.first();

	await expect(
		page.getByRole("heading", { name: "Fleet inventory and assessment" }),
	).toBeVisible();
	await expect(inventoryTable.getByText("ubuntu-web-01")).toBeVisible();
	await expect(inventoryTable.getByText("windows-app-01")).toBeVisible();
	await expect(inventoryPlanButton).toBeVisible();
	await expect(inventoryPlanButton).toBeDisabled();
});

test("reveals workload detail from the mobile inventory card flow", async ({
	page,
}) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await login(page);

	await navigateTo(page, "Inventory");
	const detailHeading = page.getByRole("heading", { name: "Workload detail" });
	await expect(detailHeading).not.toBeInViewport();
	await page.getByRole("button", { name: "Inspect workload" }).first().click();
	await expect(detailHeading).toBeInViewport();
	await expect(
		page.getByRole("button", { name: "Open migration plan" }).last(),
	).toBeVisible();
});

test("creates a migration plan from the real offline KVM fixtures", async ({
	page,
}) => {
	await login(page);

	await navigateTo(page, "Inventory");
	await expect(
		page.getByText("Assessment current", { exact: true }),
	).toBeVisible();
	const inventoryTable = page.getByRole("table");
	const inventorySection = page
		.locator("section")
		.filter({
			has: page.getByRole("heading", { name: "Workload assessment table" }),
		})
		.first();
	const inventoryPlanButton = inventorySection
		.getByRole("button", { name: "Open migration plan" })
		.first();
	const ubuntuCheckbox = inventoryTable.getByRole("checkbox", {
		name: "Select ubuntu-web-01",
	});
	await ubuntuCheckbox.check();
	await expect(ubuntuCheckbox).toBeChecked();
	await expect(inventoryPlanButton).toBeEnabled();
	await inventoryPlanButton.click();

	await expect(page.getByRole("heading", { name: "Migrations" })).toBeVisible();
	await expect(page.getByText("Imported from inventory").first()).toBeVisible();

	await page.getByLabel("Source address").fill("examples/lab/kvm");
	await page.getByRole("button", { name: "Continue", exact: true }).click();

	await page.getByLabel("Target platform").selectOption("kvm");
	await page.getByLabel("Target address").fill("examples/lab/kvm");
	await page.getByRole("button", { name: "Continue", exact: true }).click();

	await page
		.getByRole("button", { name: "Run preflight", exact: true })
		.click();
	await expect(page.getByText("Blocking checks", { exact: true })).toBeVisible();

	await page
		.getByRole("button", { name: "Save migration plan", exact: true })
		.click();

	const savedPlanSection = page
		.locator("section")
		.filter({
			has: page.getByRole("heading", { name: "Saved plan state" }),
		})
		.first();
	await expect(
		page.getByRole("heading", { name: "Saved plan state" }),
	).toBeVisible();
	await expect(savedPlanSection.getByText("Migration ID")).toBeVisible();
	await expect(
		savedPlanSection.getByText(/dashboard-migration|migration-/).first(),
	).toBeVisible();
});

	test("saves a focused workload from the workspace detail panel", async ({
	page,
}) => {
	await login(page);

	await page.getByRole("row", { name: /ubuntu-web-01/i }).click();
	const detailSaveButton = page
		.getByRole("button", {
			name: "Save selection",
		})
		.last();

	await detailSaveButton.click();
	await expect(
		page.getByRole("checkbox", { name: "Select ubuntu-web-01" }),
	).toBeChecked();
});

test("shows the workspace error state when the workspace list fails", async ({
	page,
}) => {
	await page.route("**/api/v1/workspaces", async (route) => {
		await route.fulfill({
			status: 500,
			contentType: "application/json",
			body: JSON.stringify({
				error: {
					code: "internal_error",
					message: "forced workspace failure for E2E coverage",
					request_id: "req-e2e-workspaces",
					retryable: true,
					details: {},
					field_errors: [],
				},
			}),
		});
	});

	await login(page, { landingHeading: "Pilot workspaces unavailable" });

	await expect(
		page.getByRole("heading", { name: "Pilot workspaces unavailable" }),
	).toBeVisible();
	await expect(
		page.getByText(/forced workspace failure for E2E coverage/i),
	).toBeVisible();
	await expect(page.getByRole("button", { name: "Retry" })).toBeVisible();
});
