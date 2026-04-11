export type DashboardAuthMode = "tenant" | "service-account" | "none";
export type DashboardAuthSource = "runtime" | "environment" | "none";

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
  if (typeof window === "undefined") {
    return { mode: "none", apiKey: "", source: "none" };
  }

  for (const raw of [window.sessionStorage.getItem(sessionStorageKey), window.localStorage.getItem(localStorageKey)]) {
    if (!raw) {
      continue;
    }
    try {
      const parsed = JSON.parse(raw) as Partial<DashboardAuthSession>;
      const mode = parsed.mode === "tenant" || parsed.mode === "service-account" ? parsed.mode : "none";
      const apiKey = typeof parsed.apiKey === "string" ? parsed.apiKey.trim() : "";
      if (mode === "none" || apiKey === "") {
        continue;
      }
      return {
        mode,
        apiKey,
        source: "runtime",
      };
    } catch {
      continue;
    }
  }
  return { mode: "none", apiKey: "", source: "none" };
}
