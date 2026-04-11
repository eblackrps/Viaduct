import type {
  AboutResponse,
  ApiErrorEnvelope,
  ApiFieldError,
  CurrentTenant,
  DependencyGraph,
  DiscoveryResult,
  DriftReport,
  FleetCost,
  MigrationMeta,
  MigrationCommandResponse,
  MigrationExecutionRequest,
  MigrationSpec,
  MigrationState,
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
      message: scope ? `Unable to load ${scope}: ${reason.humanMessage}` : reason.humanMessage,
      technicalDetails: buildAPIErrorTechnicalDetails(reason),
    };
  }

  if (reason instanceof Error && reason.message.trim() !== "") {
    return {
      message: scope ? `Unable to load ${scope}: ${reason.message}` : reason.message,
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

async function request<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, {
    ...init,
    headers: buildHeaders(init?.headers, init?.body !== undefined),
  });

  if (!response.ok) {
    throw await toAPIError(response);
  }

  return (await response.json()) as T;
}

async function requestBlob(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<{ blob: Blob; filename?: string; contentType: string }> {
  const response = await fetch(input, {
    ...init,
    headers: buildHeaders(init?.headers, init?.body !== undefined),
  });

  if (!response.ok) {
    throw await toAPIError(response);
  }

  return {
    blob: await response.blob(),
    filename: getFilenameFromDisposition(response.headers.get("Content-Disposition")),
    contentType: response.headers.get("Content-Type") ?? "application/octet-stream",
  };
}

export function getInventory(platform?: string): Promise<DiscoveryResult> {
  const search = platform ? `?platform=${platform}` : "";
  return request<DiscoveryResult>(`/api/v1/inventory${search}`);
}

export function getSnapshots(): Promise<SnapshotMeta[]> {
  return request<SnapshotMeta[]>("/api/v1/snapshots");
}

export function getSnapshot(id: string): Promise<DiscoveryResult> {
  return request<DiscoveryResult>(`/api/v1/snapshots/${id}`);
}

export function getGraph(): Promise<DependencyGraph> {
  return request<DependencyGraph>("/api/v1/graph");
}

export function createMigration(spec: MigrationSpec): Promise<MigrationState> {
  return request<MigrationState>("/api/v1/migrations", {
    method: "POST",
    body: JSON.stringify(spec),
  });
}

export function getMigrationState(id: string): Promise<MigrationState> {
  return request<MigrationState>(`/api/v1/migrations/${id}`);
}

export function executeMigration(id: string, payload?: MigrationExecutionRequest): Promise<MigrationCommandResponse> {
  return request<MigrationCommandResponse>(`/api/v1/migrations/${id}/execute`, {
    method: "POST",
    ...(hasExecutionPayload(payload) ? { body: JSON.stringify(payload) } : {}),
  });
}

export function resumeMigration(id: string, payload?: MigrationExecutionRequest): Promise<MigrationCommandResponse> {
  return request<MigrationCommandResponse>(`/api/v1/migrations/${id}/resume`, {
    method: "POST",
    ...(hasExecutionPayload(payload) ? { body: JSON.stringify(payload) } : {}),
  });
}

export function rollbackMigration(id: string): Promise<RollbackResult> {
  return request<RollbackResult>(`/api/v1/migrations/${id}/rollback`, {
    method: "POST",
  });
}

export function runPreflight(spec: MigrationSpec): Promise<PreflightReport> {
  return request<PreflightReport>("/api/v1/preflight", {
    method: "POST",
    body: JSON.stringify(spec),
  });
}

export function listMigrations(): Promise<MigrationMeta[]> {
  return request<MigrationMeta[]>("/api/v1/migrations");
}

export function getCosts(platform = "all"): Promise<PlatformComparison[] | FleetCost> {
  return request<PlatformComparison[] | FleetCost>(`/api/v1/costs?platform=${platform}`);
}

export function getPolicies(): Promise<PolicyReport> {
  return request<PolicyReport>("/api/v1/policies");
}

export function getDrift(baseline: string): Promise<DriftReport> {
  return request<DriftReport>(`/api/v1/drift?baseline=${baseline}`);
}

export function getRemediation(baseline?: string): Promise<RecommendationReport> {
  const search = baseline ? `?baseline=${baseline}` : "";
  return request<RecommendationReport>(`/api/v1/remediation${search}`);
}

export function runSimulation(payload: SimulationRequest): Promise<SimulationResult> {
  return request<SimulationResult>("/api/v1/simulation", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getTenantSummary(): Promise<TenantSummary> {
  return request<TenantSummary>("/api/v1/summary");
}

export function getAbout(): Promise<AboutResponse> {
  return request<AboutResponse>("/api/v1/about");
}

export function getCurrentTenant(): Promise<CurrentTenant> {
  return request<CurrentTenant>("/api/v1/tenants/current");
}

export function listWorkspaces(): Promise<PilotWorkspace[]> {
  return request<PilotWorkspace[]>("/api/v1/workspaces");
}

export function createWorkspace(payload: Partial<PilotWorkspace>): Promise<PilotWorkspace> {
  return request<PilotWorkspace>("/api/v1/workspaces", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getWorkspace(id: string): Promise<PilotWorkspace> {
  return request<PilotWorkspace>(`/api/v1/workspaces/${id}`);
}

export function updateWorkspace(id: string, payload: Partial<PilotWorkspace>): Promise<PilotWorkspace> {
  return request<PilotWorkspace>(`/api/v1/workspaces/${id}`, {
    method: "PATCH",
    body: JSON.stringify(payload),
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

export function getWorkspaceJob(workspaceID: string, jobID: string): Promise<WorkspaceJob> {
  return request<WorkspaceJob>(`/api/v1/workspaces/${workspaceID}/jobs/${jobID}`);
}

export async function exportWorkspaceReport(
  workspaceID: string,
  format: "markdown" | "json" = "markdown",
): Promise<{ blob: Blob; filename: string; contentType: string }> {
  const result = await requestBlob(`/api/v1/workspaces/${workspaceID}/reports/export`, {
    method: "POST",
    body: JSON.stringify({ format }),
  });
  return {
    blob: result.blob,
    filename: result.filename ?? `pilot-workspace-report.${format === "json" ? "json" : "md"}`,
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

function getFilenameFromDisposition(disposition: string | null): string | undefined {
  if (!disposition) {
    return undefined;
  }

  const match = disposition.match(/filename="?([^"]+)"?/i);
  return match?.[1];
}

function hasExecutionPayload(payload?: MigrationExecutionRequest): payload is MigrationExecutionRequest {
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
    return String(value);
  }
}
