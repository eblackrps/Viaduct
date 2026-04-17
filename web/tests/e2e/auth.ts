import { expect, type Page } from "@playwright/test";

export const serviceAccountKey = "sa-e2e-key";

export async function login(
	page: Page,
	options?: { landingHeading?: string | RegExp },
) {
	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Connect the Viaduct dashboard" }),
	).toBeVisible();

	await page.getByLabel("Service-account key").fill(serviceAccountKey);
	await page.getByRole("button", { name: "Connect dashboard" }).click();
	await expect(
		page.getByRole("heading", {
			name: options?.landingHeading ?? "E2E Lab Workspace",
		}),
	).toBeVisible();
	await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();
}

export async function navigateTo(page: Page, label: string) {
	const openNavigationButton = page.getByRole("button", {
		name: "Open navigation",
	});
	if (await openNavigationButton.isVisible()) {
		await openNavigationButton.click();
		await expect(
			page.getByRole("dialog", { name: "Primary navigation" }),
		).toBeVisible();
	}

	await page.getByRole("link", { name: label }).click();
}
