export type Platform = "vmware" | "proxmox" | "hyperv" | "kvm" | "nutanix";
export type PowerState = "on" | "off" | "suspended" | "unknown";
export type TenantRole = "viewer" | "operator" | "admin";
export type TenantPermission =
  | "inventory.read"
  | "reports.read"
  | "lifecycle.read"
  | "migration.manage"
  | "tenant.read"
  | "tenant.manage";
export type MigrationPhase =
  | "plan"
  | "export"
  | "convert"
  | "import"
  | "configure"
  | "verify"
  | "complete"
  | "failed"
  | "rolled_back";

export type CheckpointStatus = "pending" | "running" | "completed" | "failed";

export interface Disk {
  id: string;
  name: string;
  size_mb: number;
  thin: boolean;
  storage_backend: string;
}

export interface Nic {
  id: string;
  name: string;
  mac_address: string;
  network: string;
  connected: boolean;
  ip_addresses: string[];
}

export interface WorkloadSnapshot {
  id: string;
  name: string;
  description: string;
  created_at: string;
  size_mb: number;
}

export interface NetworkInfo {
  id: string;
  name: string;
  type: string;
  vlan_id: number;
  switch: string;
}

export interface DatastoreInfo {
  id: string;
  name: string;
  type: string;
  capacity_mb: number;
  free_mb: number;
  hosts: string[];
}

export interface HostInfo {
  id: string;
  name: string;
  cluster: string;
  cpu_cores: number;
  memory_mb: number;
  power_state: PowerState;
  connection_state: string;
}

export interface ClusterInfo {
  id: string;
  name: string;
  hosts: string[];
  total_cpu_cores: number;
  total_memory_mb: number;
  ha_enabled: boolean;
  drs_enabled: boolean;
}

export interface ResourcePoolInfo {
  id: string;
  name: string;
  cluster: string;
  cpu_limit_mhz: number;
  memory_limit_mb: number;
}

export interface VirtualMachine {
  id: string;
  name: string;
  platform: Platform;
  power_state: PowerState;
  cpu_count: number;
  memory_mb: number;
  disks: Disk[];
  nics: Nic[];
  guest_os?: string;
  host: string;
  cluster?: string;
  resource_pool?: string;
  folder?: string;
  tags?: Record<string, string>;
  snapshots?: WorkloadSnapshot[];
  created_at?: string;
  discovered_at?: string;
  source_ref?: string;
}

export interface DiscoveryResult {
  source?: string;
  platform?: Platform;
  vms: VirtualMachine[];
  networks?: NetworkInfo[];
  datastores?: DatastoreInfo[];
  hosts?: HostInfo[];
  clusters?: ClusterInfo[];
  resource_pools?: ResourcePoolInfo[];
  discovered_at?: string;
  duration?: number;
  errors?: string[];
}

export interface VMCost {
  vm: VirtualMachine;
  monthly_cpu_cost: number;
  monthly_memory_cost: number;
  monthly_storage_cost: number;
  monthly_license_cost: number;
  monthly_total: number;
  annual_total: number;
}

export interface PlatformComparison {
  vm: VirtualMachine;
  cost_by_platform: Record<string, VMCost>;
  cheapest_platform: string;
  monthly_savings: number;
}

export interface FleetCost {
  platform: string;
  total_monthly: number;
  total_annual: number;
  vm_costs: VMCost[];
  by_cluster: Record<string, number>;
  by_folder: Record<string, number>;
}

export interface PolicyRule {
  field: string;
  operator: string;
  value: unknown;
  message?: string;
}

export interface Policy {
  name: string;
  type: string;
  description?: string;
  rules: PolicyRule[];
  severity: "enforce" | "warn" | "info";
}

export interface PolicyViolation {
  policy: Policy;
  rule: PolicyRule;
  vm: VirtualMachine;
  current_value: unknown;
  severity: "enforce" | "warn" | "info";
  remediation?: string;
}

export interface PolicyReport {
  policies: Policy[];
  violations: PolicyViolation[];
  compliant_vms: number;
  non_compliant_vms: number;
  waived_violations: number;
  evaluated_at: string;
}

export interface DriftEvent {
  type: "added" | "removed" | "modified" | "policy_violation";
  vm: VirtualMachine;
  field?: string;
  old_value?: unknown;
  new_value?: unknown;
  severity: "critical" | "warning" | "info";
  detected_at: string;
}

export interface DriftReport {
  baseline: SnapshotMeta;
  current: SnapshotMeta;
  events: DriftEvent[];
  added_vms: number;
  removed_vms: number;
  modified_vms: number;
  policy_drifts: number;
  evaluated_at: string;
}

export interface SnapshotMeta {
  id: string;
  source: string;
  platform: Platform;
  vm_count: number;
  discovered_at: string;
}

export interface GraphNode {
  id: string;
  label: string;
  type: "vm" | "network" | "datastore" | "backup-job";
  platform?: Platform;
  metadata?: Record<string, string>;
}

export interface GraphEdge {
  source: string;
  target: string;
  type: "network" | "storage" | "backup";
  label: string;
}

export interface DependencyGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface WorkloadMigration {
  vm: VirtualMachine;
  phase: MigrationPhase;
  source_disk_paths?: string[];
  converted_disk_paths?: string[];
  target_vm_id?: string;
  network_mappings?: Record<string, string>;
  error?: string;
}

export interface ExecutionWindow {
  not_before?: string;
  not_after?: string;
}

export interface ApprovalGate {
  required?: boolean;
  approved_by?: string;
  approved_at?: string;
  ticket?: string;
}

export interface WaveStrategy {
  size?: number;
  dependency_aware?: boolean;
}

export interface PlannedWorkload {
  vm_id: string;
  name: string;
  dependencies?: string[];
  target_host?: string;
  target_storage?: string;
  network_map?: Record<string, string>;
}

export interface MigrationWave {
  index: number;
  reason: string;
  dependency_aware: boolean;
  workloads: PlannedWorkload[];
}

export interface MigrationPlan {
  generated_at: string;
  total_workloads: number;
  window?: ExecutionWindow;
  requires_approval?: boolean;
  approval_satisfied: boolean;
  wave_strategy: WaveStrategy;
  waves: MigrationWave[];
}

export interface MigrationCheckpoint {
  phase: MigrationPhase;
  status: CheckpointStatus;
  started_at?: string;
  completed_at?: string;
  message?: string;
  diagnostics?: string[];
}

export interface MigrationState {
  id: string;
  spec_name: string;
  source_address?: string;
  source_platform?: Platform;
  target_address?: string;
  target_platform?: Platform;
  phase: MigrationPhase;
  workloads: WorkloadMigration[];
  plan?: MigrationPlan;
  checkpoints?: MigrationCheckpoint[];
  window?: ExecutionWindow;
  approval?: ApprovalGate;
  pending_approval?: boolean;
  started_at: string;
  updated_at: string;
  completed_at?: string;
  errors?: string[];
}

export interface MigrationMeta {
  id: string;
  spec_name: string;
  phase: MigrationPhase;
  started_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface MigrationExecutionRequest {
  approved_by?: string;
  ticket?: string;
}

export interface MigrationCommandResponse {
  migration_id: string;
  status: string;
}

export interface RollbackResult {
  migration_id: string;
  target_vms_removed: number;
  files_cleaned_up: number;
  source_vms_restored: number;
  errors?: string[];
  duration: number;
}

export interface CheckResult {
  name: string;
  status: "pass" | "warn" | "fail";
  message: string;
  duration: number;
}

export interface PreflightReport {
  plan?: MigrationPlan;
  checks: CheckResult[];
  pass_count: number;
  warn_count: number;
  fail_count: number;
  can_proceed: boolean;
}

export interface MigrationSpec {
  name: string;
  source: {
    address: string;
    platform: Platform;
    credential_ref?: string;
  };
  target: {
    address: string;
    platform: Platform;
    credential_ref?: string;
    default_host?: string;
    default_storage?: string;
  };
  workloads: Array<{
    match: {
      name_pattern?: string;
      tags?: Record<string, string>;
      folder?: string;
      power_state?: PowerState;
      exclude?: string[];
    };
    overrides?: {
      target_host?: string;
      target_storage?: string;
      network_map?: Record<string, string>;
      storage_map?: Record<string, string>;
      dependencies?: string[];
    };
  }>;
  options: {
    dry_run?: boolean;
    parallel?: number;
    shutdown_source?: boolean;
    remove_source?: boolean;
    verify_boot?: boolean;
    window?: ExecutionWindow;
    approval?: ApprovalGate;
    waves?: WaveStrategy;
  };
}

export interface GraphFilters {
  nodeTypes: Record<string, boolean>;
  platform: string;
}

export interface RemediationRecommendation {
  vm: VirtualMachine;
  type: "placement" | "policy" | "drift";
  severity: string;
  summary: string;
  action: string;
  target_platform?: Platform;
  monthly_savings?: number;
  waiver_eligible?: boolean;
  sources?: string[];
}

export interface RecommendationReport {
  recommendations: RemediationRecommendation[];
  generated_at: string;
}

export interface SimulationRequest {
  target_platform: Platform;
  vm_ids?: string[];
  include_all?: boolean;
}

export interface SimulationResult {
  target_platform: Platform;
  moved_vms: number;
  current_monthly_cost: number;
  simulated_monthly_cost: number;
  monthly_delta: number;
  policy_report?: PolicyReport;
  recommendation_report?: RecommendationReport;
  simulated_inventory?: DiscoveryResult;
}

export interface TenantQuota {
  requests_per_minute?: number;
  max_snapshots?: number;
  max_migrations?: number;
}

export interface TenantSummary {
  tenant_id: string;
  workload_count: number;
  snapshot_count: number;
  active_migrations: number;
  completed_migrations: number;
  failed_migrations: number;
  pending_approvals: number;
  recommendation_count: number;
  platform_counts: Record<string, number>;
  last_discovery_at?: string;
  quotas?: TenantQuota;
  snapshot_quota_free?: number;
  migration_quota_free?: number;
}

export interface AboutResponse {
  name: string;
  api_version: string;
  version: string;
  commit: string;
  built_at: string;
  go_version: string;
  plugin_protocol: string;
  supported_platforms: string[];
  supported_permissions: TenantPermission[];
  store_backend: string;
  store_schema_version?: number;
  persistent_store: boolean;
}

export interface CurrentTenant {
  tenant_id: string;
  name: string;
  active: boolean;
  settings?: Record<string, string>;
  quotas?: TenantQuota;
  role: TenantRole;
  permissions: TenantPermission[];
  auth_method: string;
  service_account_id?: string;
  service_account_name?: string;
  service_account_count: number;
}
