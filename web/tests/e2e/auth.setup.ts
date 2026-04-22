import { expect, test as setup } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { serviceAccountKey } from "./auth";

const authFile = "playwright/.auth/dashboard-operator.json";

setup("authenticates the seeded dashboard operator", async ({ page }) => {
	mkdirSync("playwright/.auth", { recursive: true });
	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Get started" }),
	).toBeVisible();

	const useKeyButton = page.getByRole("button", { name: "Use a key instead" });
	if (await useKeyButton.isVisible().catch(() => false)) {
		await useKeyButton.click();
	}
	await page.getByLabel("Paste your key").fill(serviceAccountKey);
	await page.getByRole("button", { name: "Start session" }).click();
	await expect(
		page.getByRole("heading", { name: "E2E Lab Workspace" }),
	).toBeVisible();
	await page.context().storageState({ path: authFile });
});
