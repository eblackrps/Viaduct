import { useEffect, useMemo, useRef, useState } from "react";
import {
  createMigration,
  executeMigration,
  getInventory,
  getMigrationState,
  resumeMigration,
  rollbackMigration,
  runPreflight,
} from "../../api";
import type {
  DiscoveryResult,
  MigrationState,
  Platform,
  PreflightReport,
  RollbackResult,
} from "../../types";
import type { InventoryPlanningDraft } from "../inventory/inventoryPlanningDraft";
import { clearInventoryPlanningDraft } from "../inventory/inventoryPlanningDraft";
import { getVirtualMachineIdentity } from "../inventory/workloadIdentity";
import {
  buildExecutionPayload,
  buildMigrationSpec,
  resolveExecutionWindowState,
  validateMigrationDraft,
} from "./migrationPlanningModel";
import {
  getLocalPlanningWorkflowStatus,
  getWorkflowStatusPresentation,
  isMigrationExecutionActivePhase,
  isMigrationTerminalPhase,
} from "./migrationStatus";

interface UseMigrationWorkspaceOptions {
  planningDraft?: InventoryPlanningDraft | null;
  onPlanningDraftCleared?: () => void;
  onMigrationChange?: () => void;
}

export function useMigrationWorkspace({
  planningDraft,
  onPlanningDraftCleared,
  onMigrationChange,
}: UseMigrationWorkspaceOptions) {
  const [stage, setStage] = useState(0);
  const [migrationName, setMigrationName] = useState("dashboard-migration");
  const [sourcePlatform, setSourcePlatform] = useState<Platform>("vmware");
  const [sourceAddress, setSourceAddress] = useState("");
  const [inventory, setInventory] = useState<DiscoveryResult | null>(null);
  const [selectedWorkloadKeys, setSelectedWorkloadKeys] = useState<string[]>([]);
  const [selectionSearch, setSelectionSearch] = useState("");
  const [targetPlatform, setTargetPlatform] = useState<Platform>("proxmox");
  const [targetAddress, setTargetAddress] = useState("");
  const [defaultHost, setDefaultHost] = useState("pve-01");
  const [defaultStorage, setDefaultStorage] = useState("local-lvm");
  const [networkMap, setNetworkMap] = useState<Record<string, string>>({});
  const [parallelism, setParallelism] = useState(2);
  const [waveSize, setWaveSize] = useState(2);
  const [dependencyAware, setDependencyAware] = useState(true);
  const [shutdownSource, setShutdownSource] = useState(true);
  const [verifyBoot, setVerifyBoot] = useState(true);
  const [approvalRequired, setApprovalRequired] = useState(false);
  const [approvedBy, setApprovedBy] = useState("");
  const [approvalTicket, setApprovalTicket] = useState("");
  const [approvalRecordedAt, setApprovalRecordedAt] = useState("");
  const [scheduledStart, setScheduledStart] = useState("");
  const [scheduledEnd, setScheduledEnd] = useState("");
  const [preflight, setPreflight] = useState<PreflightReport | null>(null);
  const [preflightSpecKey, setPreflightSpecKey] = useState<string | null>(null);
  const [migrationState, setMigrationState] = useState<MigrationState | null>(null);
  const [migrationID, setMigrationID] = useState<string | null>(null);
  const [plannedSpecKey, setPlannedSpecKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [refreshingState, setRefreshingState] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [importedDraft, setImportedDraft] = useState<InventoryPlanningDraft | null>(planningDraft ?? null);
  const [draftNotice, setDraftNotice] = useState<string | null>(null);
  const [executionRequested, setExecutionRequested] = useState(false);
  const [rollbackResult, setRollbackResult] = useState<RollbackResult | null>(null);
  const inventoryRequestSequenceRef = useRef(0);
  const stateRefreshSequenceRef = useRef(0);

  const selectedWorkloads = useMemo(
    () => (inventory?.vms ?? []).filter((vm) => selectedWorkloadKeys.includes(getVirtualMachineIdentity(vm))),
    [inventory?.vms, selectedWorkloadKeys],
  );
  const filteredWorkloads = useMemo(
    () =>
      (inventory?.vms ?? []).filter((vm) =>
        [vm.name, vm.platform, vm.host, vm.cluster ?? "", vm.folder ?? ""]
          .join(" ")
          .toLowerCase()
          .includes(selectionSearch.trim().toLowerCase()),
      ),
    [inventory?.vms, selectionSearch],
  );
  const sourceNetworks = useMemo(
    () =>
      Array.from(
        new Set(
          selectedWorkloads.flatMap((vm) =>
            vm.nics.map((nic) => nic.network.trim()).filter(Boolean),
          ),
        ),
      ).sort((left, right) => left.localeCompare(right)),
    [selectedWorkloads],
  );

  const draftSpec = useMemo(
    () =>
      buildMigrationSpec({
        migrationName,
        sourcePlatform,
        sourceAddress,
        targetPlatform,
        targetAddress,
        defaultHost,
        defaultStorage,
        selectedWorkloads,
        networkMap,
        parallelism,
        waveSize,
        dependencyAware,
        shutdownSource,
        verifyBoot,
        scheduledStart,
        scheduledEnd,
        approvalRequired,
        approvedBy,
        approvalTicket,
        approvalRecordedAt,
      }),
    [
      approvalRecordedAt,
      approvalRequired,
      approvalTicket,
      approvedBy,
      defaultHost,
      defaultStorage,
      dependencyAware,
      migrationName,
      networkMap,
      parallelism,
      scheduledEnd,
      scheduledStart,
      selectedWorkloads,
      shutdownSource,
      sourceAddress,
      sourcePlatform,
      targetAddress,
      targetPlatform,
      verifyBoot,
      waveSize,
    ],
  );
  const specFingerprint = useMemo(() => JSON.stringify(draftSpec), [draftSpec]);
  const preflightStale = Boolean(preflight && preflightSpecKey !== specFingerprint);
  const planStale = Boolean(migrationState && plannedSpecKey && plannedSpecKey !== specFingerprint);
  const validationError = validateMigrationDraft(selectedWorkloads.length, sourceAddress, targetAddress);
  const windowState = resolveExecutionWindowState(scheduledStart, scheduledEnd);
  const workflowStatus = getLocalPlanningWorkflowStatus({
    selectedWorkloadCount: selectedWorkloads.length,
    hasSourceEndpoint: sourceAddress.trim() !== "",
    hasTargetEndpoint: targetAddress.trim() !== "",
    preflight,
    preflightStale,
    planStale,
    migrationState,
  });
  const workflowPresentation = getWorkflowStatusPresentation(workflowStatus);
  const executionBlockers = [
    ...(planStale ? ["Draft inputs changed after this plan was saved. Save a new plan before executing."] : []),
    ...((migrationState?.errors ?? []).map((item) => `Saved plan issue: ${item}`)),
    ...(migrationState?.pending_approval ? ["This saved plan is waiting on approval before execution can proceed."] : []),
    ...(!migrationState?.pending_approval && approvalRequired && approvedBy.trim() === "" && migrationState?.phase === "plan"
      ? ["Execution approval is still required."]
      : []),
    ...(windowState.kind === "not-started" ? [`Execution window opens at ${windowState.label}.`] : []),
    ...(windowState.kind === "closed" ? [`Execution window closed at ${windowState.label}.`] : []),
  ];
  const executionAdvisories = [
    ...(preflightStale ? ["Validation results are stale for the current draft. Rerun preflight before treating this plan as ready."] : []),
    ...(!preflightStale && preflight?.fail_count
      ? [`Latest preflight still reports ${preflight.fail_count} blocking check(s). The API does not hard-stop execution here, but the run is not operationally ready yet.`]
      : []),
    ...(!preflightStale && preflight && preflight.fail_count === 0 && preflight.warn_count > 0
      ? [`Latest preflight reports ${preflight.warn_count} warning check(s) that still need operator review.`]
      : []),
  ];
  const isPolling = Boolean(migrationState && executionRequested && isMigrationExecutionActivePhase(migrationState.phase));

  useEffect(() => {
    if (!planningDraft) {
      setImportedDraft(null);
      return;
    }
    if (planningDraft.createdAt === importedDraft?.createdAt) {
      return;
    }

    setImportedDraft(planningDraft);
    setSourcePlatform(planningDraft.sourcePlatform);
    setInventory({ source: "inventory selection", platform: planningDraft.sourcePlatform, vms: planningDraft.workloads });
    setSelectedWorkloadKeys(
      planningDraft.workloadKeys.length > 0
        ? planningDraft.workloadKeys
        : planningDraft.workloads.map((vm) => getVirtualMachineIdentity(vm)),
    );
    setSelectionSearch("");
    setStage(0);
    setDraftNotice(null);
    resetPlanningArtifacts();
  }, [importedDraft?.createdAt, planningDraft]);

  useEffect(() => {
    setNetworkMap((current) =>
      Object.fromEntries(Object.entries(current).filter(([name]) => sourceNetworks.includes(name))),
    );
  }, [sourceNetworks]);

  useEffect(() => {
    if (!migrationID || !migrationState || !isPolling) {
      return;
    }

    const interval = window.setInterval(() => {
      void refreshSavedState({ silent: true });
    }, 2000);

    return () => window.clearInterval(interval);
  }, [isPolling, migrationID, migrationState]);

  function resetPlanningArtifacts() {
    setPreflight(null);
    setPreflightSpecKey(null);
    setMigrationState(null);
    setMigrationID(null);
    setPlannedSpecKey(null);
    setExecutionRequested(false);
    setRefreshingState(false);
    setRollbackResult(null);
    setError(null);
  }

  function clearImportedDraftState({
    clearSelection = false,
    clearInventory = false,
  }: {
    clearSelection?: boolean;
    clearInventory?: boolean;
  } = {}) {
    clearInventoryPlanningDraft();
    setImportedDraft(null);
    setDraftNotice(null);
    if (clearSelection) {
      setSelectedWorkloadKeys([]);
      setSelectionSearch("");
    }
    if (clearInventory && inventory?.source === "inventory selection") {
      setInventory(null);
    }
    onPlanningDraftCleared?.();
  }

  async function refreshSavedState(options?: { silent?: boolean }) {
    if (!migrationID) {
      return;
    }

    const requestSequence = stateRefreshSequenceRef.current + 1;
    stateRefreshSequenceRef.current = requestSequence;
    if (!options?.silent) {
      setRefreshingState(true);
    }

    try {
      const nextState = await getMigrationState(migrationID);
      if (requestSequence !== stateRefreshSequenceRef.current) {
        return;
      }
      setMigrationState(nextState);
      if (isMigrationTerminalPhase(nextState.phase)) {
        setExecutionRequested(false);
      }
      onMigrationChange?.();
    } catch (reason) {
      if (requestSequence === stateRefreshSequenceRef.current) {
        setError((reason as Error).message);
      }
    } finally {
      if (!options?.silent && requestSequence === stateRefreshSequenceRef.current) {
        setRefreshingState(false);
      }
    }
  }

  async function loadInventory() {
    const requestSequence = inventoryRequestSequenceRef.current + 1;
    inventoryRequestSequenceRef.current = requestSequence;
    setLoading(true);
    setError(null);

    try {
      const result = await getInventory(sourcePlatform);
      if (requestSequence !== inventoryRequestSequenceRef.current) {
        return;
      }

      setInventory(result);

      if (!importedDraft) {
        const availableKeys = new Set(result.vms.map((vm) => getVirtualMachineIdentity(vm)));
        setSelectedWorkloadKeys((current) => current.filter((key) => availableKeys.has(key)));
        setDraftNotice(null);
      } else {
        const imported = new Set(importedDraft.workloadKeys);
        const matchedKeys = result.vms
          .filter((vm) => imported.has(getVirtualMachineIdentity(vm)))
          .map((vm) => getVirtualMachineIdentity(vm));
        setSelectedWorkloadKeys(matchedKeys);

        if (matchedKeys.length === 0) {
          setDraftNotice("The imported inventory draft did not match the refreshed platform inventory. Review the workload scope before continuing.");
        } else if (matchedKeys.length < importedDraft.workloadKeys.length) {
          setDraftNotice(`${matchedKeys.length} of ${importedDraft.workloadKeys.length} imported workloads were matched after refresh. Review the workload scope before continuing.`);
        } else {
          setDraftNotice(null);
        }
      }
    } catch (reason) {
      if (requestSequence === inventoryRequestSequenceRef.current) {
        setError((reason as Error).message);
      }
    } finally {
      if (requestSequence === inventoryRequestSequenceRef.current) {
        setLoading(false);
      }
    }
  }

  async function handlePreflight() {
    if (validationError) {
      setError(validationError);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const report = await runPreflight(draftSpec);
      setPreflight(report);
      setPreflightSpecKey(specFingerprint);
      setStage(2);
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleSavePlan() {
    if (validationError) {
      setError(validationError);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const planned = await createMigration(draftSpec);
      setMigrationState(planned);
      setMigrationID(planned.id);
      setPlannedSpecKey(specFingerprint);
      setExecutionRequested(false);
      setRollbackResult(null);
      clearImportedDraftState();
      onMigrationChange?.();
      setStage(3);
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleExecute() {
    if (!migrationID) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await executeMigration(migrationID, buildExecutionPayload(approvedBy, approvalTicket));
      setExecutionRequested(true);
      setRollbackResult(null);
      setStage(3);
      onMigrationChange?.();
      void refreshSavedState();
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleResume() {
    if (!migrationID) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await resumeMigration(migrationID, buildExecutionPayload(approvedBy, approvalTicket));
      setExecutionRequested(true);
      setRollbackResult(null);
      setStage(3);
      onMigrationChange?.();
      void refreshSavedState();
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function handleRollback() {
    if (!migrationID) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await rollbackMigration(migrationID);
      setExecutionRequested(false);
      setRollbackResult(result);
      await refreshSavedState();
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function clearImportedSelection() {
    resetPlanningArtifacts();
    clearImportedDraftState({ clearSelection: true, clearInventory: true });
  }

  function changeSourcePlatform(nextPlatform: Platform) {
    setSourcePlatform(nextPlatform);
    if (inventory?.platform !== nextPlatform) {
      resetPlanningArtifacts();
      setInventory(null);
      setSelectedWorkloadKeys([]);
      setSelectionSearch("");
      clearImportedDraftState();
    }
  }

  return {
    stage,
    setStage,
    migrationName,
    setMigrationName,
    sourcePlatform,
    changeSourcePlatform,
    sourceAddress,
    setSourceAddress,
    inventory,
    selectedWorkloadKeys,
    setSelectedWorkloadKeys,
    selectionSearch,
    setSelectionSearch,
    filteredWorkloads,
    selectedWorkloads,
    targetPlatform,
    setTargetPlatform,
    targetAddress,
    setTargetAddress,
    defaultHost,
    setDefaultHost,
    defaultStorage,
    setDefaultStorage,
    networkMap,
    setNetworkMap,
    sourceNetworks,
    parallelism,
    setParallelism,
    waveSize,
    setWaveSize,
    dependencyAware,
    setDependencyAware,
    shutdownSource,
    setShutdownSource,
    verifyBoot,
    setVerifyBoot,
    approvalRequired,
    setApprovalRequired,
    approvedBy,
    setApprovedBy,
    approvalTicket,
    setApprovalTicket,
    approvalRecordedAt,
    setApprovalRecordedAt,
    scheduledStart,
    setScheduledStart,
    scheduledEnd,
    setScheduledEnd,
    preflight,
    preflightStale,
    migrationState,
    planStale,
    loading,
    refreshingState,
    error,
    draftNotice,
    importedDraft,
    rollbackResult,
    workflowPresentation,
    validationError,
    windowState,
    executionBlockers,
    executionAdvisories,
    isPolling,
    loadInventory,
    handlePreflight,
    handleSavePlan,
    handleExecute,
    handleResume,
    handleRollback,
    refreshSavedState,
    clearImportedSelection,
  };
}

export type MigrationWorkspaceState = ReturnType<typeof useMigrationWorkspace>;
