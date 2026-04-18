import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { mkdtempSync, rmSync } from "node:fs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webDir = path.resolve(__dirname, "..");
const repoDir = path.resolve(webDir, "..");
const runtimeDir = mkdtempSync(path.join(os.tmpdir(), "viaduct-runtime-e2e-"));
const configPath = path.join(runtimeDir, "config.yaml");

let shuttingDown = false;
const child = spawn(
	"go",
	[
		"run",
		"./cmd/viaduct",
		"start",
		"--config",
		configPath,
		"--port",
		"4174",
		"--host",
		"127.0.0.1",
		"--open-browser=false",
	],
	{
		cwd: repoDir,
		stdio: "inherit",
		env: process.env,
	},
);

console.log(`Viaduct runtime smoke config: ${configPath}`);

child.on("error", (error) => {
	console.error("failed to start the Viaduct runtime smoke server", error);
	process.exitCode = 1;
	void shutdown();
});

child.on("exit", (code, signal) => {
	if (!shuttingDown) {
		console.error(
			`Viaduct runtime smoke server exited unexpectedly (code=${code ?? "null"}, signal=${signal ?? "none"}).`,
		);
		process.exitCode = code ?? 1;
	}
	cleanupRuntimeDir();
});

for (const signal of ["SIGINT", "SIGTERM"]) {
	process.on(signal, () => {
		void shutdown(signal);
	});
}

async function shutdown(signal) {
	if (shuttingDown) {
		return;
	}
	shuttingDown = true;
	await stopProcessTree(child.pid);
	cleanupRuntimeDir();
	if (signal) {
		process.exit(0);
	}
}

function cleanupRuntimeDir() {
	try {
		rmSync(runtimeDir, { recursive: true, force: true });
	} catch {}
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
			killer.on("exit", resolve);
			killer.on("error", resolve);
		});
		return;
	}

	try {
		process.kill(pid, "SIGTERM");
	} catch {
		return;
	}
}
