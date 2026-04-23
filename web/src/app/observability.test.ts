import { afterEach, describe, expect, it, vi } from "vitest";
import { reportFrontendError } from "./observability";

describe("reportFrontendError", () => {
	afterEach(() => {
		delete window.__VIADUCT_FRONTEND_MONITOR__;
	});

	it("emits a browser event and forwards the payload to the optional monitor hook", async () => {
		const capture = vi.fn();
		window.__VIADUCT_FRONTEND_MONITOR__ = { capture };

		const eventPromise = new Promise<CustomEvent>((resolve) => {
			window.addEventListener(
				"viaduct:frontend-error",
				(event) => resolve(event as CustomEvent),
				{ once: true },
			);
		});

		reportFrontendError({
			type: "error",
			message: "dashboard boom",
			timestamp: "2026-04-23T00:00:00Z",
		});

		const event = await eventPromise;
		expect(event.detail).toEqual({
			type: "error",
			message: "dashboard boom",
			timestamp: "2026-04-23T00:00:00Z",
		});
		expect(capture).toHaveBeenCalledWith({
			type: "error",
			message: "dashboard boom",
			timestamp: "2026-04-23T00:00:00Z",
		});
	});
});
