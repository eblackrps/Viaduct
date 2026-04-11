export type DashboardAuthMode = "tenant" | "service-account" | "none";
export type DashboardAuthSource = "runtime" | "environment" | "none";

export interface DashboardAuthSession {
  mode: DashboardAuthMode;
  apiKey: string;
  source: DashboardAuthSource;
}

const storageKey = "viaduct.dashboardAuth";

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

export function setDashboardAuthSession(mode: Exclude<DashboardAuthMode, "none">, apiKey: string) {
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
  window.localStorage.setItem(storageKey, JSON.stringify(nextSession));
}

export function clearDashboardAuthSession() {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.removeItem(storageKey);
}

export function getDashboardAuthMode(): DashboardAuthMode {
  return getDashboardAuthSession().mode;
}

export function hasDashboardAuthConfigured(): boolean {
  return getDashboardAuthMode() !== "none";
}

function readRuntimeAuth(): DashboardAuthSession {
  if (typeof window === "undefined") {
    return { mode: "none", apiKey: "", source: "none" };
  }

  const raw = window.localStorage.getItem(storageKey);
  if (!raw) {
    return { mode: "none", apiKey: "", source: "none" };
  }

  try {
    const parsed = JSON.parse(raw) as Partial<DashboardAuthSession>;
    const mode = parsed.mode === "tenant" || parsed.mode === "service-account" ? parsed.mode : "none";
    const apiKey = typeof parsed.apiKey === "string" ? parsed.apiKey.trim() : "";
    if (mode === "none" || apiKey === "") {
      return { mode: "none", apiKey: "", source: "none" };
    }
    return {
      mode,
      apiKey,
      source: "runtime",
    };
  } catch {
    return { mode: "none", apiKey: "", source: "none" };
  }
}
