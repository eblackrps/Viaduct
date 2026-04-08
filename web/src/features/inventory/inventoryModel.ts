import type {
  DependencyGraph,
  GraphEdge,
  GraphNode,
  Platform,
  PolicyReport,
  PolicyViolation,
  PowerState,
  RecommendationReport,
  RemediationRecommendation,
  VirtualMachine,
} from "../../types";
import { getVirtualMachineIdentity, getVirtualMachineLookupKeys } from "./workloadIdentity";

export type InventoryReadinessState = "ready" | "needs-review" | "blocked";
export type InventoryRiskState = "low" | "medium" | "high";
export type InventoryAssessmentSignal =
  | "policy-enforce"
  | "policy-warn"
  | "recommendation"
  | "snapshot"
  | "storage-gap"
  | "stale-discovery"
  | "backup-gap"
  | "power-unknown"
  | "network-gap";
export type InventorySortKey = "name" | "platform" | "cpu" | "memory" | "risk" | "readiness" | "recency" | "dependencies";

export interface InventoryFilterState {
  search: string;
  platform: Platform | "all";
  power: PowerState | "all";
  readiness: InventoryReadinessState | "all";
  risk: InventoryRiskState | "all";
  scope: "all" | "selected";
}

export interface DependencyRelation {
  edgeType: GraphEdge["type"];
  label: string;
  node: GraphNode;
}

export interface WorkloadDependencySummary {
  graphResolved: boolean;
  total: number;
  counts: Record<GraphEdge["type"], number>;
  networks: GraphNode[];
  datastores: GraphNode[];
  backups: GraphNode[];
  relations: DependencyRelation[];
}

export interface InventoryAssessmentRow {
  id: string;
  vm: VirtualMachine;
  readiness: InventoryReadinessState;
  risk: InventoryRiskState;
  riskScore: number;
  riskReasons: string[];
  signals: InventoryAssessmentSignal[];
  policyViolations: PolicyViolation[];
  recommendations: RemediationRecommendation[];
  dependencies: WorkloadDependencySummary;
  storageTotalMB: number;
  connectedNetworks: string[];
  discoveredAt?: string;
  createdAt?: string;
  lastActivityAt?: string;
  snapshotCount: number;
  connectedNicCount: number;
  disconnectedNicCount: number;
  searchDocument: string;
  assessmentIncomplete: boolean;
  missingSources: Array<"graph" | "policies" | "remediation">;
}

export interface InventoryAssessmentSourceState {
  graph: boolean;
  policies: boolean;
  remediation: boolean;
}

export interface InventoryAssessmentSummary {
  total: number;
  ready: number;
  needsReview: number;
  blocked: number;
  highRisk: number;
  mediumRisk: number;
  lowRisk: number;
  withBackups: number;
  withStorageDependencies: number;
  withNetworkDependencies: number;
  selected: number;
}

const riskWeight: Record<InventoryRiskState, number> = {
  low: 0,
  medium: 1,
  high: 2,
};

const readinessWeight: Record<InventoryReadinessState, number> = {
  ready: 0,
  "needs-review": 1,
  blocked: 2,
};

const signalLabels: Record<InventoryAssessmentSignal, string> = {
  "policy-enforce": "Policy enforcement",
  "policy-warn": "Policy warnings",
  recommendation: "Remediation guidance",
  snapshot: "Snapshots present",
  "storage-gap": "Storage metadata gaps",
  "stale-discovery": "Stale discovery",
  "backup-gap": "No backup relationship",
  "power-unknown": "Unknown power state",
  "network-gap": "Incomplete network dependency",
};

export function buildInventoryAssessmentRows(
  vms: VirtualMachine[],
  graph: DependencyGraph | null,
  policies: PolicyReport | null,
  remediation: RecommendationReport | null,
  sources: InventoryAssessmentSourceState,
  fallbackDiscoveredAt?: string,
): InventoryAssessmentRow[] {
  const policyIndex = buildPolicyIndex(policies);
  const recommendationIndex = buildRecommendationIndex(remediation);
  const dependencyIndex = buildDependencyIndex(graph);

  return vms.map((vm) => {
    const dependencySummary = resolveDependenciesForVM(vm, dependencyIndex);
    const policyViolations = lookupInventoryItems(policyIndex, vm, getPolicyViolationKey);
    const recommendations = lookupInventoryItems(recommendationIndex, vm, getRecommendationKey);
    const signals = new Set<InventoryAssessmentSignal>();
    const riskReasons: string[] = [];
    let riskScore = 0;

    const enforceViolations = policyViolations.filter((item) => item.severity === "enforce");
    const warningViolations = policyViolations.filter((item) => item.severity === "warn");
    const snapshotCount = vm.snapshots?.length ?? 0;
    const missingStorageMetadata = vm.disks.filter((disk) => disk.storage_backend.trim() === "").length;
    const powerUnknown = vm.power_state === "unknown";
    const noBackupRelationship =
      dependencySummary.graphResolved && dependencyIndex.hasBackupSignals && dependencySummary.backups.length === 0;
    const networkGap = dependencySummary.graphResolved && vm.nics.length > 0 && dependencySummary.networks.length === 0;
    const discoveredAt = vm.discovered_at ?? fallbackDiscoveredAt;
    const staleDiscovery = discoveredAt ? hoursSince(discoveredAt) > 72 : false;
    const missingSources = ([
      !sources.graph ? "graph" : null,
      !sources.policies ? "policies" : null,
      !sources.remediation ? "remediation" : null,
    ].filter(Boolean) as Array<"graph" | "policies" | "remediation">);

    if (enforceViolations.length > 0) {
      signals.add("policy-enforce");
      riskScore += enforceViolations.length * 4;
      riskReasons.push(`${enforceViolations.length} enforce-level policy violation(s)`);
    }
    if (warningViolations.length > 0) {
      signals.add("policy-warn");
      riskScore += warningViolations.length * 2;
      riskReasons.push(`${warningViolations.length} warning-level policy issue(s)`);
    }
    if (recommendations.length > 0) {
      signals.add("recommendation");
      riskScore += recommendations.length * 2;
      riskReasons.push(`${recommendations.length} remediation recommendation(s)`);
    }
    if (snapshotCount > 0) {
      signals.add("snapshot");
      riskScore += 1;
      riskReasons.push(`${snapshotCount} VM snapshot(s) present`);
    }
    if (missingStorageMetadata > 0) {
      signals.add("storage-gap");
      riskScore += missingStorageMetadata * 2;
      riskReasons.push(`Storage backend missing on ${missingStorageMetadata} disk(s)`);
    }
    if (powerUnknown) {
      signals.add("power-unknown");
      riskScore += 2;
      riskReasons.push("Power state could not be confirmed");
    }
    if (noBackupRelationship) {
      signals.add("backup-gap");
      riskScore += 1;
      riskReasons.push("No backup relationship is exposed in the current graph");
    }
    if (networkGap) {
      signals.add("network-gap");
      riskScore += 1;
      riskReasons.push("Network dependencies could not be resolved from the graph");
    }
    if (staleDiscovery) {
      signals.add("stale-discovery");
      riskScore += 1;
      riskReasons.push("Discovery data is older than 72 hours");
    }

    const readiness = resolveReadinessState({
      enforceCount: enforceViolations.length,
      missingStorageMetadata,
      powerUnknown,
      riskScore,
    });
    const risk = resolveRiskState(riskScore);
    const connectedNetworks = Array.from(
      new Set(
        vm.nics
          .map((nic) => nic.network.trim())
          .filter(Boolean),
      ),
    );
    const connectedNicCount = vm.nics.filter((nic) => nic.connected).length;
    const disconnectedNicCount = vm.nics.filter((nic) => !nic.connected).length;
    const storageTotalMB = vm.disks.reduce((total, disk) => total + disk.size_mb, 0);
    const lastSnapshotActivity = [...(vm.snapshots ?? [])]
      .map((snapshot) => snapshot.created_at)
      .filter(Boolean)
      .sort((left, right) => right.localeCompare(left))[0];
    const lastActivityAt = [vm.discovered_at, lastSnapshotActivity, vm.created_at]
      .filter((value): value is string => Boolean(value))
      .sort((left, right) => right.localeCompare(left))[0];

    return {
      id: getVirtualMachineIdentity(vm),
      vm,
      readiness,
      risk,
      riskScore,
      riskReasons,
      signals: Array.from(signals),
      policyViolations,
      recommendations,
      dependencies: dependencySummary,
      storageTotalMB,
      connectedNetworks,
      discoveredAt,
      createdAt: vm.created_at,
      lastActivityAt,
      snapshotCount,
      connectedNicCount,
      disconnectedNicCount,
      searchDocument: buildSearchDocument(vm, dependencySummary, policyViolations, recommendations),
      assessmentIncomplete: missingSources.length > 0,
      missingSources,
    };
  });
}

export function filterInventoryRows(rows: InventoryAssessmentRow[], filters: InventoryFilterState): InventoryAssessmentRow[] {
  const text = filters.search.trim().toLowerCase();

  return rows.filter((row) => {
    if (filters.platform !== "all" && row.vm.platform !== filters.platform) {
      return false;
    }
    if (filters.power !== "all" && row.vm.power_state !== filters.power) {
      return false;
    }
    if (filters.readiness !== "all" && row.readiness !== filters.readiness) {
      return false;
    }
    if (filters.risk !== "all" && row.risk !== filters.risk) {
      return false;
    }
    if (text && !row.searchDocument.includes(text)) {
      return false;
    }
    return true;
  });
}

export function sortInventoryRows(
  rows: InventoryAssessmentRow[],
  sortKey: InventorySortKey,
  sortDirection: "asc" | "desc",
): InventoryAssessmentRow[] {
  return [...rows].sort((left, right) => {
    const direction = sortDirection === "asc" ? 1 : -1;
    let comparison = 0;

    switch (sortKey) {
      case "platform":
        comparison = compareStrings(left.vm.platform, right.vm.platform);
        break;
      case "cpu":
        comparison = left.vm.cpu_count - right.vm.cpu_count;
        break;
      case "memory":
        comparison = left.vm.memory_mb - right.vm.memory_mb;
        break;
      case "risk":
        comparison = riskWeight[left.risk] - riskWeight[right.risk] || left.riskScore - right.riskScore;
        break;
      case "readiness":
        comparison = readinessWeight[left.readiness] - readinessWeight[right.readiness] || left.riskScore - right.riskScore;
        break;
      case "recency":
        comparison = compareNumbers(timestampValue(left.discoveredAt), timestampValue(right.discoveredAt));
        break;
      case "dependencies":
        comparison = left.dependencies.total - right.dependencies.total;
        break;
      case "name":
      default:
        comparison = compareStrings(left.vm.name, right.vm.name);
        break;
    }

    if (comparison === 0) {
      comparison = compareStrings(left.vm.name, right.vm.name);
    }

    return comparison * direction;
  });
}

export function summarizeInventoryRows(rows: InventoryAssessmentRow[], selectedCount: number): InventoryAssessmentSummary {
  return rows.reduce<InventoryAssessmentSummary>(
    (summary, row) => {
      summary.total += 1;
      summary.selected = selectedCount;
      summary[row.readiness === "ready" ? "ready" : row.readiness === "needs-review" ? "needsReview" : "blocked"] += 1;
      summary[row.risk === "low" ? "lowRisk" : row.risk === "medium" ? "mediumRisk" : "highRisk"] += 1;
      if (row.dependencies.backups.length > 0) {
        summary.withBackups += 1;
      }
      if (row.dependencies.datastores.length > 0) {
        summary.withStorageDependencies += 1;
      }
      if (row.dependencies.networks.length > 0) {
        summary.withNetworkDependencies += 1;
      }
      return summary;
    },
    {
      total: 0,
      ready: 0,
      needsReview: 0,
      blocked: 0,
      highRisk: 0,
      mediumRisk: 0,
      lowRisk: 0,
      withBackups: 0,
      withStorageDependencies: 0,
      withNetworkDependencies: 0,
      selected: selectedCount,
    },
  );
}

export function summarizeAssessmentSignals(rows: InventoryAssessmentRow[]): Array<{ label: string; count: number }> {
  const counts = new Map<InventoryAssessmentSignal, number>();

  for (const row of rows) {
    for (const signal of row.signals) {
      counts.set(signal, (counts.get(signal) ?? 0) + 1);
    }
  }

  return Array.from(counts.entries())
    .map(([signal, count]) => ({ label: signalLabels[signal], count }))
    .sort((left, right) => right.count - left.count)
    .slice(0, 5);
}

export function formatTimestamp(value?: string): string {
  if (!value) {
    return "Unavailable";
  }

  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return "Unavailable";
  }

  return timestamp.toLocaleString();
}

export function formatRelativeTime(value?: string): string {
  if (!value) {
    return "Unavailable";
  }

  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return "Unavailable";
  }

  const deltaMinutes = Math.max(0, Math.round((Date.now() - timestamp.getTime()) / 60000));
  if (deltaMinutes < 1) {
    return "just now";
  }
  if (deltaMinutes < 60) {
    return `${deltaMinutes}m ago`;
  }
  const deltaHours = Math.round(deltaMinutes / 60);
  if (deltaHours < 24) {
    return `${deltaHours}h ago`;
  }
  const deltaDays = Math.round(deltaHours / 24);
  return `${deltaDays}d ago`;
}

function buildPolicyIndex(report: PolicyReport | null): Map<string, PolicyViolation[]> {
  const index = new Map<string, PolicyViolation[]>();

  for (const violation of report?.violations ?? []) {
    for (const key of getVirtualMachineLookupKeys(violation.vm)) {
      index.set(key, [...(index.get(key) ?? []), violation]);
    }
  }

  return index;
}

function buildRecommendationIndex(report: RecommendationReport | null): Map<string, RemediationRecommendation[]> {
  const index = new Map<string, RemediationRecommendation[]>();

  for (const recommendation of report?.recommendations ?? []) {
    for (const key of getVirtualMachineLookupKeys(recommendation.vm)) {
      index.set(key, [...(index.get(key) ?? []), recommendation]);
    }
  }

  return index;
}

function buildDependencyIndex(graph: DependencyGraph | null): {
  graphResolved: boolean;
  hasBackupSignals: boolean;
  nodeById: Map<string, GraphNode>;
  relationsByNodeId: Map<string, DependencyRelation[]>;
} {
  const nodeById = new Map<string, GraphNode>((graph?.nodes ?? []).map((node) => [node.id, node]));
  const relationsByNodeId = new Map<string, DependencyRelation[]>();

  for (const edge of graph?.edges ?? []) {
    const sourceNode = nodeById.get(edge.source);
    const targetNode = nodeById.get(edge.target);
    if (sourceNode && targetNode) {
      relationsByNodeId.set(edge.source, [
        ...(relationsByNodeId.get(edge.source) ?? []),
        { edgeType: edge.type, label: edge.label, node: targetNode },
      ]);
      relationsByNodeId.set(edge.target, [
        ...(relationsByNodeId.get(edge.target) ?? []),
        { edgeType: edge.type, label: edge.label, node: sourceNode },
      ]);
    }
  }

  return {
    graphResolved: graph !== null,
    hasBackupSignals: (graph?.nodes ?? []).some((node) => node.type === "backup-job"),
    nodeById,
    relationsByNodeId,
  };
}

function resolveDependenciesForVM(
  vm: VirtualMachine,
  index: {
    graphResolved: boolean;
    hasBackupSignals: boolean;
    nodeById: Map<string, GraphNode>;
    relationsByNodeId: Map<string, DependencyRelation[]>;
  },
): WorkloadDependencySummary {
  const candidateNodeIDs = [`vm:${vm.id}`, `vm:${vm.name}`].filter((value) => value !== "vm:");
  const nodeID =
    candidateNodeIDs.find((value) => index.nodeById.has(value)) ??
    Array.from(index.nodeById.values()).find((node) => node.type === "vm" && node.label.trim().toLowerCase() === vm.name.trim().toLowerCase())?.id;
  const relations = nodeID ? dedupeRelations(index.relationsByNodeId.get(nodeID) ?? []) : [];
  const networks = relations.filter((relation) => relation.node.type === "network").map((relation) => relation.node);
  const datastores = relations.filter((relation) => relation.node.type === "datastore").map((relation) => relation.node);
  const backups = relations.filter((relation) => relation.node.type === "backup-job").map((relation) => relation.node);

  return {
    graphResolved: index.graphResolved,
    total: relations.length,
    counts: {
      network: relations.filter((relation) => relation.edgeType === "network").length,
      storage: relations.filter((relation) => relation.edgeType === "storage").length,
      backup: relations.filter((relation) => relation.edgeType === "backup").length,
    },
    networks: dedupeNodes(networks),
    datastores: dedupeNodes(datastores),
    backups: dedupeNodes(backups),
    relations,
  };
}

function resolveReadinessState({
  enforceCount,
  missingStorageMetadata,
  powerUnknown,
  riskScore,
}: {
  enforceCount: number;
  missingStorageMetadata: number;
  powerUnknown: boolean;
  riskScore: number;
}): InventoryReadinessState {
  if (enforceCount > 0 || missingStorageMetadata > 0 || powerUnknown) {
    return "blocked";
  }
  if (riskScore >= 3) {
    return "needs-review";
  }
  return "ready";
}

function resolveRiskState(riskScore: number): InventoryRiskState {
  if (riskScore >= 6) {
    return "high";
  }
  if (riskScore >= 3) {
    return "medium";
  }
  return "low";
}

function lookupInventoryItems<T>(index: Map<string, T[]>, vm: VirtualMachine, getKey: (item: T) => string): T[] {
  const values = new Map<string, T>();

  for (const key of getVirtualMachineLookupKeys(vm)) {
    for (const item of index.get(key) ?? []) {
      values.set(getKey(item), item);
    }
  }

  return Array.from(values.values());
}

function buildSearchDocument(
  vm: VirtualMachine,
  dependencies: WorkloadDependencySummary,
  policyViolations: PolicyViolation[],
  recommendations: RemediationRecommendation[],
): string {
  return [
    vm.name,
    vm.platform,
    vm.power_state,
    vm.guest_os ?? "",
    vm.host,
    vm.cluster ?? "",
    vm.resource_pool ?? "",
    vm.folder ?? "",
    vm.source_ref ?? "",
    Object.entries(vm.tags ?? {})
      .map(([key, value]) => `${key} ${value}`)
      .join(" "),
    vm.nics.map((nic) => `${nic.name} ${nic.network} ${nic.ip_addresses.join(" ")}`).join(" "),
    vm.disks.map((disk) => `${disk.name} ${disk.storage_backend}`).join(" "),
    dependencies.relations.map((relation) => `${relation.edgeType} ${relation.node.label} ${relation.label}`).join(" "),
    policyViolations.map((violation) => `${violation.policy.name} ${violation.rule.field} ${violation.remediation ?? ""}`).join(" "),
    recommendations.map((recommendation) => `${recommendation.summary} ${recommendation.action} ${recommendation.type}`).join(" "),
  ]
    .join(" ")
    .toLowerCase();
}

function dedupeRelations(relations: DependencyRelation[]): DependencyRelation[] {
  const seen = new Set<string>();
  const items: DependencyRelation[] = [];

  for (const relation of relations) {
    const key = `${relation.edgeType}:${relation.node.id}:${relation.label}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    items.push(relation);
  }

  return items;
}

function getPolicyViolationKey(item: PolicyViolation): string {
  return [
    getVirtualMachineIdentity(item.vm),
    item.policy.name,
    item.rule.field,
    item.rule.operator,
    String(item.rule.value),
    item.severity,
  ].join("|");
}

function getRecommendationKey(item: RemediationRecommendation): string {
  return [getVirtualMachineIdentity(item.vm), item.type, item.summary, item.action].join("|");
}

function dedupeNodes(nodes: GraphNode[]): GraphNode[] {
  const seen = new Set<string>();
  const items: GraphNode[] = [];

  for (const node of nodes) {
    if (seen.has(node.id)) {
      continue;
    }
    seen.add(node.id);
    items.push(node);
  }

  return items;
}

function compareStrings(left: string, right: string): number {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: "base" });
}

function compareNumbers(left: number, right: number): number {
  return left - right;
}

function timestampValue(value?: string): number {
  if (!value) {
    return 0;
  }

  const timestamp = new Date(value);
  return Number.isNaN(timestamp.getTime()) ? 0 : timestamp.getTime();
}

function hoursSince(value: string): number {
  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return 0;
  }
  return (Date.now() - timestamp.getTime()) / 3600000;
}
