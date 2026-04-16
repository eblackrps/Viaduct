import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { login } from "./auth";

test("keeps the auth bootstrap screen free of critical accessibility violations", async ({
	page,
}) => {
	await page.goto("/");
	await expect(
		page.getByRole("heading", { name: "Connect the Viaduct dashboard" }),
	).toBeVisible();

	const results = await new AxeBuilder({ page }).analyze();
	expect(results.violations).toEqual([]);
});

test("keeps the authenticated workspace flow free of critical accessibility violations", async ({
	page,
}) => {
	await login(page);

	const results = await new AxeBuilder({ page }).analyze();
	expect(results.violations).toEqual([]);
});
