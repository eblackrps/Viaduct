/**
 * Runtime dashboard auth persists only a short-lived session identifier in Web Storage.
 * Environment-provided keys remain available for explicit pre-seeded development flows,
 * while browser-entered runtime keys are exchanged for a server-backed session and are
 * not retained client-side after session creation completes.
 */
export type DashboardAuthMode = "local" | "tenant" | "service-account" | "none";
type DashboardAuthSource = "runtime" | "environment" | "none";
export type DashboardAuthPersistence =
	| "session"
	| "local"
	| "environment"
	| "none";

interface DashboardAuthSession {
	mode: DashboardAuthMode;
	apiKey: string;
	source: DashboardAuthSource;
	sessionID?: string;
}

interface StoredRuntimeSession {
	mode: Exclude<DashboardAuthMode, "none">;
	session_id: string;
}

const sessionStorageKey = "viaduct.dashboardAuth";
const localStorageKey = "viaduct.dashboardAuth.remembered";

const environmentTenantKey = import.meta.env.VITE_VIADUCT_API_KEY?.trim() ?? "";
const environmentServiceAccountKey =
	import.meta.env.VITE_VIADUCT_SERVICE_ACCOUNT_KEY?.trim() ?? "";

export function getDashboardAuthSession(): DashboardAuthSession {
	const runtime = readRuntimeAuth();
	if (runtime.mode !== "none") {
		return runtime;
	}
	if (environmentServiceAccountKey !== "") {
		return {
			mode: "service-account",
			apiKey: environmentServiceAccountKey,
			source: "environment",
		};
	}
	if (environmentTenantKey !== "") {
		return {
			mode: "tenant",
			apiKey: environmentTenantKey,
			source: "environment",
		};
	}
	return {
		mode: "none",
		apiKey: "",
		source: "none",
	};
}

export function getDashboardAuthPersistence(): DashboardAuthPersistence {
	const runtime = readStoredRuntimeAuth();
	if (runtime?.persistence) {
		return runtime.persistence;
	}
	if (environmentServiceAccountKey !== "" || environmentTenantKey !== "") {
		return "environment";
	}
	return "none";
}

export function setDashboardAuthSession(
	mode: Exclude<DashboardAuthMode, "none">,
	options?: { remember?: boolean; sessionID?: string },
) {
	const sessionID = options?.sessionID?.trim() ?? "";
	if (sessionID === "") {
		clearDashboardAuthSession();
		return;
	}
	if (typeof window === "undefined") {
		return;
	}
	const storedSession: StoredRuntimeSession = {
		mode,
		session_id: sessionID,
	};
	window.sessionStorage.removeItem(sessionStorageKey);
	window.localStorage.removeItem(localStorageKey);
	if (options?.remember) {
		window.localStorage.setItem(localStorageKey, JSON.stringify(storedSession));
		return;
	}
	window.sessionStorage.setItem(
		sessionStorageKey,
		JSON.stringify(storedSession),
	);
}

export function clearDashboardAuthSession() {
	if (typeof window === "undefined") {
		return;
	}
	window.sessionStorage.removeItem(sessionStorageKey);
	window.localStorage.removeItem(localStorageKey);
}

export function getDashboardAuthMode(): DashboardAuthMode {
	return getDashboardAuthSession().mode;
}

export function hasDashboardAuthConfigured(): boolean {
	return getDashboardAuthMode() !== "none";
}

function readRuntimeAuth(): DashboardAuthSession {
	const runtime = readStoredRuntimeAuth();
	if (runtime) {
		return {
			...runtime.session,
			apiKey: "",
			source: "runtime",
		};
	}
	return { mode: "none", apiKey: "", source: "none" };
}

function readStoredRuntimeAuth(): {
	session: Omit<DashboardAuthSession, "apiKey" | "source"> &
		Pick<DashboardAuthSession, "mode" | "sessionID">;
	persistence: Exclude<DashboardAuthPersistence, "environment" | "none">;
} | null {
	if (typeof window === "undefined") {
		return null;
	}

	const sources: Array<{
		raw: string | null;
		persistence: Exclude<DashboardAuthPersistence, "environment" | "none">;
		storageKey: string;
	}> = [
		{
			raw: window.sessionStorage.getItem(sessionStorageKey),
			persistence: "session",
			storageKey: sessionStorageKey,
		},
		{
			raw: window.localStorage.getItem(localStorageKey),
			persistence: "local",
			storageKey: localStorageKey,
		},
	];

	for (const candidate of sources) {
		const parsed = parseRuntimeSession(candidate.raw, candidate.storageKey);
		if (!parsed) {
			continue;
		}
		return {
			session: parsed,
			persistence: candidate.persistence,
		};
	}

	return null;
}

function parseRuntimeSession(
	raw: string | null,
	storageKey: string,
): DashboardAuthSession | null {
	if (!raw) {
		return null;
	}

	try {
		const parsed = JSON.parse(raw) as Partial<StoredRuntimeSession>;
		const mode =
			parsed.mode === "tenant" ||
			parsed.mode === "service-account" ||
			parsed.mode === "local"
				? parsed.mode
				: "none";
		const sessionID =
			typeof parsed.session_id === "string" ? parsed.session_id.trim() : "";
		if (mode === "none" || sessionID === "") {
			console.warn(
				"invalid dashboard auth session payload, clearing stored data",
				{
					storageKey,
				},
			);
			clearStoredRuntimeSession(storageKey);
			return null;
		}
		return {
			mode,
			source: "runtime",
			apiKey: "",
			sessionID,
		};
	} catch (error) {
		console.warn(
			"failed to parse dashboard auth session, clearing stored data",
			{
				error,
				storageKey,
			},
		);
		clearStoredRuntimeSession(storageKey);
		return null;
	}
}

function clearStoredRuntimeSession(storageKey: string) {
	if (typeof window === "undefined") {
		return;
	}
	if (storageKey === sessionStorageKey) {
		window.sessionStorage.removeItem(sessionStorageKey);
		return;
	}
	if (storageKey === localStorageKey) {
		window.localStorage.removeItem(localStorageKey);
	}
}
