import { useEffect, useState } from "react";
import {
	createDashboardAuthSession,
	deleteDashboardAuthSession,
	describeError,
	getAbout,
	getCurrentTenant,
	isAPIError,
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
	error: ErrorDisplay | null;
	refresh: () => Promise<void>;
	connect: (
		mode: Exclude<DashboardAuthMode, "none">,
		apiKey: string,
		remember?: boolean,
	) => Promise<void>;
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

	async function refresh() {
		setAbout((current) => current);
		setStatus("checking");
		const [aboutResult, tenantResult] = await Promise.allSettled([
			getAbout(),
			getCurrentTenant(),
		]);
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
			setCurrentTenant(null);
			setError(null);
			setStatus("unauthenticated");
			return;
		}

		setCurrentTenant(null);
		setError(
			describeError(tenantResult.reason, {
				scope: "dashboard authentication",
				fallback: "Unable to validate the configured dashboard credentials.",
			}),
		);
		setStatus("error");
	}

	async function connect(
		mode: Exclude<DashboardAuthMode, "none">,
		apiKey: string,
		remember = false,
	) {
		const session = await createDashboardAuthSession(mode, apiKey, remember);
		setDashboardAuthSession(mode, {
			remember,
			sessionID: session.session_id,
		});
		await refresh();
	}

	function signOut() {
		clearDashboardAuthSession();
		void deleteDashboardAuthSession()
			.catch(() => undefined)
			.finally(() => {
				void refresh();
			});
	}

	useEffect(() => {
		void refresh();
	}, []);

	return {
		status,
		about,
		currentTenant,
		error,
		refresh,
		connect,
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
	return (
		reason.code === "missing_credentials" ||
		reason.code === "invalid_credentials"
	);
}
