import { hasApiKeyConfigured, type ErrorDisplay } from "../../api";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { TenantSummary } from "../../types";
import { useSettingsData } from "./useSettingsData";

interface SettingsPageProps {
  summary: TenantSummary | null;
  authSourceLabel: string;
  authPersistenceLabel: string;
  onForgetRuntimeKey?: (() => void) | undefined;
}

export function SettingsPage({
  summary,
  authSourceLabel,
  authPersistenceLabel,
  onForgetRuntimeKey,
}: SettingsPageProps) {
  const authConfigured = hasApiKeyConfigured();
  const { about, currentTenant, loading, errors } = useSettingsData();
  const supportedPlatforms = about?.supported_platforms ?? Object.keys(summary?.platform_counts ?? {});
  const platformCounts = summary?.platform_counts ?? {};
  const settingsError = [errors.about?.message, errors.currentTenant?.message].filter(Boolean).join(" ");
  const settingsErrorDetails = Array.from(
    new Set([...(errors.about?.technicalDetails ?? []), ...(errors.currentTenant?.technicalDetails ?? [])]),
  );
  const showEmpty = !loading && !about && !currentTenant;

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Settings"
        title="Operator settings"
        description="Read-only runtime context for the current tenant, runtime credential handling, and dashboard-side operator assumptions."
        badges={[
          {
            label: authSourceLabel,
            tone: authConfigured ? "success" : "warning",
          },
          { label: authPersistenceLabel, tone: "neutral" },
          { label: currentTenant?.tenant_id ? `Tenant ${currentTenant.tenant_id}` : summary?.tenant_id ? `Tenant ${summary.tenant_id}` : "No tenant summary", tone: "neutral" },
        ]}
      />

      {loading && !about && !currentTenant && (
        <LoadingState
          title="Loading workspace settings"
          message="Retrieving operator build metadata and the effective tenant context from the Viaduct API."
        />
      )}

      {showEmpty &&
        (settingsError ? (
          <ErrorState
            title="Workspace settings unavailable"
            message={settingsError}
            technicalDetails={settingsErrorDetails}
          />
        ) : (
          <EmptyState
            title="No runtime context available"
            message="The dashboard could not load either build metadata or the current tenant context from the operator API."
          />
        ))}

      <section className="grid gap-5 xl:grid-cols-2">
        <SectionCard title="Operator connection" description="Current dashboard runtime assumptions for API access.">
          {errors.about ? (
            <InlineError error={errors.about} />
          ) : (
            <div className="space-y-4">
              <SettingRow label="API base" value="/api/v1 via Vite proxy in development or packaged backend in release builds" />
              <SettingRow label="Authentication" value={authSourceLabel} />
              <SettingRow label="Credential persistence" value={authPersistenceLabel} />
              <SettingRow label="Version" value={about ? `${about.name} ${about.version} (${about.api_version})` : "Unavailable"} />
              <SettingRow label="Build commit" value={about?.commit || "Unavailable"} />
              <SettingRow label="Plugin protocol" value={about?.plugin_protocol || "Unavailable"} />
              <SettingRow label="Store backend" value={about ? `${about.store_backend}${about.persistent_store ? " (persistent)" : " (ephemeral)"}` : "Unavailable"} />
            </div>
          )}
        </SectionCard>

        <SectionCard title="Tenant context" description="Current summary-level state visible to the dashboard shell.">
          {errors.currentTenant ? (
            <InlineError error={errors.currentTenant} />
          ) : (
            <div className="grid gap-3 md:grid-cols-2">
              <ContextStat label="Tenant name" value={currentTenant?.name ?? summary?.tenant_id ?? "Unavailable"} />
              <ContextStat label="Role" value={currentTenant?.role ?? "Unavailable"} />
              <ContextStat label="Auth method" value={currentTenant?.auth_method ?? "Unavailable"} />
              <ContextStat label="Service account" value={currentTenant?.service_account_name ?? "Tenant credential"} />
              <ContextStat label="Service accounts" value={String(currentTenant?.service_account_count ?? 0)} />
              <ContextStat label="Workloads" value={String(summary?.workload_count ?? 0)} />
              <ContextStat label="Snapshots" value={String(summary?.snapshot_count ?? 0)} />
              <ContextStat label="Active migrations" value={String(summary?.active_migrations ?? 0)} />
              <ContextStat label="Pending approvals" value={String(summary?.pending_approvals ?? 0)} />
            </div>
          )}
        </SectionCard>

        <SectionCard title="Runtime credential handling" description="Session-scoped browser storage is the default operator path. Remembered keys should be limited to trusted workstations.">
          <div className="grid gap-3 md:grid-cols-2">
            <ContextStat label="Credential source" value={authSourceLabel} />
            <ContextStat label="Persistence" value={authPersistenceLabel} />
            <ContextStat label="Recommended default" value="Session-scoped browser storage" />
            <ContextStat label="Shared-workstation guidance" value="Forget remembered keys after the session ends" />
          </div>
          {onForgetRuntimeKey ? (
            <div className="mt-4 flex flex-wrap gap-2">
              <button type="button" onClick={onForgetRuntimeKey} className="operator-button-danger">
                Forget browser key
              </button>
            </div>
          ) : (
            <p className="mt-4 text-sm text-slate-600">
              No browser-managed runtime key is active right now. If the dashboard is using an environment-provided key, rotate it through the deployment environment instead.
            </p>
          )}
        </SectionCard>

        <SectionCard title="Permissions and quotas" description="Effective operator permissions and tenant-level fairness controls." className="xl:col-span-2">
          {currentTenant ? (
            <>
              <div className="flex flex-wrap gap-2">
                {currentTenant.permissions.map((permission) => (
                  <StatusBadge key={permission} tone="info">
                    {permission}
                  </StatusBadge>
                ))}
                {currentTenant.permissions.length === 0 && <StatusBadge tone="neutral">No permissions reported</StatusBadge>}
              </div>
              <div className="mt-4 grid gap-3 md:grid-cols-3">
                <ContextStat label="Requests/min" value={String(currentTenant.quotas?.requests_per_minute ?? 0)} />
                <ContextStat label="Max snapshots" value={String(currentTenant.quotas?.max_snapshots ?? 0)} />
                <ContextStat label="Max migrations" value={String(currentTenant.quotas?.max_migrations ?? 0)} />
              </div>
            </>
          ) : (
            <p className="text-sm text-slate-500">Tenant permissions and quotas appear here when the current tenant endpoint is available.</p>
          )}
        </SectionCard>

        <SectionCard title="Observed platform coverage" description="Supported platforms from the operator build and workload counts from the current tenant summary." className="xl:col-span-2">
          <div className="flex flex-wrap gap-2">
            {supportedPlatforms.map((platform) => (
              <StatusBadge key={platform} tone={(platformCounts[platform] ?? 0) > 0 ? "info" : "neutral"}>
                {platform}: {platformCounts[platform] ?? 0}
              </StatusBadge>
            ))}
            {supportedPlatforms.length === 0 && <StatusBadge tone="neutral">No platforms reported</StatusBadge>}
          </div>
          <p className="mt-4 text-sm text-slate-500">
            Build metadata is sourced from `/api/v1/about`, while tenant role, auth method, and effective permissions come from `/api/v1/tenants/current`.
          </p>
        </SectionCard>
      </section>
    </div>
  );
}

function SettingRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 text-sm font-semibold text-ink">{value}</p>
    </div>
  );
}

function ContextStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 font-semibold text-ink">{value}</p>
    </div>
  );
}

function InlineError({ error }: { error: ErrorDisplay }) {
  return (
    <div className="rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">
      <p>{error.message}</p>
      {error.technicalDetails.length > 0 && (
        <div className="mt-3 rounded-2xl bg-white/70 px-4 py-3 text-xs text-rose-800">
          {error.technicalDetails.map((detail, index) => (
            <p key={`${detail}-${index}`} className={index === 0 ? undefined : "mt-1"}>
              {detail}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
