import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
	testDir: "./tests/e2e",
	testMatch: "**/runtime-operator.spec.ts",
	timeout: 90_000,
	workers: 1,
	expect: {
		timeout: 10_000,
	},
	fullyParallel: false,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: "http://127.0.0.1:4174",
		trace: "retain-on-failure",
	},
	webServer: {
		command: "npm run build && node scripts/start-runtime-smoke.mjs",
		cwd: __dirname,
		url: "http://127.0.0.1:4175/readyz",
		reuseExistingServer: false,
		timeout: 60_000,
	},
	projects: [
		{
			name: "chromium",
			use: {
				...devices["Desktop Chrome"],
			},
		},
	],
});
