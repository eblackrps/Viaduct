import { expect, type Page } from "@playwright/test";
import { operatorRoutes, type OperatorRouteExpectation } from "./routes";

export const serviceAccountKey = "sa-e2e-key";
export const emptyStorageState = { cookies: [], origins: [] };
export { operatorRoutes };

export async function login(
	page: Page,
	options?: { landingHeading?: string | RegExp },
) {
	await page.goto("/");
	const landingHeading = options?.landingHeading
		? page.getByRole("heading", {
				name: options.landingHeading,
			})
		: null;
	const bootstrapHeading = page.getByRole("heading", {
		name: "Get started",
	});
	const signOutButton = page.getByRole("button", { name: "Sign out" });
	await expect
		.poll(
			async () => {
				if (await bootstrapHeading.isVisible().catch(() => false)) {
					return "bootstrap";
				}
				if (await appShellReady(page)) {
					return "authenticated";
				}
				return "pending";
			},
			{
				timeout: 10_000,
				message:
					"expected either the authenticated dashboard shell or the Get started screen",
			},
		)
		.not.toBe("pending");
	const authState = (await bootstrapHeading.isVisible().catch(() => false))
		? "bootstrap"
		: "authenticated";
	if (authState === "bootstrap") {
		await expect(bootstrapHeading).toBeVisible();
		const useKeyButton = page.getByRole("button", { name: "Use a key instead" });
		if (await useKeyButton.isVisible().catch(() => false)) {
			await useKeyButton.click();
		}
		await page.getByLabel("Paste your key").fill(serviceAccountKey);
		await page.getByRole("button", { name: "Start session" }).click();
	}
	if (landingHeading) {
		await expect(landingHeading).toBeVisible();
	} else {
		await expect.poll(() => appShellReady(page)).toBe(true);
	}
	if (await signOutButton.isVisible().catch(() => false)) {
		await expect(signOutButton).toBeVisible();
	}
}

export async function openAuthenticatedApp(page: Page) {
	await page.goto("/");
	await expect.poll(() => appShellReady(page)).toBe(true);
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

	await page
		.getByRole("navigation", { name: "Primary" })
		.getByRole("link", {
			name: new RegExp(`^${escapeForRegex(label)}(?:\\s|$)`),
		})
		.click();
}

export async function navigateToRoute(
	page: Page,
	route: OperatorRouteExpectation,
) {
	await navigateTo(page, route.label);
	await expect(
		page.getByRole("heading", { name: route.heading, exact: true }),
	).toBeVisible();
	if (route.readyHeading) {
		await expect(
			page.getByRole("heading", { name: route.readyHeading, exact: true }),
		).toBeVisible();
	}
}

async function appShellReady(page: Page) {
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
			.getByRole("button", { name: "Refresh" })
			.first()
			.isVisible()
			.catch(() => false))
	);
}

function escapeForRegex(value: string) {
	return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
