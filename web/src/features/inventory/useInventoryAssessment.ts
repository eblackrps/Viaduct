import { useEffect, useMemo, useRef, useState } from "react";
import { getGraph, isAbortError } from "../../api";
import type {
	DependencyGraph,
	DiscoveryResult,
	SnapshotMeta,
} from "../../types";
import { useLifecycleSignals } from "../lifecycle/useLifecycleSignals";
import {
	buildInventoryAssessmentRows,
	summarizeAssessmentSignals,
	type InventoryAssessmentSourceState,
	type InventoryAssessmentRow,
} from "./inventoryModel";

interface InventoryAssessmentOptions {
	inventory: DiscoveryResult | null;
	latestSnapshot: SnapshotMeta | null;
	refreshToken: number;
}

interface InventoryAssessmentErrors {
	graph?: string;
	policies?: string;
	remediation?: string;
}

interface InventoryAssessmentState {
	rows: InventoryAssessmentRow[];
	loading: boolean;
	errors: InventoryAssessmentErrors;
	error: string | null;
	topSignals: Array<{ label: string; count: number }>;
	sources: InventoryAssessmentSourceState;
}

export function useInventoryAssessment({
	inventory,
	latestSnapshot,
	refreshToken,
}: InventoryAssessmentOptions): InventoryAssessmentState {
	const {
		policies,
		remediation,
		loading: lifecycleLoading,
		errors: lifecycleErrors,
	} = useLifecycleSignals({
		baselineId: latestSnapshot?.id ?? null,
		refreshToken,
		includePolicies: true,
		includeRemediation: true,
	});
	const [graph, setGraph] = useState<DependencyGraph | null>(null);
	const [graphLoading, setGraphLoading] = useState(Boolean(inventory));
	const [graphError, setGraphError] = useState<string | null>(null);
	const requestSequenceRef = useRef(0);
	const controllerRef = useRef<AbortController | null>(null);
	const hasInventory = Boolean(inventory);

	useEffect(() => {
		if (!hasInventory) {
			setGraph(null);
			setGraphLoading(false);
			setGraphError(null);
			return;
		}

		const requestSequence = requestSequenceRef.current + 1;
		requestSequenceRef.current = requestSequence;
		setGraphLoading(true);
		controllerRef.current?.abort();
		const controller = new AbortController();
		controllerRef.current = controller;

		getGraph({ signal: controller.signal })
			.then((result) => {
				if (requestSequence !== requestSequenceRef.current) {
					return;
				}
				setGraph(result);
				setGraphError(null);
			})
			.catch((reason: Error) => {
				if (
					requestSequence !== requestSequenceRef.current ||
					isAbortError(reason)
				) {
					return;
				}
				setGraphError(reason.message);
				setGraph(null);
			})
			.finally(() => {
				if (requestSequence === requestSequenceRef.current) {
					setGraphLoading(false);
				}
			});
		return () => {
			controllerRef.current?.abort();
		};
	}, [hasInventory, refreshToken]);

	const sources = useMemo<InventoryAssessmentSourceState>(
		() => ({
			graph: hasInventory && !graphLoading && !graphError,
			policies:
				!lifecycleLoading && !lifecycleErrors.policies && policies !== null,
			remediation:
				!lifecycleLoading &&
				!lifecycleErrors.remediation &&
				remediation !== null,
		}),
		[
			graphError,
			graphLoading,
			hasInventory,
			lifecycleErrors.policies,
			lifecycleErrors.remediation,
			lifecycleLoading,
			policies,
			remediation,
		],
	);

	const rows = useMemo(
		() =>
			buildInventoryAssessmentRows(
				inventory?.vms ?? [],
				graph,
				policies,
				remediation,
				sources,
				latestSnapshot?.discovered_at ?? inventory?.discovered_at,
			),
		[
			graph,
			inventory,
			latestSnapshot?.discovered_at,
			policies,
			remediation,
			sources,
		],
	);
	const topSignals = useMemo(() => summarizeAssessmentSignals(rows), [rows]);
	const errors: InventoryAssessmentErrors = {
		graph: graphError ?? undefined,
		policies: lifecycleErrors.policies,
		remediation: lifecycleErrors.remediation,
	};

	return {
		rows,
		loading: graphLoading || lifecycleLoading,
		errors,
		error:
			[errors.graph, errors.policies, errors.remediation]
				.filter(Boolean)
				.join(" ") || null,
		topSignals,
		sources,
	};
}
