import { useEffect, useState } from "react";
import { describeError, getAbout, getCurrentTenant, type ErrorDisplay } from "../../api";
import {
  clearDashboardAuthSession,
  getDashboardAuthSession,
  hasDashboardAuthConfigured,
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
  connect: (mode: Exclude<DashboardAuthMode, "none">, apiKey: string) => Promise<void>;
  signOut: () => void;
}

export function useAuthBootstrap(): AuthBootstrapState {
  const [status, setStatus] = useState<AuthBootstrapState["status"]>("checking");
  const [about, setAbout] = useState<AboutResponse | null>(null);
  const [currentTenant, setCurrentTenant] = useState<CurrentTenant | null>(null);
  const [error, setError] = useState<ErrorDisplay | null>(null);

  async function refresh() {
    setAbout((current) => current);
    if (!hasDashboardAuthConfigured()) {
      setCurrentTenant(null);
      setError(null);
      setStatus("unauthenticated");
      return;
    }

    setStatus("checking");
    const [aboutResult, tenantResult] = await Promise.allSettled([getAbout(), getCurrentTenant()]);
    if (aboutResult.status === "fulfilled") {
      setAbout(aboutResult.value);
    }

    if (tenantResult.status === "fulfilled") {
      setCurrentTenant(tenantResult.value);
      setError(null);
      setStatus("authenticated");
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

  async function connect(mode: Exclude<DashboardAuthMode, "none">, apiKey: string) {
    setDashboardAuthSession(mode, apiKey);
    await refresh();
  }

  function signOut() {
    clearDashboardAuthSession();
    setCurrentTenant(null);
    setError(null);
    setStatus("unauthenticated");
  }

  useEffect(() => {
    const session = getDashboardAuthSession();
    if (session.mode === "none") {
      setStatus("unauthenticated");
      void getAbout().then(setAbout).catch(() => null);
      return;
    }
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
