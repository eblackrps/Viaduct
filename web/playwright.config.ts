import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	testDir: "./tests/e2e",
	testIgnore: "**/runtime-operator.spec.ts",
	snapshotPathTemplate: "{testDir}/__snapshots__/{testFilePath}/{arg}{ext}",
	timeout: 60_000,
	workers: 1,
	expect: {
		timeout: 10_000,
		toHaveScreenshot: {
			caret: "hide",
			maxDiffPixelRatio: 0.01,
			scale: "css",
		},
	},
	fullyParallel: false,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: "http://127.0.0.1:4173",
		trace: "retain-on-failure",
	},
	webServer: {
		command: "npm run build && npm run e2e:serve",
		cwd: __dirname,
		url: "http://127.0.0.1:4173/api/v1/health",
		reuseExistingServer: false,
		timeout: 120_000,
	},
	projects: [
		{
			name: "setup",
			testMatch: "**/auth.setup.ts",
			use: {
				...devices["Desktop Chrome"],
			},
		},
		{
			name: "chromium",
			dependencies: ["setup"],
			testIgnore: ["**/auth.setup.ts", "**/runtime-operator.spec.ts"],
			use: {
				...devices["Desktop Chrome"],
				storageState: "playwright/.auth/dashboard-operator.json",
			},
		},
	],
});
