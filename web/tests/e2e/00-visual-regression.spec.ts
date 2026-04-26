import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import {
	navigateToRoute,
	openAuthenticatedApp,
	operatorRoutes,
} from "./auth";

test.skip(
	process.platform !== "linux",
	"visual regression baselines are recorded against the Linux CI renderer",
);

const desktopRoutes = operatorRoutes;
const mobileRoutes = operatorRoutes.slice(0, 5);

async function captureRoute(page: Page, label: string) {
	await page.locator("body").evaluate((element) => {
		(element as HTMLElement).style.scrollBehavior = "auto";
	});
	await expect(page).toHaveScreenshot(label, {
		animations: "disabled",
		fullPage: false,
		maxDiffPixelRatio: 0.08,
	});
}

for (const route of desktopRoutes) {
	test(`captures desktop visual regression for ${route.label}`, async ({
		page,
	}) => {
		await page.setViewportSize({ width: 1440, height: 1200 });
		await openAuthenticatedApp(page);
		await navigateToRoute(page, route);
		await captureRoute(page, `${route.snapshotName}-desktop.png`);
	});
}

for (const route of mobileRoutes) {
	test(`captures mobile visual regression for ${route.label}`, async ({
		page,
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await openAuthenticatedApp(page);
		await navigateToRoute(page, route);
		await captureRoute(page, `${route.snapshotName}-mobile.png`);
	});
}
