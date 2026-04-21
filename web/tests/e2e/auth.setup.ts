import { expect, test as setup } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { serviceAccountKey } from "./auth";

const authFile = "playwright/.auth/dashboard-operator.json";

setup("authenticates the seeded dashboard operator", async ({ page }) => {
	mkdirSync("playwright/.auth", { recursive: true });
	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Connect the Viaduct dashboard" }),
	).toBeVisible();

	await page.getByLabel("Service-account key").fill(serviceAccountKey);
	await page.getByRole("button", { name: "Connect dashboard" }).click();
	await expect(
		page.getByRole("heading", { name: "E2E Lab Workspace" }),
	).toBeVisible();
	await page.context().storageState({ path: authFile });
});
