export type DashboardAuthMode = "tenant" | "service-account" | "none";
export type DashboardAuthSource = "runtime" | "environment" | "none";
export type DashboardAuthPersistence = "session" | "local" | "environment" | "none";

export interface DashboardAuthSession {
  mode: DashboardAuthMode;
  apiKey: string;
  source: DashboardAuthSource;
}

const sessionStorageKey = "viaduct.dashboardAuth";
const localStorageKey = "viaduct.dashboardAuth.remembered";

const environmentTenantKey = import.meta.env.VITE_VIADUCT_API_KEY?.trim() ?? "";
const environmentServiceAccountKey = import.meta.env.VITE_VIADUCT_SERVICE_ACCOUNT_KEY?.trim() ?? "";

export function getDashboardAuthSession(): DashboardAuthSession {
  const runtime = readRuntimeAuth();
  if (runtime.mode !== "none" && runtime.apiKey !== "") {
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
  apiKey: string,
  options?: { remember?: boolean },
) {
  const nextSession: DashboardAuthSession = {
    mode,
    apiKey: apiKey.trim(),
    source: "runtime",
  };
  if (nextSession.apiKey === "") {
    clearDashboardAuthSession();
    return;
  }
  if (typeof window === "undefined") {
    return;
  }
  window.sessionStorage.removeItem(sessionStorageKey);
  window.localStorage.removeItem(localStorageKey);
  if (options?.remember) {
    window.localStorage.setItem(localStorageKey, JSON.stringify(nextSession));
    return;
  }
  window.sessionStorage.setItem(sessionStorageKey, JSON.stringify(nextSession));
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
    return runtime.session;
  }
  return { mode: "none", apiKey: "", source: "none" };
}

function readStoredRuntimeAuth():
  | {
      session: DashboardAuthSession;
      persistence: Exclude<DashboardAuthPersistence, "environment" | "none">;
    }
  | null {
  if (typeof window === "undefined") {
    return null;
  }

  const sources: Array<{
    raw: string | null;
    persistence: Exclude<DashboardAuthPersistence, "environment" | "none">;
  }> = [
    { raw: window.sessionStorage.getItem(sessionStorageKey), persistence: "session" },
    { raw: window.localStorage.getItem(localStorageKey), persistence: "local" },
  ];

  for (const candidate of sources) {
    const parsed = parseRuntimeSession(candidate.raw);
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

function parseRuntimeSession(raw: string | null): DashboardAuthSession | null {
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as Partial<DashboardAuthSession>;
    const mode = parsed.mode === "tenant" || parsed.mode === "service-account" ? parsed.mode : "none";
    const apiKey = typeof parsed.apiKey === "string" ? parsed.apiKey.trim() : "";
    if (mode === "none" || apiKey === "") {
      return null;
    }
    return {
      mode,
      apiKey,
      source: "runtime",
    };
  } catch {
    return null;
  }
}
