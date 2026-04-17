import { beforeEach, describe, expect, it, vi } from "vitest";
import {
	clearDashboardAuthSession,
	getDashboardAuthPersistence,
	getDashboardAuthSession,
	setDashboardAuthSession,
} from "./runtimeAuth";

describe("runtimeAuth", () => {
	beforeEach(() => {
		clearDashboardAuthSession();
		vi.restoreAllMocks();
	});

	it("stores only a session marker for remembered runtime auth", () => {
		setDashboardAuthSession("tenant", {
			apiKey: "tenant-key",
			remember: true,
			sessionID: "session-123",
		});

		expect(window.sessionStorage.getItem("viaduct.dashboardAuth")).toBeNull();
		expect(
			window.localStorage.getItem("viaduct.dashboardAuth.remembered"),
		).toBe(
			JSON.stringify({
				mode: "tenant",
				session_id: "session-123",
			}),
		);
		expect(getDashboardAuthSession()).toEqual({
			mode: "tenant",
			apiKey: "tenant-key",
			source: "runtime",
			sessionID: "session-123",
		});
		expect(getDashboardAuthPersistence()).toBe("local");
	});

	it("clears corrupted storage and warns when parsing fails", () => {
		const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
		window.localStorage.setItem(
			"viaduct.dashboardAuth.remembered",
			"{not-json",
		);

		expect(getDashboardAuthSession()).toEqual({
			mode: "none",
			apiKey: "",
			source: "none",
		});
		expect(
			window.localStorage.getItem("viaduct.dashboardAuth.remembered"),
		).toBeNull();
		expect(warn).toHaveBeenCalledWith(
			"failed to parse dashboard auth session, clearing stored data",
			expect.objectContaining({
				storageKey: "viaduct.dashboardAuth.remembered",
			}),
		);
	});
});
