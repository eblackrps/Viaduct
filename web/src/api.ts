import type {
  AboutResponse,
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
  PreflightReport,
  RecommendationReport,
  RollbackResult,
  SnapshotMeta,
  SimulationRequest,
  SimulationResult,
  TenantSummary,
} from "./types";

const tenantApiKey = import.meta.env.VITE_VIADUCT_API_KEY?.trim();
const serviceAccountApiKey = import.meta.env.VITE_VIADUCT_SERVICE_ACCOUNT_KEY?.trim();
export type ReportName = "summary" | "migrations" | "audit";
export type ReportFormat = "json" | "csv";
export type DashboardAuthMode = "tenant" | "service-account" | "none";

export const dashboardAuthMode: DashboardAuthMode = serviceAccountApiKey
  ? "service-account"
  : tenantApiKey
    ? "tenant"
    : "none";
export const hasApiKeyConfigured = dashboardAuthMode !== "none";

function buildHeaders(initHeaders?: HeadersInit, hasBody = false): Headers {
  const headers = new Headers(initHeaders);
  if (hasBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (dashboardAuthMode === "service-account" && serviceAccountApiKey) {
    headers.set("X-Service-Account-Key", serviceAccountApiKey);
    headers.delete("X-API-Key");
  } else if (dashboardAuthMode === "tenant" && tenantApiKey) {
    headers.set("X-API-Key", tenantApiKey);
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
    throw new Error(await response.text());
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
    throw new Error(await response.text());
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
