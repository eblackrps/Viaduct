import type {
  DependencyGraph,
  DiscoveryResult,
  DriftReport,
  FleetCost,
  MigrationMeta,
  MigrationSpec,
  MigrationState,
  PlatformComparison,
  PolicyReport,
  PreflightReport,
  RecommendationReport,
  SnapshotMeta,
  SimulationRequest,
  SimulationResult,
  TenantSummary,
} from "./types";

const apiKey = import.meta.env.VITE_VIADUCT_API_KEY;

async function request<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, {
    headers: {
      "Content-Type": "application/json",
      ...(apiKey ? { "X-API-Key": apiKey } : {}),
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return (await response.json()) as T;
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

export function executeMigration(id: string): Promise<{ migration_id: string; status: string }> {
  return request(`/api/v1/migrations/${id}/execute`, {
    method: "POST",
  });
}

export function resumeMigration(id: string): Promise<{ migration_id: string; status: string }> {
  return request(`/api/v1/migrations/${id}/resume`, {
    method: "POST",
  });
}

export function rollbackMigration(id: string): Promise<MigrationState | Record<string, unknown>> {
  return request(`/api/v1/migrations/${id}/rollback`, {
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
