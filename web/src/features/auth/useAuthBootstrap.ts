import { useCallback, useEffect, useRef, useState } from "react";
import {
	createDashboardAuthSession,
	deleteDashboardAuthSession,
	describeError,
	getAbout,
	getCurrentTenant,
	isAPIError,
	requestManager,
	type ErrorDisplay,
} from "../../api";
import {
	clearDashboardAuthSession,
	getDashboardAuthSession,
	setDashboardAuthSession,
	type DashboardAuthMode,
} from "../../runtimeAuth";
import type { AboutResponse, CurrentTenant } from "../../types";

export interface AuthBootstrapState {
	status: "checking" | "authenticated" | "unauthenticated" | "error";
	about: AboutResponse | null;
	currentTenant: CurrentTenant | null;
	localOperatorAvailable: boolean;
	error: ErrorDisplay | null;
	refresh: () => Promise<void>;
	connect: (
		mode: Exclude<DashboardAuthMode, "none">,
		apiKey: string,
		remember?: boolean,
	) => Promise<void>;
	connectLocal: (remember?: boolean) => Promise<void>;
	signOut: () => void;
}

export function useAuthBootstrap(): AuthBootstrapState {
	const [status, setStatus] =
		useState<AuthBootstrapState["status"]>("checking");
	const [about, setAbout] = useState<AboutResponse | null>(null);
	const [currentTenant, setCurrentTenant] = useState<CurrentTenant | null>(
		null,
	);
	const [error, setError] = useState<ErrorDisplay | null>(null);
	const suppressAutoLocalSession = useRef(false);

	const startLocalSession = useCallback(async (remember = false) => {
		requestManager.cancelAll();
		const session = await createDashboardAuthSession("local", "", remember);
		requestManager.cancelAll();
		setDashboardAuthSession("local", {
			remember,
			sessionID: session.session_id,
		});
	}, []);

	const refresh = useCallback(async () => {
		setStatus("checking");
		const [aboutResult, tenantResult] = await Promise.allSettled([
			getAbout({ dedupe: false }),
			getCurrentTenant({ dedupe: false }),
		]);
		const nextAbout =
			aboutResult.status === "fulfilled" ? aboutResult.value : null;
		if (aboutResult.status === "fulfilled") {
			setAbout(aboutResult.value);
		}

		if (tenantResult.status === "fulfilled") {
			setCurrentTenant(tenantResult.value);
			setError(null);
			setStatus("authenticated");
			return;
		}

		if (isExpectedUnauthenticated(tenantResult.reason)) {
			if (getDashboardAuthSession().source === "runtime") {
				clearDashboardAuthSession();
			}
			if (
				nextAbout?.local_operator_session_enabled &&
				!suppressAutoLocalSession.current
			) {
				try {
					await startLocalSession(false);
					setCurrentTenant(await getCurrentTenant({ dedupe: false }));
					setError(null);
					setStatus("authenticated");
					return;
				} catch (reason) {
					clearDashboardAuthSession();
					setCurrentTenant(null);
					setError(
						describeError(reason, {
							scope: "your local session",
							fallback: "Unable to start the local dashboard session.",
						}),
					);
					setStatus("error");
					return;
				}
			}
			setCurrentTenant(null);
			setError(null);
			setStatus("unauthenticated");
			return;
		}

		setCurrentTenant(null);
		setError(
			describeError(tenantResult.reason, {
				scope: "your session",
				fallback: "Unable to check your session.",
			}),
		);
		setStatus("error");
	}, [startLocalSession]);

	async function connect(
		mode: Exclude<DashboardAuthMode, "none">,
		apiKey: string,
		remember = false,
	) {
		requestManager.cancelAll();
		try {
			const session = await createDashboardAuthSession(mode, apiKey, remember);
			requestManager.cancelAll();
			setDashboardAuthSession(mode, {
				remember,
				sessionID: session.session_id,
			});
			await refresh();
		} catch (reason) {
			requestManager.cancelAll();
			setCurrentTenant(null);
			setError(
				describeError(reason, {
					scope: "your session",
					fallback: "Unable to start your session.",
				}),
			);
			setStatus("error");
			throw reason;
		}
	}

	async function connectLocal(remember = false) {
		suppressAutoLocalSession.current = false;
		try {
			await startLocalSession(remember);
			await refresh();
		} catch (reason) {
			requestManager.cancelAll();
			setCurrentTenant(null);
			setError(
				describeError(reason, {
					scope: "your session",
					fallback: "Unable to start your session.",
				}),
			);
			setStatus("error");
			throw reason;
		}
	}

	function signOut() {
		suppressAutoLocalSession.current = true;
		requestManager.cancelAll();
		clearDashboardAuthSession();
		void deleteDashboardAuthSession()
			.catch(() => undefined)
			.finally(() => {
				void refresh();
			});
	}

	useEffect(() => {
		void refresh();
	}, [refresh]);

	return {
		status,
		about,
		currentTenant,
		localOperatorAvailable: Boolean(about?.local_operator_session_enabled),
		error,
		refresh,
		connect,
		connectLocal,
		signOut,
	};
}

function isExpectedUnauthenticated(reason: unknown): boolean {
	if (!isAPIError(reason)) {
		return false;
	}
	if (reason.status === 401) {
		return true;
	}
	return false;
}
