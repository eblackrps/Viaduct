import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import {
	emptyStorageState,
	login,
	navigateToRoute,
	openAuthenticatedApp,
	operatorRoutes,
} from "./auth";

function expectNoSeriousOrCriticalViolations(
	results: Awaited<ReturnType<AxeBuilder["analyze"]>>,
) {
	const actionableViolations = results.violations.filter(
		(violation) =>
			violation.impact === "critical" || violation.impact === "serious",
	);
	expect(actionableViolations).toEqual([]);
}

test.describe("unauthenticated accessibility", () => {
	test.use({ storageState: emptyStorageState });

	test("keeps the Get started screen free of critical accessibility violations", async ({
		page,
	}) => {
		await page.goto("/");
		await expect(
			page.getByRole("heading", { name: "Get started" }),
		).toBeVisible();

		const results = await new AxeBuilder({ page }).analyze();
		expectNoSeriousOrCriticalViolations(results);
	});
});

test("keeps the authenticated workspace flow free of critical accessibility violations", async ({
	page,
}) => {
	await openAuthenticatedApp(page);

	const results = await new AxeBuilder({ page }).analyze();
	expectNoSeriousOrCriticalViolations(results);
});

test("keeps the top-level operator routes free of serious accessibility violations", async ({
	page,
}) => {
	await openAuthenticatedApp(page);

	for (const route of operatorRoutes) {
		await test.step(route.label, async () => {
			await navigateToRoute(page, route);

			const results = await new AxeBuilder({ page })
				.include("main")
				.analyze();
			expectNoSeriousOrCriticalViolations(results);
		});
	}
});
