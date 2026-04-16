import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./runtimeAuth", () => ({
	getDashboardAuthMode: () => "tenant",
	getDashboardAuthSession: () => ({
		mode: "tenant",
		apiKey: "tenant-key",
		source: "environment",
	}),
	hasDashboardAuthConfigured: () => true,
}));

import {
	TimeoutError,
	createDashboardAuthSession,
	getAbout,
	getSnapshots,
	requestManager,
} from "./api";
import type { APIError } from "./api";

describe("api", () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	afterEach(() => {
		requestManager.cancelAll();
		vi.useRealTimers();
	});

	it("applies auth headers and pagination query params to GET requests", async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(
				JSON.stringify({
					items: [],
					pagination: { total: 0, page: 2, per_page: 25, total_pages: 0 },
				}),
				{ status: 200 },
			),
		);
		vi.stubGlobal("fetch", fetchMock);

		await getSnapshots({ page: 2, perPage: 25 });

		const [input, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(input).toBe("/api/v2/snapshots?page=2&per_page=25");
		expect(init.credentials).toBe("same-origin");
		expect(new Headers(init.headers).get("X-API-Key")).toBe("tenant-key");
	});

	it("surfaces structured API errors", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(
					JSON.stringify({
						error: {
							code: "invalid_credentials",
							message: "bad credentials",
							request_id: "req-123",
							retryable: false,
							details: {},
							field_errors: [],
						},
					}),
					{ status: 401 },
				),
			),
		);

		await expect(getAbout()).rejects.toMatchObject({
			status: 401,
			code: "invalid_credentials",
			requestID: "req-123",
		} satisfies Partial<APIError>);
	});

	it("throws a TimeoutError when fetch does not complete in time", async () => {
		vi.useFakeTimers();
		vi.stubGlobal(
			"fetch",
			vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
				return new Promise<Response>((_resolve, reject) => {
					init?.signal?.addEventListener("abort", () => {
						reject(new DOMException("Timed out", "AbortError"));
					});
				});
			}),
		);

		const promise = getAbout({ timeoutMs: 5 });
		const assertion = expect(promise).rejects.toBeInstanceOf(TimeoutError);
		await vi.advanceTimersByTimeAsync(5);

		await assertion;
	});

	it("creates a cookie-backed auth session without forwarding existing auth headers", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(
				new Response(
					JSON.stringify({ session_id: "session-123", mode: "tenant" }),
					{ status: 201 },
				),
			);
		vi.stubGlobal("fetch", fetchMock);

		await createDashboardAuthSession("tenant", "new-key", true);

		const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		const headers = new Headers(init.headers);
		expect(headers.get("X-API-Key")).toBeNull();
		expect(JSON.parse(typeof init.body === "string" ? init.body : "")).toEqual({
			mode: "tenant",
			api_key: "new-key",
			remember: true,
		});
	});
});
