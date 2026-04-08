import { useEffect, useRef, useState } from "react";
import { getCosts, getDrift, getPolicies, getRemediation, runSimulation } from "../../api";
import type {
  DriftReport,
  Platform,
  PlatformComparison,
  PolicyReport,
  RecommendationReport,
  SimulationResult,
} from "../../types";

interface LifecycleSignalOptions {
  baselineId?: string | null;
  refreshToken?: number;
  includeCosts?: boolean;
  includePolicies?: boolean;
  includeDrift?: boolean;
  includeRemediation?: boolean;
}

export interface LifecycleSignalErrors {
  costs?: string;
  policies?: string;
  drift?: string;
  remediation?: string;
  simulation?: string;
}

export interface LifecycleSignalState {
  costs: PlatformComparison[];
  policies: PolicyReport | null;
  drift: DriftReport | null;
  remediation: RecommendationReport | null;
  simulation: SimulationResult | null;
  loading: boolean;
  refreshing: boolean;
  simulationLoading: boolean;
  errors: LifecycleSignalErrors;
  refresh: () => Promise<void>;
  simulate: (targetPlatform: Platform) => Promise<void>;
}

export function useLifecycleSignals({
  baselineId,
  refreshToken = 0,
  includeCosts = false,
  includePolicies = false,
  includeDrift = false,
  includeRemediation = false,
}: LifecycleSignalOptions): LifecycleSignalState {
  const [costs, setCosts] = useState<PlatformComparison[]>([]);
  const [policies, setPolicies] = useState<PolicyReport | null>(null);
  const [drift, setDrift] = useState<DriftReport | null>(null);
  const [remediation, setRemediation] = useState<RecommendationReport | null>(null);
  const [simulation, setSimulation] = useState<SimulationResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [simulationLoading, setSimulationLoading] = useState(false);
  const [errors, setErrors] = useState<LifecycleSignalErrors>({});
  const hasLoadedRef = useRef(false);
  const previousBaselineIdRef = useRef<string | null | undefined>(baselineId);
  const requestSequenceRef = useRef(0);

  async function refresh() {
    const requestSequence = requestSequenceRef.current + 1;
    requestSequenceRef.current = requestSequence;
    const initialLoad = !hasLoadedRef.current;
    const baselineChanged = previousBaselineIdRef.current !== baselineId;
    previousBaselineIdRef.current = baselineId;
    setLoading(true);
    setRefreshing(!initialLoad);

    if (baselineChanged) {
      if (includeDrift) {
        setDrift(null);
      }
      if (includeRemediation) {
        setRemediation(null);
      }
    }

    const nextErrors: LifecycleSignalErrors = {};
    const [costResult, policyResult, driftResult, remediationResult] = await Promise.allSettled([
      includeCosts ? getCosts("all") : Promise.resolve<PlatformComparison[] | null>(null),
      includePolicies ? getPolicies() : Promise.resolve<PolicyReport | null>(null),
      includeDrift && baselineId ? getDrift(baselineId) : Promise.resolve<DriftReport | null>(null),
      includeRemediation ? getRemediation(baselineId ?? undefined) : Promise.resolve<RecommendationReport | null>(null),
    ]);

    if (requestSequence !== requestSequenceRef.current) {
      return;
    }

    if (includeCosts) {
      if (costResult.status === "fulfilled") {
        setCosts(Array.isArray(costResult.value) ? costResult.value : []);
      } else {
        nextErrors.costs = toErrorMessage("cost comparisons", costResult.reason);
      }
    } else {
      setCosts([]);
    }

    if (includePolicies) {
      if (policyResult.status === "fulfilled") {
        setPolicies(policyResult.value);
      } else {
        nextErrors.policies = toErrorMessage("policy evaluation", policyResult.reason);
      }
    } else {
      setPolicies(null);
    }

    if (includeDrift) {
      if (!baselineId) {
        setDrift(null);
      } else if (driftResult.status === "fulfilled") {
        setDrift(driftResult.value);
      } else {
        nextErrors.drift = toErrorMessage("drift comparison", driftResult.reason);
      }
    } else {
      setDrift(null);
    }

    if (includeRemediation) {
      if (remediationResult.status === "fulfilled") {
        setRemediation(remediationResult.value);
      } else {
        nextErrors.remediation = toErrorMessage("remediation guidance", remediationResult.reason);
      }
    } else {
      setRemediation(null);
    }

    const settledErrors: LifecycleSignalErrors = {
      costs: includeCosts && costResult.status === "rejected" ? nextErrors.costs : undefined,
      policies: includePolicies && policyResult.status === "rejected" ? nextErrors.policies : undefined,
      drift: includeDrift && baselineId && driftResult.status === "rejected" ? nextErrors.drift : undefined,
      remediation: includeRemediation && remediationResult.status === "rejected" ? nextErrors.remediation : undefined,
    };
    setErrors((current) => ({ ...settledErrors, simulation: current.simulation }));
    hasLoadedRef.current = true;
    if (requestSequence === requestSequenceRef.current) {
      setLoading(false);
      setRefreshing(false);
    }
  }

  useEffect(() => {
    void refresh();
  }, [baselineId, refreshToken, includeCosts, includePolicies, includeDrift, includeRemediation]);

  async function simulate(targetPlatform: Platform) {
    try {
      setSimulationLoading(true);
      setSimulation(await runSimulation({ target_platform: targetPlatform, include_all: true }));
      setErrors((current) => ({ ...current, simulation: undefined }));
    } catch (reason) {
      setErrors((current) => ({ ...current, simulation: toErrorMessage("simulation", reason) }));
    } finally {
      setSimulationLoading(false);
    }
  }

  return {
    costs,
    policies,
    drift,
    remediation,
    simulation,
    loading,
    refreshing,
    simulationLoading,
    errors,
    refresh,
    simulate,
  };
}

function toErrorMessage(scope: string, reason: unknown): string {
  if (reason instanceof Error && reason.message.trim() !== "") {
    return `Unable to load ${scope}: ${reason.message}`;
  }
  return `Unable to load ${scope}.`;
}
