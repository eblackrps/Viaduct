import type {
	AboutResponse,
	ApiErrorEnvelope,
	ApiFieldError,
	CurrentTenant,
	DependencyGraph,
	DiscoveryResult,
	DriftReport,
	FleetCost,
	InventoryListResponse,
	MigrationMeta,
	MigrationCommandResponse,
	MigrationExecutionRequest,
	MigrationSpec,
	MigrationState,
	PaginatedResponse,
	PlatformComparison,
	PolicyReport,
	PilotWorkspace,
	PreflightReport,
	RecommendationReport,
	RollbackResult,
	SnapshotMeta,
	SimulationRequest,
	SimulationResult,
	TenantSummary,
	WorkspaceJob,
	WorkspaceJobType,
} from "./types";
import {
	getDashboardAuthMode,
	getDashboardAuthSession,
	hasDashboardAuthConfigured,
	type DashboardAuthMode,
} from "./runtimeAuth";
export type ReportName = "summary" | "migrations" | "audit";
export type ReportFormat = "json" | "csv";

export interface ErrorDisplay {
	message: string;
	technicalDetails: string[];
}

interface APIErrorOptions {
	status: number;
	message: string;
	requestID?: string;
	code?: string;
	retryable?: boolean;
	details?: Record<string, unknown>;
	fieldErrors?: ApiFieldError[];
}

export class TimeoutError extends Error {
	readonly timeoutMs: number;

	constructor(timeoutMs: number) {
		super(`Request timed out after ${Math.round(timeoutMs / 1000)}s.`);
		Object.setPrototypeOf(this, new.target.prototype);
		this.name = "TimeoutError";
		this.timeoutMs = timeoutMs;
	}
}

export class APIError extends Error {
	readonly status: number;
	readonly humanMessage: string;
	readonly requestID?: string;
	readonly code?: string;
	readonly retryable: boolean;
	readonly details: Record<string, unknown>;
	readonly fieldErrors: ApiFieldError[];

	constructor({
		status,
		message,
		requestID,
		code,
		retryable = false,
		details = {},
		fieldErrors = [],
	}: APIErrorOptions) {
		super(requestID ? `${message} Request ID: ${requestID}.` : message);
		Object.setPrototypeOf(this, new.target.prototype);
		this.name = "APIError";
		this.status = status;
		this.humanMessage = message;
		this.requestID = requestID;
		this.code = code;
		this.retryable = retryable;
		this.details = details;
		this.fieldErrors = fieldErrors;
	}
}

export function dashboardAuthMode(): DashboardAuthMode {
	return getDashboardAuthMode();
}

export function hasApiKeyConfigured(): boolean {
	return hasDashboardAuthConfigured();
}

export function isAPIError(reason: unknown): reason is APIError {
	return reason instanceof APIError;
}

export function isTimeoutError(reason: unknown): reason is TimeoutError {
	return reason instanceof TimeoutError;
}

export function isAbortError(reason: unknown): boolean {
	return reason instanceof DOMException
		? reason.name === "AbortError"
		: reason instanceof Error && reason.name === "AbortError";
}

export function describeError(
	reason: unknown,
	options: {
		scope?: string;
		fallback: string;
	},
): ErrorDisplay {
	const scope = options.scope?.trim();
	if (isAPIError(reason)) {
		return {
			message: scope
				? `Unable to load ${scope}: ${reason.humanMessage}`
				: reason.humanMessage,
			technicalDetails: buildAPIErrorTechnicalDetails(reason),
		};
	}

	if (isTimeoutError(reason)) {
		return {
			message: scope
				? `Unable to load ${scope}: ${reason.message}`
				: reason.message,
			technicalDetails: [`Timeout: ${reason.timeoutMs}ms`],
		};
	}

	if (reason instanceof Error && reason.message.trim() !== "") {
		return {
			message: scope
				? `Unable to load ${scope}: ${reason.message}`
				: reason.message,
			technicalDetails: [],
		};
	}

	return {
		message: scope ? `Unable to load ${scope}.` : options.fallback,
		technicalDetails: [],
	};
}

function buildHeaders(initHeaders?: HeadersInit, hasBody = false): Headers {
	const headers = new Headers(initHeaders);
	if (hasBody && !headers.has("Content-Type")) {
		headers.set("Content-Type", "application/json");
	}
	const session = getDashboardAuthSession();
	if (session.mode === "service-account" && session.apiKey) {
		headers.set("X-Service-Account-Key", session.apiKey);
		headers.delete("X-API-Key");
	} else if (session.mode === "tenant" && session.apiKey) {
		headers.set("X-API-Key", session.apiKey);
		headers.delete("X-Service-Account-Key");
	}
	return headers;
}

interface RequestOptions {
	signal?: AbortSignal;
	timeoutMs?: number;
	dedupe?: boolean;
	skipAuth?: boolean;
}

interface ListRequestOptions extends RequestOptions {
	page?: number;
	perPage?: number;
}

const configuredTimeoutValue =
	typeof import.meta.env.VITE_VIADUCT_API_TIMEOUT_MS === "string"
		? import.meta.env.VITE_VIADUCT_API_TIMEOUT_MS
		: "30000";
const defaultRequestTimeoutMs =
	Number.parseInt(configuredTimeoutValue, 10) || 30000;

class RequestManager {
	private readonly inflight = new Map<
		string,
		{ controller: AbortController; promise: Promise<Response> }
	>();

	async fetch(
		input: RequestInfo | URL,
		init?: RequestInit,
		options?: RequestOptions,
	): Promise<Response> {
		const method = (init?.method ?? "GET").toUpperCase();
		const dedupeKey =
			options?.dedupe === false || method !== "GET"
				? ""
				: `${method}:${toRequestKey(input)}`;
		if (dedupeKey && this.inflight.has(dedupeKey)) {
			return this.inflight
				.get(dedupeKey)!
				.promise.then((response) => response.clone());
		}

		const controller = new AbortController();
		const timeoutMs = options?.timeoutMs ?? defaultRequestTimeoutMs;
		let timedOut = false;
		const timeoutID = window.setTimeout(() => {
			timedOut = true;
			controller.abort(new DOMException("Timed out", "AbortError"));
		}, timeoutMs);
		const forwardAbort = () => controller.abort();
		options?.signal?.addEventListener("abort", forwardAbort, { once: true });

		const promise = (async () => {
			try {
				return await fetch(input, {
					...init,
					credentials: "same-origin",
					headers: options?.skipAuth
						? new Headers(init?.headers)
						: buildHeaders(init?.headers, init?.body !== undefined),
					signal: controller.signal,
				});
			} catch (reason) {
				if (timedOut) {
					throw new TimeoutError(timeoutMs);
				}
				throw reason;
			} finally {
				window.clearTimeout(timeoutID);
				options?.signal?.removeEventListener("abort", forwardAbort);
				if (dedupeKey) {
					this.inflight.delete(dedupeKey);
				}
			}
		})();

		if (dedupeKey) {
			this.inflight.set(dedupeKey, { controller, promise });
			return promise.then((response) => response.clone());
		}
		return promise;
	}

	cancel(key: string) {
		const entry = this.inflight.get(key);
		if (!entry) {
			return;
		}
		entry.controller.abort(new DOMException("Canceled", "AbortError"));
		this.inflight.delete(key);
	}

	cancelAll() {
		for (const [key, entry] of this.inflight.entries()) {
			entry.controller.abort(new DOMException("Canceled", "AbortError"));
			this.inflight.delete(key);
		}
	}
}

export const requestManager = new RequestManager();

function toRequestKey(input: RequestInfo | URL): string {
	if (typeof input === "string") {
		return input;
	}
	if (input instanceof URL) {
		return input.toString();
	}
	return input.url;
}

function withListParams(path: string, options?: ListRequestOptions): string {
	const url = new URL(path, window.location.origin);
	if (options?.page) {
		url.searchParams.set("page", String(options.page));
	}
	if (options?.perPage) {
		url.searchParams.set("per_page", String(options.perPage));
	}
	return `${url.pathname}${url.search}`;
}

async function request<T>(
	input: RequestInfo | URL,
	init?: RequestInit,
	options?: RequestOptions,
): Promise<T> {
	const response = await requestManager.fetch(input, init, options);
	if (!response.ok) {
		throw await toAPIError(response);
	}

	return (await response.json()) as T;
}

async function requestVoid(
	input: RequestInfo | URL,
	init?: RequestInit,
	options?: RequestOptions,
): Promise<void> {
	const response = await requestManager.fetch(input, init, options);
	if (!response.ok) {
		throw await toAPIError(response);
	}
}

async function requestBlob(
	input: RequestInfo | URL,
	init?: RequestInit,
	options?: RequestOptions,
): Promise<{ blob: Blob; filename?: string; contentType: string }> {
	const response = await requestManager.fetch(input, init, options);
	if (!response.ok) {
		throw await toAPIError(response);
	}

	return {
		blob: await response.blob(),
		filename: getFilenameFromDisposition(
			response.headers.get("Content-Disposition"),
		),
		contentType:
			response.headers.get("Content-Type") ?? "application/octet-stream",
	};
}

export function getInventory(
	platform?: string,
	options?: ListRequestOptions,
): Promise<InventoryListResponse> {
	const search = new URLSearchParams();
	if (platform) {
		search.set("platform", platform);
	}
	if (options?.page) {
		search.set("page", String(options.page));
	}
	if (options?.perPage) {
		search.set("per_page", String(options.perPage));
	}
	const suffix = search.toString();
	return request<InventoryListResponse>(
		`/api/v2/inventory${suffix ? `?${suffix}` : ""}`,
		undefined,
		options,
	);
}

export function getSnapshots(
	options?: ListRequestOptions,
): Promise<PaginatedResponse<SnapshotMeta>> {
	return request<PaginatedResponse<SnapshotMeta>>(
		withListParams("/api/v2/snapshots", options),
		undefined,
		options,
	);
}

export function getSnapshot(
	id: string,
	options?: RequestOptions,
): Promise<DiscoveryResult> {
	return request<DiscoveryResult>(
		`/api/v1/snapshots/${id}`,
		undefined,
		options,
	);
}

export function getGraph(options?: RequestOptions): Promise<DependencyGraph> {
	return request<DependencyGraph>("/api/v1/graph", undefined, options);
}

export function createMigration(
	spec: MigrationSpec,
	options?: RequestOptions,
): Promise<MigrationState> {
	return request<MigrationState>(
		"/api/v1/migrations",
		{
			method: "POST",
			body: JSON.stringify(spec),
		},
		options,
	);
}

export function getMigrationState(
	id: string,
	options?: RequestOptions,
): Promise<MigrationState> {
	return request<MigrationState>(
		`/api/v1/migrations/${id}`,
		undefined,
		options,
	);
}

export function executeMigration(
	id: string,
	payload?: MigrationExecutionRequest,
	options?: RequestOptions,
): Promise<MigrationCommandResponse> {
	return request<MigrationCommandResponse>(
		`/api/v1/migrations/${id}/execute`,
		{
			method: "POST",
			...(hasExecutionPayload(payload)
				? { body: JSON.stringify(payload) }
				: {}),
		},
		options,
	);
}

export function resumeMigration(
	id: string,
	payload?: MigrationExecutionRequest,
	options?: RequestOptions,
): Promise<MigrationCommandResponse> {
	return request<MigrationCommandResponse>(
		`/api/v1/migrations/${id}/resume`,
		{
			method: "POST",
			...(hasExecutionPayload(payload)
				? { body: JSON.stringify(payload) }
				: {}),
		},
		options,
	);
}

export function rollbackMigration(
	id: string,
	options?: RequestOptions,
): Promise<RollbackResult> {
	return request<RollbackResult>(
		`/api/v1/migrations/${id}/rollback`,
		{
			method: "POST",
		},
		options,
	);
}

export function runPreflight(
	spec: MigrationSpec,
	options?: RequestOptions,
): Promise<PreflightReport> {
	return request<PreflightReport>(
		"/api/v1/preflight",
		{
			method: "POST",
			body: JSON.stringify(spec),
		},
		options,
	);
}

export function listMigrations(
	options?: ListRequestOptions,
): Promise<PaginatedResponse<MigrationMeta>> {
	return request<PaginatedResponse<MigrationMeta>>(
		withListParams("/api/v2/migrations", options),
		undefined,
		options,
	);
}

export function getCosts(
	platform = "all",
	options?: RequestOptions,
): Promise<PlatformComparison[] | FleetCost> {
	return request<PlatformComparison[] | FleetCost>(
		`/api/v1/costs?platform=${platform}`,
		undefined,
		options,
	);
}

export function getPolicies(options?: RequestOptions): Promise<PolicyReport> {
	return request<PolicyReport>("/api/v1/policies", undefined, options);
}

export function getDrift(
	baseline: string,
	options?: RequestOptions,
): Promise<DriftReport> {
	return request<DriftReport>(
		`/api/v1/drift?baseline=${baseline}`,
		undefined,
		options,
	);
}

export function getRemediation(
	baseline?: string,
	options?: RequestOptions,
): Promise<RecommendationReport> {
	const search = baseline ? `?baseline=${baseline}` : "";
	return request<RecommendationReport>(
		`/api/v1/remediation${search}`,
		undefined,
		options,
	);
}

export function runSimulation(
	payload: SimulationRequest,
	options?: RequestOptions,
): Promise<SimulationResult> {
	return request<SimulationResult>(
		"/api/v1/simulation",
		{
			method: "POST",
			body: JSON.stringify(payload),
		},
		options,
	);
}

export function getTenantSummary(
	options?: RequestOptions,
): Promise<TenantSummary> {
	return request<TenantSummary>("/api/v1/summary", undefined, options);
}

export function getAbout(options?: RequestOptions): Promise<AboutResponse> {
	return request<AboutResponse>("/api/v1/about", undefined, options);
}

export function getCurrentTenant(
	options?: RequestOptions,
): Promise<CurrentTenant> {
	return request<CurrentTenant>("/api/v1/tenants/current", undefined, options);
}

export function listWorkspaces(): Promise<PilotWorkspace[]> {
	return request<PilotWorkspace[]>("/api/v1/workspaces");
}

export function createDashboardAuthSession(
	mode: Exclude<DashboardAuthMode, "none">,
	apiKey: string,
	remember = false,
	options?: RequestOptions,
): Promise<{
	session_id: string;
	mode: DashboardAuthMode;
	expires_at?: string;
}> {
	return request<{
		session_id: string;
		mode: DashboardAuthMode;
		expires_at?: string;
	}>(
		"/api/v1/auth/session",
		{
			method: "POST",
			body: JSON.stringify({
				mode,
				api_key: apiKey.trim(),
				remember,
			}),
		},
		{ ...options, dedupe: false, skipAuth: true },
	);
}

export function deleteDashboardAuthSession(
	options?: RequestOptions,
): Promise<void> {
	return requestVoid(
		"/api/v1/auth/session",
		{
			method: "DELETE",
		},
		{ ...options, dedupe: false, skipAuth: true },
	);
}

export function createWorkspace(
	payload: Partial<PilotWorkspace>,
): Promise<PilotWorkspace> {
	return request<PilotWorkspace>("/api/v1/workspaces", {
		method: "POST",
		body: JSON.stringify(payload),
	});
}

export function getWorkspace(id: string): Promise<PilotWorkspace> {
	return request<PilotWorkspace>(`/api/v1/workspaces/${id}`);
}

export function updateWorkspace(
	id: string,
	payload: Partial<PilotWorkspace>,
): Promise<PilotWorkspace> {
	return request<PilotWorkspace>(`/api/v1/workspaces/${id}`, {
		method: "PATCH",
		body: JSON.stringify(payload),
	});
}

export function deleteWorkspace(id: string): Promise<void> {
	return requestVoid(`/api/v1/workspaces/${id}`, {
		method: "DELETE",
	});
}

export function listWorkspaceJobs(id: string): Promise<WorkspaceJob[]> {
	return request<WorkspaceJob[]>(`/api/v1/workspaces/${id}/jobs`);
}

export function createWorkspaceJob(
	id: string,
	payload: {
		type: WorkspaceJobType;
		requested_by?: string;
		source_connection_ids?: string[];
		selected_workload_ids?: string[];
		simulation?: SimulationRequest;
	},
): Promise<WorkspaceJob> {
	return request<WorkspaceJob>(`/api/v1/workspaces/${id}/jobs`, {
		method: "POST",
		body: JSON.stringify(payload),
	});
}

export function getWorkspaceJob(
	workspaceID: string,
	jobID: string,
): Promise<WorkspaceJob> {
	return request<WorkspaceJob>(
		`/api/v1/workspaces/${workspaceID}/jobs/${jobID}`,
	);
}

export async function exportWorkspaceReport(
	workspaceID: string,
	format: "markdown" | "json" = "markdown",
): Promise<{ blob: Blob; filename: string; contentType: string }> {
	const result = await requestBlob(
		`/api/v1/workspaces/${workspaceID}/reports/export`,
		{
			method: "POST",
			body: JSON.stringify({ format }),
		},
	);
	return {
		blob: result.blob,
		filename:
			result.filename ??
			`pilot-workspace-report.${format === "json" ? "json" : "md"}`,
		contentType: result.contentType,
	};
}

export async function downloadReport(
	name: ReportName,
	format: ReportFormat = "json",
): Promise<{ blob: Blob; filename: string; contentType: string }> {
	const result = await requestBlob(`/api/v1/reports/${name}?format=${format}`);
	return {
		blob: result.blob,
		filename: result.filename ?? `${name}.${format}`,
		contentType: result.contentType,
	};
}

function getFilenameFromDisposition(
	disposition: string | null,
): string | undefined {
	if (!disposition) {
		return undefined;
	}

	const match = disposition.match(/filename="?([^"]+)"?/i);
	return match?.[1];
}

function hasExecutionPayload(
	payload?: MigrationExecutionRequest,
): payload is MigrationExecutionRequest {
	return Boolean(payload?.approved_by?.trim() || payload?.ticket?.trim());
}

async function toAPIError(response: Response): Promise<Error> {
	const body = (await response.text()).trim();
	if (body) {
		try {
			const payload = JSON.parse(body) as ApiErrorEnvelope;
			if (payload?.error?.message) {
				return new APIError({
					status: response.status,
					message: payload.error.message,
					requestID: payload.error.request_id?.trim() || undefined,
					code: payload.error.code?.trim() || undefined,
					retryable: payload.error.retryable,
					details: payload.error.details ?? {},
					fieldErrors: payload.error.field_errors ?? [],
				});
			}
		} catch {
			// Fall through to the raw response body.
		}
	}
	return new APIError({
		status: response.status,
		message: body || `Request failed with status ${response.status}`,
	});
}

function buildAPIErrorTechnicalDetails(error: APIError): string[] {
	const details: string[] = [`HTTP status: ${error.status}`];

	if (error.code) {
		details.push(`Code: ${error.code}`);
	}
	if (error.requestID) {
		details.push(`Request ID: ${error.requestID}`);
	}
	if (error.retryable) {
		details.push("Retryable: yes");
	}

	for (const [key, value] of Object.entries(error.details)) {
		details.push(`${formatDetailLabel(key)}: ${formatDetailValue(value)}`);
	}

	for (const fieldError of error.fieldErrors) {
		const path = fieldError.path.trim() || "request";
		details.push(`Field ${path}: ${fieldError.message}`);
	}

	return details;
}

function formatDetailLabel(key: string): string {
	return key.replace(/_/g, " ");
}

function formatDetailValue(value: unknown): string {
	if (value === null) {
		return "null";
	}
	if (typeof value === "string") {
		return value;
	}
	if (typeof value === "number" || typeof value === "boolean") {
		return String(value);
	}
	if (Array.isArray(value)) {
		return value.map((item) => formatDetailValue(item)).join(", ");
	}

	try {
		return JSON.stringify(value);
	} catch {
		if (value instanceof Error) {
			return value.message;
		}
		return "unserializable detail";
	}
}
