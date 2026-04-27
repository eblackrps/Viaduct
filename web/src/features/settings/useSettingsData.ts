import { useEffect, useRef, useState } from "react";
import {
	describeError,
	getAbout,
	getCurrentTenant,
	getReadiness,
	isAbortError,
	type ErrorDisplay,
} from "../../api";
import type {
	AboutResponse,
	CurrentTenant,
	ReadinessResponse,
} from "../../types";

interface SettingsDataErrors {
	about?: ErrorDisplay;
	currentTenant?: ErrorDisplay;
	readiness?: ErrorDisplay;
}

interface SettingsDataState {
	about: AboutResponse | null;
	currentTenant: CurrentTenant | null;
	readiness: ReadinessResponse | null;
	loading: boolean;
	errors: SettingsDataErrors;
}

export function useSettingsData(): SettingsDataState {
	const [about, setAbout] = useState<AboutResponse | null>(null);
	const [currentTenant, setCurrentTenant] = useState<CurrentTenant | null>(
		null,
	);
	const [readiness, setReadiness] = useState<ReadinessResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [errors, setErrors] = useState<SettingsDataErrors>({});
	const requestSequenceRef = useRef(0);
	const controllerRef = useRef<AbortController | null>(null);

	useEffect(() => {
		controllerRef.current?.abort();
		const controller = new AbortController();
		controllerRef.current = controller;
		const requestSequence = requestSequenceRef.current + 1;
		requestSequenceRef.current = requestSequence;

		void Promise.allSettled([
			getAbout({ signal: controller.signal }),
			getCurrentTenant({ signal: controller.signal }),
			getReadiness({ signal: controller.signal }),
		])
			.then(([aboutResult, currentTenantResult, readinessResult]) => {
				if (requestSequence !== requestSequenceRef.current) {
					return;
				}

				const nextErrors: SettingsDataErrors = {
					about:
						aboutResult.status === "rejected" &&
						!isAbortError(aboutResult.reason)
							? describeError(aboutResult.reason, {
									scope: "build metadata",
									fallback: "Unable to load build metadata.",
								})
							: undefined,
					currentTenant:
						currentTenantResult.status === "rejected" &&
						!isAbortError(currentTenantResult.reason)
							? describeError(currentTenantResult.reason, {
									scope: "tenant context",
									fallback: "Unable to load tenant context.",
								})
							: undefined,
					readiness:
						readinessResult.status === "rejected" &&
						!isAbortError(readinessResult.reason)
							? describeError(readinessResult.reason, {
									scope: "runtime readiness",
									fallback: "Unable to load runtime readiness.",
								})
							: undefined,
				};

				if (aboutResult.status === "fulfilled") {
					setAbout(aboutResult.value);
				}
				if (currentTenantResult.status === "fulfilled") {
					setCurrentTenant(currentTenantResult.value);
				}
				if (readinessResult.status === "fulfilled") {
					setReadiness(readinessResult.value);
				}

				setErrors(nextErrors);
			})
			.finally(() => {
				if (requestSequence === requestSequenceRef.current) {
					setLoading(false);
				}
			});
		return () => {
			controllerRef.current?.abort();
		};
	}, []);

	return {
		about,
		currentTenant,
		readiness,
		loading,
		errors,
	};
}
