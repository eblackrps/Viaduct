import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import { copyFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";
import { chromium } from "playwright";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(__dirname, "..");
const repoDir = path.resolve(webDir, "..");
const baseURL = "http://127.0.0.1:4173";
const serviceAccountKey = "sa-e2e-key";

const dashboardShots = [
	{
		name: "auth-bootstrap.png",
		capture: captureAuthBootstrap,
	},
	{
		name: "pilot-workspace.png",
		capture: captureWorkspace,
	},
	{
		name: "inventory-assessment.png",
		capture: captureInventory,
	},
	{
		name: "migration-ops.png",
		capture: captureMigrations,
	},
];

const labCopies = [
	{
		source: "pilot-workspace.png",
		target: path.resolve(
			repoDir,
			"examples",
			"lab",
			"screenshots",
			"lab-workspace-overview.png",
		),
	},
	{
		source: "migration-ops.png",
		target: path.resolve(
			repoDir,
			"examples",
			"lab",
			"screenshots",
			"lab-migration-ops.png",
		),
	},
];

const screenshotDir = path.resolve(
	repoDir,
	"docs",
	"operations",
	"demo",
	"screenshots",
);

async function main() {
	const server = spawn(
		"go",
		["run", "../tests/e2e/server", "-port", "4173", "-web-dir", "dist"],
		{
			cwd: webDir,
			stdio: "inherit",
		},
	);

	try {
		await waitForHealth();

		const browser = await chromium.launch({ headless: true });
		const context = await browser.newContext({
			baseURL,
			viewport: { width: 1880, height: 1100 },
			deviceScaleFactor: 1,
		});
		const page = await context.newPage();
		await page.emulateMedia({ reducedMotion: "reduce" });

		for (const shot of dashboardShots) {
			await shot.capture(page, path.resolve(screenshotDir, shot.name));
		}

		await browser.close();

		for (const copy of labCopies) {
			const sourcePath = path.resolve(screenshotDir, copy.source);
			await copyFile(sourcePath, copy.target);
		}
	} finally {
		await stopProcessTree(server.pid);
	}
}

async function captureAuthBootstrap(page, targetPath) {
	await page.goto("/");
	await page.getByRole("heading", { name: "Connect the Viaduct dashboard" }).waitFor();
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function login(page) {
	await page.goto("/");
	await page.getByLabel("Service-account key").fill(serviceAccountKey);
	await page.getByRole("button", { name: "Connect dashboard" }).click();
	await page.getByRole("heading", { name: "E2E Lab Workspace" }).waitFor();
}

async function captureWorkspace(page, targetPath) {
	await login(page);
	await page.goto("/#/workspaces");
	await page.getByRole("heading", { name: "E2E Lab Workspace" }).waitFor();
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureInventory(page, targetPath) {
	await page.goto("/#/inventory");
	await page.getByRole("heading", { name: "Fleet inventory and assessment" }).waitFor();
	await page.getByRole("row", { name: /ubuntu-web-01/i }).click();
	await page
		.getByRole("heading", { name: "Workload assessment table" })
		.scrollIntoViewIfNeeded();
	await page.evaluate(() => window.scrollBy(0, -140));
	await delay(200);
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureMigrations(page, targetPath) {
	await page.goto("/#/migrations");
	await page.getByRole("heading", { name: "Migrations" }).waitFor();
	await page
		.locator("section")
		.filter({
			has: page.getByRole("heading", { name: "Planning intake" }),
		})
		.first()
		.screenshot({ path: targetPath });
}

async function waitForHealth() {
	for (let attempt = 0; attempt < 60; attempt += 1) {
		try {
			const response = await fetch(`${baseURL}/api/v1/health`);
			if (response.ok) {
				return;
			}
		} catch {}

		await delay(1000);
	}

	throw new Error("Timed out waiting for the screenshot fixture server to start.");
}

async function stopProcessTree(pid) {
	if (!pid) {
		return;
	}

	if (process.platform === "win32") {
		await new Promise((resolve) => {
			const killer = spawn("taskkill", ["/PID", String(pid), "/T", "/F"], {
				stdio: "ignore",
			});
			killer.on("exit", () => resolve());
			killer.on("error", () => resolve());
		});
		return;
	}

	try {
		process.kill(pid, "SIGTERM");
	} catch {}
}

main().catch((error) => {
	console.error(error);
	process.exitCode = 1;
});
