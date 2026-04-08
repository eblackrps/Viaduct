import { useEffect, useRef, useState } from "react";
import { getInventory, getSnapshots, getTenantSummary, listMigrations } from "../api";
import type { DiscoveryResult, MigrationMeta, SnapshotMeta, TenantSummary } from "../types";

export interface OperatorOverviewErrors {
  inventory?: string;
  snapshots?: string;
  migrations?: string;
  summary?: string;
}

export interface OperatorOverviewState {
  inventory: DiscoveryResult | null;
  snapshots: SnapshotMeta[];
  latestSnapshot: SnapshotMeta | null;
  migrations: MigrationMeta[];
  summary: TenantSummary | null;
  loading: boolean;
  refreshing: boolean;
  refreshToken: number;
  errors: OperatorOverviewErrors;
  error: string | null;
  refresh: () => Promise<void>;
}

export function useOperatorOverview(): OperatorOverviewState {
  const [inventory, setInventory] = useState<DiscoveryResult | null>(null);
  const [snapshots, setSnapshots] = useState<SnapshotMeta[]>([]);
  const [migrations, setMigrations] = useState<MigrationMeta[]>([]);
  const [summary, setSummary] = useState<TenantSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);
  const [errors, setErrors] = useState<OperatorOverviewErrors>({});
  const [error, setError] = useState<string | null>(null);
  const hasLoadedRef = useRef(false);
  const requestSequenceRef = useRef(0);

  async function refresh() {
    const requestSequence = requestSequenceRef.current + 1;
    requestSequenceRef.current = requestSequence;
    const initialLoad = !hasLoadedRef.current;
    setLoading(initialLoad);
    setRefreshing(!initialLoad);

    const nextErrors: OperatorOverviewErrors = {};
    const [inventoryResult, snapshotResult, migrationResult, summaryResult] = await Promise.allSettled([
      getInventory(),
      getSnapshots(),
      listMigrations(),
      getTenantSummary(),
    ]);

    if (requestSequence !== requestSequenceRef.current) {
      return;
    }

    if (inventoryResult.status === "fulfilled") {
      setInventory(inventoryResult.value);
    } else {
      nextErrors.inventory = errorMessage("inventory", inventoryResult.reason);
    }

    if (snapshotResult.status === "fulfilled") {
      setSnapshots(snapshotResult.value);
    } else {
      nextErrors.snapshots = errorMessage("discovery snapshots", snapshotResult.reason);
    }

    if (migrationResult.status === "fulfilled") {
      setMigrations(migrationResult.value);
    } else {
      nextErrors.migrations = errorMessage("migration history", migrationResult.reason);
    }

    if (summaryResult.status === "fulfilled") {
      setSummary(summaryResult.value);
    } else {
      nextErrors.summary = errorMessage("tenant summary", summaryResult.reason);
    }

    const settledErrors: OperatorOverviewErrors = {
      inventory: inventoryResult.status === "rejected" ? nextErrors.inventory : undefined,
      snapshots: snapshotResult.status === "rejected" ? nextErrors.snapshots : undefined,
      migrations: migrationResult.status === "rejected" ? nextErrors.migrations : undefined,
      summary: summaryResult.status === "rejected" ? nextErrors.summary : undefined,
    };
    setErrors(settledErrors);
    setError(composeGlobalError(settledErrors));
    hasLoadedRef.current = true;
    if (requestSequence === requestSequenceRef.current) {
      if (!initialLoad) {
        setRefreshToken((current) => current + 1);
      }
      setLoading(false);
      setRefreshing(false);
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  return {
    inventory,
    snapshots,
    latestSnapshot: snapshots[0] ?? null,
    migrations,
    summary,
    loading,
    refreshing,
    refreshToken,
    errors,
    error,
    refresh,
  };
}

function composeGlobalError(errors: OperatorOverviewErrors): string | null {
  const primaryMessages = [errors.inventory, errors.snapshots, errors.migrations, errors.summary].filter(Boolean);
  return primaryMessages.length > 0 ? primaryMessages.join(" ") : null;
}

function errorMessage(scope: string, reason: unknown): string {
  if (reason instanceof Error && reason.message.trim() !== "") {
    return `Unable to load ${scope}: ${reason.message}`;
  }
  return `Unable to load ${scope}.`;
}
