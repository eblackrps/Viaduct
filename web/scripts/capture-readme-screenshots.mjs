import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawn } from "node:child_process";
import { copyFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(__dirname, "..");
const repoDir = path.resolve(webDir, "..");
const baseURL = "http://127.0.0.1:4173";

const dashboardShots = [
	{
		name: "auth-bootstrap.png",
		capture: captureLocalEntry,
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
		name: "dependency-graph.png",
		capture: captureGraph,
	},
	{
		name: "migration-ops.png",
		capture: captureMigrations,
	},
	{
		name: "reports-history.png",
		capture: captureReports,
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

const siteCopies = [
	"pilot-workspace.png",
	"inventory-assessment.png",
	"dependency-graph.png",
	"reports-history.png",
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
		[
			"run",
			"../tests/e2e/server",
			"-port",
			"4173",
			"-web-dir",
			"dist",
			"-local-runtime",
		],
		{
			cwd: webDir,
			stdio: "inherit",
		},
	);

	try {
		await waitForHealth();

		const chromium = await loadChromium();
		const browser = await chromium.launch({ headless: true });

		for (const shot of dashboardShots) {
			console.log(`Capturing ${shot.name}...`);
			const context = await browser.newContext({
				baseURL,
				viewport: { width: 1880, height: 1100 },
				deviceScaleFactor: 1,
			});
			const page = await context.newPage();
			await page.emulateMedia({ reducedMotion: "reduce" });
			try {
				await shot.capture(page, path.resolve(screenshotDir, shot.name));
			} finally {
				await context.close();
			}
		}

		await browser.close();

		for (const copy of labCopies) {
			const sourcePath = path.resolve(screenshotDir, copy.source);
			await copyFile(sourcePath, copy.target);
		}
		for (const fileName of siteCopies) {
			await copyFile(
				path.resolve(screenshotDir, fileName),
				path.resolve(repoDir, "site", "assets", fileName),
			);
		}
	} finally {
		await stopProcessTree(server.pid);
	}
}

async function login(page) {
	await page.goto("/");
	await page.getByRole("heading", { name: "E2E Lab Workspace" }).waitFor();
}

async function captureLocalEntry(page, targetPath) {
	await login(page);
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureWorkspace(page, targetPath) {
	await login(page);
	await page.goto("/#/workspaces", { waitUntil: "domcontentloaded" });
	await page.getByRole("heading", { name: "E2E Lab Workspace" }).waitFor();
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureInventory(page, targetPath) {
	await login(page);
	await page.goto("/#/inventory", { waitUntil: "domcontentloaded" });
	await page
		.getByRole("heading", { name: "Fleet inventory and assessment" })
		.waitFor();
	await page.getByRole("row", { name: /ubuntu-web-01/i }).click();
	await page
		.getByRole("heading", { name: "Workload assessment table" })
		.scrollIntoViewIfNeeded();
	await page.evaluate(() => window.scrollBy(0, -140));
	await delay(200);
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureMigrations(page, targetPath) {
	await login(page);
	await page.goto("/#/migrations", { waitUntil: "domcontentloaded" });
	await page.getByRole("heading", { name: "Migrations" }).waitFor();
	await page.getByRole("heading", { name: "Planning intake" }).waitFor();
	await page
		.locator("section")
		.filter({
			has: page.getByRole("heading", { name: "Planning intake" }),
		})
		.first()
		.screenshot({ path: targetPath });
}

async function captureGraph(page, targetPath) {
	await login(page);
	await page.goto("/#/graph", { waitUntil: "domcontentloaded" });
	await page
		.getByRole("heading", { name: "Dependency graph", exact: true })
		.waitFor();
	await delay(1500);
	await page.screenshot({ path: targetPath, fullPage: false });
}

async function captureReports(page, targetPath) {
	await login(page);
	await page.goto("/#/reports", { waitUntil: "domcontentloaded" });
	await page.getByRole("heading", { name: "Reports and history" }).waitFor();
	await delay(1500);
	await page.screenshot({ path: targetPath, fullPage: false });
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

	throw new Error(
		"Timed out waiting for the screenshot fixture server to start.",
	);
}

async function loadChromium() {
	const override = process.env.VIADUCT_PLAYWRIGHT_MODULE_PATH;
	if (override) {
		const playwright = await import(pathToFileURL(override).href);
		return playwright.chromium;
	}

	const playwright = await import("playwright");
	return playwright.chromium;
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
