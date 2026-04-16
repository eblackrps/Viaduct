import type { PolicyReport } from "../types";

interface PolicyDashboardProps {
	report: PolicyReport | null;
}

export function PolicyDashboard({ report }: PolicyDashboardProps) {
	return (
		<section className="panel p-5">
			<div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
				<div>
					<p className="font-display text-2xl text-ink">Policy Engine</p>
					<p className="text-sm text-slate-500">
						Enforcement, compliance, affinity, and cost policies evaluated
						against the latest inventory.
					</p>
				</div>
				<div className="flex gap-3 text-sm">
					<span className="rounded-full bg-emerald-100 px-3 py-1 font-semibold text-emerald-700">
						{report?.compliant_vms ?? 0} compliant
					</span>
					<span className="rounded-full bg-amber-100 px-3 py-1 font-semibold text-amber-700">
						{report?.non_compliant_vms ?? 0} flagged
					</span>
					<span className="rounded-full bg-slate-200 px-3 py-1 font-semibold text-slate-700">
						{report?.waived_violations ?? 0} waived
					</span>
				</div>
			</div>

			<div className="mt-5 grid gap-4 xl:grid-cols-[0.85fr_1.15fr]">
				<div className="space-y-3">
					{(report?.policies ?? []).map((policy) => (
						<article
							key={policy.name}
							className="rounded-2xl bg-slate-50 px-4 py-4"
						>
							<p className="font-semibold text-ink">{policy.name}</p>
							<p className="mt-1 text-sm text-slate-500">
								{policy.type} · {policy.severity}
							</p>
							{policy.description && (
								<p className="mt-2 text-sm text-slate-600">
									{policy.description}
								</p>
							)}
						</article>
					))}
				</div>

				<div className="space-y-3">
					{(report?.violations ?? []).map((violation, index) => (
						<article
							key={`${violation.vm.id}-${violation.rule.field}-${index}`}
							className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-4"
						>
							<div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
								<div>
									<p className="font-semibold text-ink">{violation.vm.name}</p>
									<p className="text-sm text-slate-600">
										{violation.policy.name} · {violation.rule.field}
									</p>
								</div>
								<span className="rounded-full bg-ink px-3 py-1 text-xs font-semibold text-white">
									{violation.severity}
								</span>
							</div>
							<p className="mt-3 text-sm text-slate-700">
								{violation.remediation ??
									violation.rule.message ??
									"Review this workload against the policy."}
							</p>
						</article>
					))}

					{(report?.violations ?? []).length === 0 && (
						<p className="rounded-2xl border border-dashed border-slate-300 px-4 py-6 text-sm text-slate-500">
							No policy violations are currently reported.
						</p>
					)}
				</div>
			</div>
		</section>
	);
}
