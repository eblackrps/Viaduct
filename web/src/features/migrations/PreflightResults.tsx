import { StatusBadge } from "../../components/primitives/StatusBadge";
import { getCheckTone, getReadableCheckName } from "./migrationStatus";
import type { CheckResult } from "../../types";

interface PreflightResultsProps {
	checks: CheckResult[];
}

export function PreflightResults({ checks }: PreflightResultsProps) {
	return (
		<div className="space-y-5">
			<Bucket
				title="Blocking checks"
				checks={checks.filter((check) => check.status === "fail")}
				emptyMessage="No blocking checks were reported in the latest validation run."
			/>
			<Bucket
				title="Warnings"
				checks={checks.filter((check) => check.status === "warn")}
				emptyMessage="No warning checks were reported in the latest validation run."
			/>
			<Bucket
				title="Passed checks"
				checks={checks.filter((check) => check.status === "pass")}
				emptyMessage="No passing checks were reported in the latest validation run."
			/>
		</div>
	);
}

function Bucket({
	title,
	checks,
	emptyMessage,
}: {
	title: string;
	checks: CheckResult[];
	emptyMessage: string;
}) {
	return (
		<div>
			<p className="text-xs uppercase tracking-[0.18em] text-slate-500">
				{title}
			</p>
			<div className="mt-3 space-y-3">
				{checks.length > 0 ? (
					checks.map((check) => (
						<article
							key={check.name}
							className="rounded-2xl border border-slate-200 bg-white px-4 py-4 text-sm text-slate-600"
						>
							<div className="flex flex-wrap items-center justify-between gap-3">
								<p className="font-semibold text-ink">
									{getReadableCheckName(check.name)}
								</p>
								<StatusBadge tone={getCheckTone(check.status)}>
									{check.status}
								</StatusBadge>
							</div>
							<p className="mt-2 text-slate-500">{check.message}</p>
						</article>
					))
				) : (
					<p className="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-600">
						{emptyMessage}
					</p>
				)}
			</div>
		</div>
	);
}
