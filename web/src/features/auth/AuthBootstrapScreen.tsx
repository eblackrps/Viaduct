import { ShieldCheck } from "lucide-react";
import { useState, type FormEvent, type ReactElement } from "react";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import {
  getDashboardAuthPersistence,
  getDashboardAuthSession,
  type DashboardAuthPersistence,
} from "../../runtimeAuth";
import type { AuthBootstrapState } from "./useAuthBootstrap";

interface AuthBootstrapScreenProps {
  auth: AuthBootstrapState;
}

export function AuthBootstrapScreen({ auth }: AuthBootstrapScreenProps) {
  const [mode, setMode] = useState<"service-account" | "tenant">("service-account");
  const [apiKey, setAPIKey] = useState("");
  const [remember, setRemember] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const storedSession = getDashboardAuthSession();
  const persistence = getDashboardAuthPersistence();
  const runtimeKeyPresent = storedSession.source === "runtime" && storedSession.mode !== "none";
  const errorActions = [
    <button key="retry" type="button" onClick={() => void auth.refresh()} className="operator-button-danger">
      Retry validation
    </button>,
    runtimeKeyPresent ? (
      <button key="forget" type="button" onClick={() => auth.signOut()} className="operator-button-secondary">
        Forget browser key
      </button>
    ) : null,
  ].filter((action): action is ReactElement => action !== null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    try {
      await auth.connect(mode, apiKey, remember);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-transparent px-4 py-4 md:px-6 md:py-6">
      <div className="mx-auto max-w-6xl space-y-6">
        <div className="panel overflow-hidden p-6 lg:p-7">
          <PageHeader
            eyebrow="Operator Bootstrap"
            title="Connect the Viaduct dashboard"
            description="Use a scoped service-account key for day-to-day operator work. Use a tenant key only when you are bootstrapping tenant access or handling a break-glass administrative task."
            badges={[
              { label: auth.about ? `${auth.about.name} ${auth.about.version}` : "Build metadata pending", tone: "neutral" },
              { label: "Session storage is the default", tone: "info" },
              { label: "Runtime authentication", tone: "accent" },
            ]}
          />

          {auth.status === "checking" && (
            <div className="mt-6">
              <LoadingState
                title="Validating dashboard credentials"
                message="The dashboard is confirming the provided service-account or tenant key against the current Viaduct API."
              />
            </div>
          )}

          {auth.status === "error" && auth.error && (
            <div className="mt-6">
              <ErrorState
                title="Authentication failed"
                message={auth.error.message}
                technicalDetails={auth.error.technicalDetails}
                actions={errorActions}
              />
            </div>
          )}

          <section className="mt-6 grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
            <SectionCard
              eyebrow="Authentication"
              title="Runtime credential bootstrap"
              description="The dashboard accepts runtime credentials so operators do not need to rebuild the frontend to rotate or replace keys."
            >
              <form className="space-y-4" onSubmit={handleSubmit}>
                <div className="operator-toggle">
                  <button
                    type="button"
                    onClick={() => setMode("service-account")}
                    className={`operator-toggle-button ${mode === "service-account" ? "operator-toggle-button-active" : ""}`}
                  >
                    Service account
                  </button>
                  <button
                    type="button"
                    onClick={() => setMode("tenant")}
                    className={`operator-toggle-button ${mode === "tenant" ? "operator-toggle-button-active" : ""}`}
                  >
                    Tenant key
                  </button>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="md:col-span-2">
                    <label className="block">
                      <span className="operator-kicker">{mode === "service-account" ? "Service-account key" : "Tenant key"}</span>
                      <input
                        type="password"
                        value={apiKey}
                        onChange={(event) => setAPIKey(event.target.value)}
                        className="operator-input mt-2"
                        placeholder={mode === "service-account" ? "Paste the operator service-account key" : "Paste the tenant bootstrap key"}
                      />
                    </label>
                    <p className="mt-2 text-sm text-slate-600">
                      {mode === "service-account"
                        ? "Preferred for the guided workspace flow, background jobs, and exported reports."
                        : "Use only when you need tenant bootstrap access or administrative recovery."}
                    </p>
                  </div>

                  <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50/90 px-4 py-4 text-sm text-slate-700 md:col-span-2">
                    <input
                      type="checkbox"
                      checked={remember}
                      onChange={(event) => setRemember(event.target.checked)}
                      className="mt-0.5 h-4 w-4 rounded border-slate-300 text-sky-600 focus:ring-sky-500"
                    />
                    <span>
                      <span className="block font-semibold text-ink">Remember this browser</span>
                      <span className="mt-1 block text-slate-600">
                        Keep the key in local browser storage for trusted operator workstations. Leave this off for shared devices, temporary sessions, or validation labs.
                      </span>
                    </span>
                  </label>
                </div>

                <div className="flex flex-wrap gap-2">
                  <button type="submit" disabled={submitting || apiKey.trim() === ""} className="operator-button">
                    {submitting ? "Connecting..." : "Connect dashboard"}
                  </button>
                  {runtimeKeyPresent ? (
                    <button type="button" onClick={() => auth.signOut()} className="operator-button-secondary">
                      Forget browser key
                    </button>
                  ) : null}
                </div>
              </form>
            </SectionCard>

            <div className="space-y-5">
              <SectionCard
                eyebrow="Recommended flow"
                title="What operators do next"
                description="The dashboard is optimized for the workspace-first flow, not a disconnected collection of demo pages."
              >
                <div className="space-y-3">
                  {[
                    {
                      step: "01",
                      title: "Create a workspace",
                      body: "Capture the source connection, credential reference, and target assumptions in one persisted record.",
                    },
                    {
                      step: "02",
                      title: "Run discovery and inspect",
                      body: "Persist snapshots, review workloads, and inspect dependency context before planning.",
                    },
                    {
                      step: "03",
                      title: "Simulate, plan, and export",
                      body: "Keep readiness results, saved plans, notes, and exported reports attached to the same workspace.",
                    },
                  ].map((item) => (
                    <div key={item.step} className="metric-surface flex items-start gap-3">
                      <span className="inline-flex h-10 w-10 items-center justify-center rounded-2xl bg-ink text-sm font-semibold text-white">
                        {item.step}
                      </span>
                      <div>
                        <p className="text-sm font-semibold text-ink">{item.title}</p>
                        <p className="mt-1 text-sm leading-6 text-slate-600">{item.body}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </SectionCard>

              <SectionCard
                eyebrow="Credential handling"
                title="Storage and recovery"
                description="Keep credential behavior explicit so operators know when a browser session can be trusted and when it should be cleared."
              >
                <div className="grid gap-3">
                  <StorageRow label="Current source" value={describeCurrentSource(storedSession.mode, storedSession.source)} />
                  <StorageRow label="Persistence" value={describePersistence(persistence)} />
                  <StorageRow label="Recommended default" value="Session-only browser storage" />
                </div>
              </SectionCard>

              <SectionCard
                eyebrow="Current API context"
                title="Operator visibility"
                description="Build metadata and tenant identity are shown here as soon as the API responds."
              >
                <div className="flex flex-wrap gap-2">
                  {auth.currentTenant ? (
                    <>
                      <StatusBadge tone="success">{auth.currentTenant.name}</StatusBadge>
                      <StatusBadge tone="info">{auth.currentTenant.role}</StatusBadge>
                      <StatusBadge tone="neutral">{auth.currentTenant.auth_method}</StatusBadge>
                    </>
                  ) : (
                    <StatusBadge tone="neutral">Tenant context will appear after validation</StatusBadge>
                  )}
                </div>
                <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50/90 px-4 py-4 text-sm text-slate-600">
                  <div className="flex items-start gap-3">
                    <div className="rounded-2xl bg-white p-2 text-slate-600">
                      <ShieldCheck className="h-4 w-4" />
                    </div>
                    <div>
                      <p className="font-semibold text-ink">Tenant-scoped operator access</p>
                      <p className="mt-1 leading-6">
                        The dashboard uses the same tenant-scoped API and persisted backend model as the CLI and packaged operator surfaces.
                      </p>
                    </div>
                  </div>
                </div>
              </SectionCard>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function StorageRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-surface">
      <p className="operator-kicker">{label}</p>
      <p className="mt-2 text-sm font-semibold text-ink">{value}</p>
    </div>
  );
}

function describeCurrentSource(mode: "tenant" | "service-account" | "none", source: "runtime" | "environment" | "none"): string {
  if (mode === "none" || source === "none") {
    return "No credential configured";
  }
  if (mode === "service-account") {
    return source === "runtime" ? "Service-account key entered at runtime" : "Service-account key provided by environment";
  }
  return source === "runtime" ? "Tenant key entered at runtime" : "Tenant key provided by environment";
}

function describePersistence(persistence: DashboardAuthPersistence): string {
  switch (persistence) {
    case "session":
      return "Stored only for the current browser session";
    case "local":
      return "Stored in local browser storage until removed";
    case "environment":
      return "Provided by environment configuration";
    default:
      return "Nothing stored yet";
  }
}
