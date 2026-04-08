import type { StatusTone } from "../../components/primitives/StatusBadge";
import type { CheckResult, MigrationMeta, MigrationPhase, MigrationState, PreflightReport } from "../../types";

export type MigrationWorkflowStatus =
  | "draft"
  | "ready"
  | "warning"
  | "blocked"
  | "running"
  | "failed"
  | "completed"
  | "rolled-back";

export type PersistedMigrationListStatus =
  | "planned"
  | "running"
  | "failed"
  | "completed"
  | "rolled-back";

interface WorkflowStatusPresentation {
  label: string;
  tone: StatusTone;
  description: string;
}

const workflowStatusPresentation: Record<MigrationWorkflowStatus, WorkflowStatusPresentation> = {
  draft: {
    label: "Draft",
    tone: "neutral",
    description: "The migration scope is being prepared and has not been validated for execution yet.",
  },
  ready: {
    label: "Ready",
    tone: "success",
    description: "The current plan can proceed based on the latest validation or saved plan state.",
  },
  warning: {
    label: "Warning",
    tone: "warning",
    description: "The plan is usable, but operators still need to review non-blocking risks or pending approvals.",
  },
  blocked: {
    label: "Blocked",
    tone: "danger",
    description: "Execution is blocked until the current plan or environment issues are resolved.",
  },
  running: {
    label: "Running",
    tone: "accent",
    description: "Viaduct is actively executing or resuming migration phases for this run.",
  },
  failed: {
    label: "Failed",
    tone: "danger",
    description: "The run encountered a failure and requires operator action before it can continue.",
  },
  completed: {
    label: "Completed",
    tone: "success",
    description: "The migration finished successfully and the recorded run is complete.",
  },
  "rolled-back": {
    label: "Rolled back",
    tone: "neutral",
    description: "Rollback completed and the run is no longer in progress.",
  },
};

const checkNameLabels: Record<string, string> = {
  "source-connectivity": "Source connectivity",
  "target-connectivity": "Target connectivity",
  "execution-window": "Execution window",
  "approval-gate": "Approval gate",
  "disk-space": "Target disk capacity",
  "network-mappings": "Network mappings",
  "name-conflicts": "Target name conflicts",
  "source-backup": "Source recovery points",
  "disk-formats": "Disk conversion path",
  "resource-availability": "Target compute capacity",
  "rollback-readiness": "Rollback readiness",
  "execution-plan": "Execution runbook",
};

export function getWorkflowStatusPresentation(status: MigrationWorkflowStatus): WorkflowStatusPresentation {
  return workflowStatusPresentation[status];
}

export function getMigrationWorkflowStatus(
  migration: Pick<MigrationMeta, "phase"> | Pick<MigrationState, "phase" | "pending_approval" | "errors"> | null | undefined,
): MigrationWorkflowStatus {
  if (!migration) {
    return "draft";
  }

  switch (migration.phase) {
    case "complete":
      return "completed";
    case "failed":
      return "failed";
    case "rolled_back":
      return "rolled-back";
    case "plan":
      if ("errors" in migration && (migration.errors?.length ?? 0) > 0) {
        return "blocked";
      }
      if ("pending_approval" in migration && migration.pending_approval) {
        return "warning";
      }
      return "ready";
    default:
      return "running";
  }
}

export function getLocalPlanningWorkflowStatus({
  selectedWorkloadCount,
  hasSourceEndpoint,
  hasTargetEndpoint,
  preflight,
  preflightStale = false,
  planStale = false,
  migrationState,
}: {
  selectedWorkloadCount: number;
  hasSourceEndpoint: boolean;
  hasTargetEndpoint: boolean;
  preflight: PreflightReport | null;
  preflightStale?: boolean;
  planStale?: boolean;
  migrationState?: MigrationState | null;
}): MigrationWorkflowStatus {
  if (migrationState) {
    if (planStale) {
      return "blocked";
    }
    return getMigrationWorkflowStatus(migrationState);
  }

  if (preflight) {
    if (preflightStale) {
      return "warning";
    }
    if (preflight.fail_count > 0) {
      return "blocked";
    }
    if (preflight.warn_count > 0) {
      return "warning";
    }
    if (preflight.can_proceed) {
      return "ready";
    }
  }

  if (selectedWorkloadCount === 0 || !hasSourceEndpoint || !hasTargetEndpoint) {
    return "draft";
  }

  return "draft";
}

export function countMigrationsByWorkflowStatus(migrations: MigrationMeta[]): Record<MigrationWorkflowStatus, number> {
  const counts: Record<MigrationWorkflowStatus, number> = {
    draft: 0,
    ready: 0,
    warning: 0,
    blocked: 0,
    running: 0,
    failed: 0,
    completed: 0,
    "rolled-back": 0,
  };

  for (const migration of migrations) {
    counts[getMigrationWorkflowStatus(migration)] += 1;
  }

  return counts;
}

export function getPersistedMigrationListStatus(phase: MigrationPhase): PersistedMigrationListStatus {
  switch (phase) {
    case "plan":
      return "planned";
    case "failed":
      return "failed";
    case "complete":
      return "completed";
    case "rolled_back":
      return "rolled-back";
    default:
      return "running";
  }
}

export function getPersistedMigrationListPresentation(status: PersistedMigrationListStatus): WorkflowStatusPresentation {
  switch (status) {
    case "planned":
      return {
        label: "Plan saved",
        tone: "neutral",
        description: "The migration plan is persisted, but list metadata alone does not expose readiness or approval details.",
      };
    case "running":
      return workflowStatusPresentation.running;
    case "failed":
      return workflowStatusPresentation.failed;
    case "completed":
      return workflowStatusPresentation.completed;
    case "rolled-back":
      return workflowStatusPresentation["rolled-back"];
    default:
      return workflowStatusPresentation.draft;
  }
}

export function countPersistedMigrationListStatuses(migrations: MigrationMeta[]): Record<PersistedMigrationListStatus, number> {
  const counts: Record<PersistedMigrationListStatus, number> = {
    planned: 0,
    running: 0,
    failed: 0,
    completed: 0,
    "rolled-back": 0,
  };

  for (const migration of migrations) {
    counts[getPersistedMigrationListStatus(migration.phase)] += 1;
  }

  return counts;
}

export function describeMigrationPhase(phase: MigrationPhase): string {
  switch (phase) {
    case "plan":
      return "Plan saved";
    case "export":
      return "Exporting source data";
    case "convert":
      return "Converting disks";
    case "import":
      return "Importing workloads";
    case "configure":
      return "Applying target settings";
    case "verify":
      return "Verifying target boot";
    case "complete":
      return "Migration complete";
    case "failed":
      return "Execution failed";
    case "rolled_back":
      return "Rollback completed";
    default:
      return phase;
  }
}

export function isMigrationExecutionActivePhase(phase: MigrationPhase): boolean {
  return ["export", "convert", "import", "configure", "verify"].includes(phase);
}

export function isMigrationTerminalPhase(phase: MigrationPhase): boolean {
  return ["complete", "failed", "rolled_back"].includes(phase);
}

export function getCheckTone(status: CheckResult["status"]): StatusTone {
  switch (status) {
    case "pass":
      return "success";
    case "warn":
      return "warning";
    case "fail":
      return "danger";
    default:
      return "neutral";
  }
}

export function getReadableCheckName(name: string): string {
  return checkNameLabels[name] ?? name.replace(/-/g, " ");
}

export function getPreflightSummary(report: PreflightReport | null, stale = false): string {
  if (!report) {
    return "Preflight has not been run for this draft yet.";
  }
  if (stale) {
    return "Preflight was run, but the draft changed afterwards and needs validation again.";
  }
  if (report.fail_count > 0) {
    return `${report.fail_count} blocking check(s) must be resolved before execution can proceed.`;
  }
  if (report.warn_count > 0) {
    return `${report.warn_count} warning check(s) still need operator review before execution.`;
  }
  return "The latest preflight checks passed and the draft is ready for execution review.";
}
