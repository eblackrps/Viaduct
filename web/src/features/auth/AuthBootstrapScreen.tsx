import { useState, type FormEvent } from "react";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { AuthBootstrapState } from "./useAuthBootstrap";

interface AuthBootstrapScreenProps {
  auth: AuthBootstrapState;
}

export function AuthBootstrapScreen({ auth }: AuthBootstrapScreenProps) {
  const [mode, setMode] = useState<"service-account" | "tenant">("service-account");
  const [apiKey, setAPIKey] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    try {
      await auth.connect(mode, apiKey);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-transparent px-4 py-6 md:px-6">
      <div className="mx-auto max-w-5xl space-y-6 rounded-[2rem] border border-white/60 bg-white/35 p-6 backdrop-blur-sm">
        <PageHeader
          eyebrow="Pilot Bootstrap"
          title="Connect the operator dashboard"
          description="Use a scoped service-account key for the normal pilot flow, or a tenant key when you are bootstrapping the tenant or doing break-glass administration."
          badges={[
            { label: auth.about ? `${auth.about.name} ${auth.about.version}` : "About pending", tone: "neutral" },
            { label: "Service-account first", tone: "info" },
          ]}
        />

        {auth.status === "checking" && (
          <LoadingState
            title="Validating dashboard credentials"
            message="The dashboard is confirming the provided tenant or service-account key against the current Viaduct API."
          />
        )}

        {auth.status === "error" && auth.error && (
          <ErrorState
            title="Authentication failed"
            message={auth.error.message}
            technicalDetails={auth.error.technicalDetails}
            actions={[
              <button
                key="retry"
                type="button"
                onClick={() => void auth.refresh()}
                className="rounded-full border border-rose-200 bg-white px-4 py-2 text-sm font-semibold text-rose-700 transition hover:bg-rose-50"
              >
                Retry validation
              </button>,
            ]}
          />
        )}

        <section className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
          <SectionCard
            title="Runtime authentication"
            description="This dashboard now accepts runtime credentials instead of depending on build-time-only environment variables."
          >
            <form className="space-y-4" onSubmit={handleSubmit}>
              <div className="inline-flex rounded-full border border-slate-200 bg-white p-1 text-sm">
                <button
                  type="button"
                  onClick={() => setMode("service-account")}
                  className={`rounded-full px-3 py-1.5 font-semibold transition ${mode === "service-account" ? "bg-ink text-white" : "text-slate-600 hover:bg-slate-50"}`}
                >
                  Service account
                </button>
                <button
                  type="button"
                  onClick={() => setMode("tenant")}
                  className={`rounded-full px-3 py-1.5 font-semibold transition ${mode === "tenant" ? "bg-ink text-white" : "text-slate-600 hover:bg-slate-50"}`}
                >
                  Tenant key
                </button>
              </div>

              <label className="block">
                <span className="text-sm font-semibold text-ink">{mode === "service-account" ? "Service-account key" : "Tenant key"}</span>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(event) => setAPIKey(event.target.value)}
                  className="mt-2 w-full rounded-2xl border border-slate-200 bg-white px-4 py-3 text-sm text-ink outline-none transition focus:border-sky-400"
                  placeholder={mode === "service-account" ? "Paste the pilot service-account key" : "Paste the tenant bootstrap key"}
                />
              </label>

              <button
                type="submit"
                disabled={submitting || apiKey.trim() === ""}
                className="rounded-full bg-ink px-4 py-2 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {submitting ? "Connecting..." : "Connect dashboard"}
              </button>
            </form>
          </SectionCard>

          <SectionCard title="Expected pilot flow" description="The v1.6.0 dashboard is optimized for the assessment-to-pilot wedge.">
            <div className="space-y-3">
              {[
                "Create a pilot workspace with source and target assumptions.",
                "Run discovery and persist the current snapshot baseline.",
                "Inspect workloads and dependency graph output.",
                "Simulate readiness, save a migration plan, and export the report.",
              ].map((item) => (
                <div key={item} className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-700">
                  {item}
                </div>
              ))}
            </div>
            {auth.currentTenant && (
              <div className="mt-4 flex flex-wrap gap-2">
                <StatusBadge tone="success">{auth.currentTenant.name}</StatusBadge>
                <StatusBadge tone="info">{auth.currentTenant.role}</StatusBadge>
                <StatusBadge tone="neutral">{auth.currentTenant.auth_method}</StatusBadge>
              </div>
            )}
          </SectionCard>
        </section>
      </div>
    </div>
  );
}
