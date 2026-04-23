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
	downloadReport,
	getActiveRequestControllerCount,
	getAbout,
	getCosts,
	getDrift,
	getInflightDedupeCount,
	getInflightRequestCount,
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

		const [input, init] = fetchMock.mock.calls[0] as [URL, RequestInit];
		expect(`${input.pathname}${input.search}`).toBe(
			"/api/v2/snapshots?page=2&per_page=25",
		);
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
		expect(headers.get("Content-Type")).toBe("application/json");
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

		const [input, init] = fetchMock.mock.calls[0] as [URL, RequestInit];
		expect(`${input.pathname}${input.search}`).toBe("/api/v1/auth/session");
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

		const pageOnePromise = requestManager.fetch(
			"/api/v2/inventory?page=1",
			undefined,
			{ dedupe: true },
		);
		const pageTwoPromise = requestManager.fetch(
			"/api/v2/inventory?page=2",
			undefined,
			{ dedupe: true },
		);

		expect(pageOnePromise).not.toBe(pageTwoPromise);

		await Promise.all([pageOnePromise, pageTwoPromise]);
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it("dedupes identical GET requests and releases the inflight entry when they settle", async () => {
		let resolveFetch: ((response: Response) => void) | undefined;
		const fetchMock = vi.fn().mockImplementation(
			() =>
				new Promise<Response>((resolve) => {
					resolveFetch = resolve;
				}),
		);
		vi.stubGlobal("fetch", fetchMock);

		const firstPromise = requestManager.fetch("/api/v1/about", undefined, {
			dedupe: true,
		});
		const secondPromise = requestManager.fetch("/api/v1/about", undefined, {
			dedupe: true,
		});
		const thirdPromise = requestManager.fetch("/api/v1/about", undefined, {
			dedupe: true,
		});

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(getInflightRequestCount()).toBe(1);
		expect(getInflightDedupeCount()).toBe(1);

		resolveFetch?.(
			new Response(JSON.stringify({ version: "3.1.1" }), { status: 200 }),
		);

		await Promise.all([thirdPromise, secondPromise, firstPromise]);
		expect(getInflightRequestCount()).toBe(0);
		expect(getInflightDedupeCount()).toBe(0);
	});

	it("TestDedupeRefCountingClean", async () => {
		let resolveFetch: ((response: Response) => void) | undefined;
		const fetchMock = vi.fn().mockImplementation(
			() =>
				new Promise<Response>((resolve) => {
					resolveFetch = resolve;
				}),
		);
		vi.stubGlobal("fetch", fetchMock);

		const firstBatch = Array.from({ length: 25 }, () =>
			requestManager.fetch("/api/v1/about", undefined, { dedupe: true }),
		);

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(getInflightRequestCount()).toBe(1);
		expect(getInflightDedupeCount()).toBe(1);

		resolveFetch?.(
			new Response(JSON.stringify({ version: "3.1.1" }), { status: 200 }),
		);

		const secondBatch = Array.from({ length: 25 }, () =>
			requestManager.fetch("/api/v1/about", undefined, { dedupe: true }),
		);

		await Promise.all([...firstBatch, ...secondBatch]);
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(getInflightRequestCount()).toBe(0);
		expect(getInflightDedupeCount()).toBe(0);
	});

	it("resolves request URLs against the base tag when present", async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(
				new Response(JSON.stringify({ version: "3.1.1" }), { status: 200 }),
			);
		vi.stubGlobal("fetch", fetchMock);

		const base = document.createElement("base");
		base.href = "https://viaduct.example.com/operator/";
		document.head.appendChild(base);

		try {
			await getAbout();
		} finally {
			base.remove();
		}

		const [input] = fetchMock.mock.calls[0] as [URL, RequestInit];
		expect(input.toString()).toBe(
			"https://viaduct.example.com/operator/api/v1/about",
		);
	});

	it("forwards an external abort signal reason to fetch", async () => {
		const externalController = new AbortController();
		const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
			return new Promise<Response>((_resolve, reject) => {
				init?.signal?.addEventListener(
					"abort",
					() =>
						reject(
							(init?.signal?.reason ??
								new DOMException("request aborted", "AbortError")) as Error,
						),
					{ once: true },
				);
			});
		});
		vi.stubGlobal("fetch", fetchMock);

		const abortReason = new DOMException("Canceled", "AbortError");
		const promise = requestManager.fetch("/api/v1/about", undefined, {
			signal: externalController.signal,
		});
		externalController.abort(abortReason);

		await expect(promise).rejects.toBe(abortReason);
	});

	it("rejects without issuing fetch when the external signal is already aborted", async () => {
		const externalController = new AbortController();
		const abortReason = new DOMException("Canceled", "AbortError");
		externalController.abort(abortReason);

		const fetchMock = vi.fn();
		vi.stubGlobal("fetch", fetchMock);

		await expect(
			requestManager.fetch("/api/v1/about", undefined, {
				signal: externalController.signal,
			}),
		).rejects.toBe(abortReason);

		expect(fetchMock).not.toHaveBeenCalled();
		expect(getActiveRequestControllerCount()).toBe(0);
	});

	it.each([
		['attachment; filename="report.csv"', "report.csv"],
		["attachment; filename*=UTF-8''r%C3%A9sum%C3%A9.csv", "résumé.csv"],
		[
			"attachment; filename=\"report.csv\"; filename*=UTF-8''r%C3%A9sum%C3%A9.csv",
			"résumé.csv",
		],
	])(
		"derives report filenames from content disposition %s",
		async (contentDisposition: string, expectedFilename: string) => {
			vi.stubGlobal(
				"fetch",
				vi.fn().mockResolvedValue(
					new Response("report", {
						status: 200,
						headers: {
							"Content-Disposition": contentDisposition,
							"Content-Type": "text/csv",
						},
					}),
				),
			);

			await expect(downloadReport("summary", "csv")).resolves.toMatchObject({
				filename: expectedFilename,
				contentType: "text/csv",
			});
		},
	);

	it("does not leak request controllers when request URL normalization throws", async () => {
		await expect(
			requestManager.fetch({ url: "http://%" } as Request),
		).rejects.toThrow();
		expect(getActiveRequestControllerCount()).toBe(0);
	});
});
