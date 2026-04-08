import type { Platform, VirtualMachine } from "../../types";
import { getVirtualMachineIdentity } from "./workloadIdentity";

const inventoryPlanningDraftKey = "viaduct.inventoryPlanningDraft";
const inventoryPlanningDraftVersion = 1;
const maxDraftAgeMs = 24 * 60 * 60 * 1000;

export interface InventoryPlanningDraft {
  version: number;
  createdAt: string;
  sourcePlatform: Platform;
  workloadKeys: string[];
  workloads: VirtualMachine[];
}

export function loadInventoryPlanningDraft(): InventoryPlanningDraft | null {
  if (typeof window === "undefined") {
    return null;
  }

  const payload = window.sessionStorage.getItem(inventoryPlanningDraftKey);
  if (!payload) {
    return null;
  }

  try {
    const draft = JSON.parse(payload) as InventoryPlanningDraft;
    if (!draft || !Array.isArray(draft.workloads) || typeof draft.sourcePlatform !== "string") {
      window.sessionStorage.removeItem(inventoryPlanningDraftKey);
      return null;
    }
    const createdAt = typeof draft.createdAt === "string" ? draft.createdAt : new Date().toISOString();
    const createdAtMs = new Date(createdAt).getTime();
    if (Number.isNaN(createdAtMs) || Date.now() - createdAtMs > maxDraftAgeMs) {
      window.sessionStorage.removeItem(inventoryPlanningDraftKey);
      return null;
    }

    return {
      version: typeof draft.version === "number" ? draft.version : inventoryPlanningDraftVersion,
      createdAt,
      sourcePlatform: draft.sourcePlatform,
      workloadKeys:
        Array.isArray(draft.workloadKeys) && draft.workloadKeys.length > 0
          ? draft.workloadKeys
          : draft.workloads.map((vm) => getVirtualMachineIdentity(vm)),
      workloads: draft.workloads,
    };
  } catch {
    window.sessionStorage.removeItem(inventoryPlanningDraftKey);
    return null;
  }
}

export function saveInventoryPlanningDraft(draft: InventoryPlanningDraft): void {
  if (typeof window === "undefined") {
    return;
  }

  window.sessionStorage.setItem(
    inventoryPlanningDraftKey,
    JSON.stringify({
      ...draft,
      version: inventoryPlanningDraftVersion,
      workloadKeys: draft.workloadKeys.length > 0 ? draft.workloadKeys : draft.workloads.map((vm) => getVirtualMachineIdentity(vm)),
    }),
  );
}

export function clearInventoryPlanningDraft(): void {
  if (typeof window === "undefined") {
    return;
  }

  window.sessionStorage.removeItem(inventoryPlanningDraftKey);
}
