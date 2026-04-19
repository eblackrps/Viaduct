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
	createRequestController,
	createDashboardAuthSession,
	getActiveRequestControllerCount,
	getAbout,
	getCosts,
	getDrift,
	getRemediation,
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

	it("creates a local runtime auth session without sending an api key", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(
				new Response(
					JSON.stringify({ session_id: "session-456", mode: "local" }),
					{ status: 201 },
				),
			);
		vi.stubGlobal("fetch", fetchMock);

		await createDashboardAuthSession("local", "ignored-key", true);

		const [input, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(input).toBe("/api/v1/auth/session");
		expect(JSON.parse(typeof init.body === "string" ? init.body : "")).toEqual({
			mode: "local",
			remember: true,
		});
	});

	it("creates a shared request controller that times out and cleans up", async () => {
		vi.useFakeTimers();

		const requestController = createRequestController({ timeoutMs: 5 });
		const abortListener = vi.fn();
		requestController.signal.addEventListener("abort", abortListener);

		await vi.advanceTimersByTimeAsync(5);

		expect(abortListener).toHaveBeenCalledTimes(1);
		expect(requestController.timedOut()).toBe(true);

		requestController.cleanup();
	});

	it("encodes operator query params with URLSearchParams", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(new Response(JSON.stringify([]), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200 }));
		vi.stubGlobal("fetch", fetchMock);

		await getCosts("vmware & kvm");
		await getDrift("baseline east/west");
		await getRemediation("north+south");

		const costsURL = new URL(
			fetchMock.mock.calls[0][0] as string,
			window.location.origin,
		);
		expect(costsURL.pathname).toBe("/api/v1/costs");
		expect(costsURL.searchParams.get("platform")).toBe("vmware & kvm");

		const driftURL = new URL(
			fetchMock.mock.calls[1][0] as string,
			window.location.origin,
		);
		expect(driftURL.pathname).toBe("/api/v1/drift");
		expect(driftURL.searchParams.get("baseline")).toBe("baseline east/west");

		const remediationURL = new URL(
			fetchMock.mock.calls[2][0] as string,
			window.location.origin,
		);
		expect(remediationURL.pathname).toBe("/api/v1/remediation");
		expect(remediationURL.searchParams.get("baseline")).toBe("north+south");
	});

	it("does not dedupe concurrent GET requests with different query strings", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(
				new Response(JSON.stringify({ items: [] }), { status: 200 }),
			);
		vi.stubGlobal("fetch", fetchMock);

		const pageOnePromise = requestManager.fetch("/api/v2/inventory?page=1");
		const pageTwoPromise = requestManager.fetch("/api/v2/inventory?page=2");

		expect(pageOnePromise).not.toBe(pageTwoPromise);

		await Promise.all([pageOnePromise, pageTwoPromise]);
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it("does not leak request controllers when request URL normalization throws", async () => {
		await expect(
			requestManager.fetch({ url: "http://%" } as Request),
		).rejects.toThrow();
		expect(getActiveRequestControllerCount()).toBe(0);
	});
});
