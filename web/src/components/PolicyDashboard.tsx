import type { PolicyReport } from "../types";
import { InlineNotice } from "./primitives/InlineNotice";
import { SectionCard } from "./primitives/SectionCard";
import { StatusBadge } from "./primitives/StatusBadge";

interface PolicyDashboardProps {
	report: PolicyReport | null;
}

export function PolicyDashboard({ report }: PolicyDashboardProps) {
	return (
		<SectionCard
			title="Policy engine"
			description="Enforcement, compliance, affinity, and cost policies evaluated against the latest inventory."
			actions={
				<div className="flex flex-wrap gap-2">
					<StatusBadge tone="success">
						{report?.compliant_vms ?? 0} compliant
					</StatusBadge>
					<StatusBadge tone="warning">
						{report?.non_compliant_vms ?? 0} flagged
					</StatusBadge>
					<StatusBadge tone="neutral">
						{report?.waived_violations ?? 0} waived
					</StatusBadge>
				</div>
			}
		>
			<div className="grid gap-4 xl:grid-cols-[0.8fr_1.2fr]">
				<div className="space-y-3">
					{(report?.policies ?? []).map((policy) => (
						<article key={policy.name} className="list-card">
							<div className="flex flex-wrap gap-2">
								<StatusBadge tone="neutral">{policy.type}</StatusBadge>
								<StatusBadge tone="info">{policy.severity}</StatusBadge>
							</div>
							<p className="mt-3 font-semibold text-ink">{policy.name}</p>
							{policy.description ? (
								<p className="mt-2 text-sm leading-6 text-slate-600">
									{policy.description}
								</p>
							) : null}
						</article>
					))}

					{(report?.policies ?? []).length === 0 ? (
						<InlineNotice message="No policies were returned." tone="neutral" />
					) : null}
				</div>

				<div className="space-y-3">
					{(report?.violations ?? []).map((violation, index) => (
						<article
							key={`${violation.vm.id}-${violation.rule.field}-${index}`}
							className="rounded-2xl border border-amber-200/90 bg-amber-50/80 px-4 py-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]"
						>
							<div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
								<div>
									<p className="font-semibold text-ink">{violation.vm.name}</p>
									<p className="mt-1 text-sm text-slate-600">
										{violation.policy.name} • {violation.rule.field}
									</p>
								</div>
								<StatusBadge tone="warning">{violation.severity}</StatusBadge>
							</div>
							<p className="mt-3 text-sm leading-6 text-slate-700">
								{violation.remediation ??
									violation.rule.message ??
									"Review this workload against the policy."}
							</p>
						</article>
					))}

					{(report?.violations ?? []).length === 0 ? (
						<InlineNotice
							message="No policy violations are currently reported."
							tone="success"
						/>
					) : null}
				</div>
			</div>
		</SectionCard>
	);
}
