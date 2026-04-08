import { PolicyDashboard } from "../../components/PolicyDashboard";
import { EmptyState } from "../../components/primitives/EmptyState";
import { ErrorState } from "../../components/primitives/ErrorState";
import { LoadingState } from "../../components/primitives/LoadingState";
import { PageHeader } from "../../components/primitives/PageHeader";
import { SectionCard } from "../../components/primitives/SectionCard";
import { useLifecycleSignals } from "../lifecycle/useLifecycleSignals";

interface PolicyPageProps {
  refreshToken: number;
}

export function PolicyPage({ refreshToken }: PolicyPageProps) {
  const { policies: report, loading, errors } = useLifecycleSignals({ refreshToken, includePolicies: true });
  const error = errors.policies;

  return (
    <div className="space-y-6">
      <PageHeader
        eyebrow="Policy"
        title="Policy controls"
        description="Inspect evaluated rules, rule-level violations, and the current enforcement posture across discovered workloads."
        badges={[
          { label: `${report?.policies.length ?? 0} policies`, tone: "neutral" },
          { label: `${report?.violations.length ?? 0} violations`, tone: (report?.violations.length ?? 0) > 0 ? "warning" : "success" },
        ]}
      />

      {loading && !report && (
        <LoadingState
          title="Loading policy results"
          message="Evaluating lifecycle policies and loading the latest compliance posture for the current tenant."
        />
      )}

      {!loading && !report && (
        error ? (
          <ErrorState title="Policy evaluation unavailable" message={error} />
        ) : (
          <EmptyState
            title="No policy report available"
            message="The policy engine has not returned an evaluation result for the current tenant yet."
          />
        )
      )}

      {report && (
        <>
          <SectionCard title="Evaluation context" description="High-level policy execution metadata for the current report.">
            <div className="grid gap-3 md:grid-cols-4">
              <PolicyContext label="Policies" value={String(report.policies.length)} />
              <PolicyContext label="Violations" value={String(report.violations.length)} />
              <PolicyContext label="Waived" value={String(report.waived_violations)} />
              <PolicyContext label="Evaluated" value={new Date(report.evaluated_at).toLocaleString()} />
            </div>
          </SectionCard>
          <PolicyDashboard report={report} />
        </>
      )}
    </div>
  );
}

function PolicyContext({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl bg-slate-50 px-4 py-4">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 font-semibold text-ink">{value}</p>
    </div>
  );
}
