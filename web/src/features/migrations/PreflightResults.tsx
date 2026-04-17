import { InlineNotice } from "../../components/primitives/InlineNotice";
import { StatusBadge } from "../../components/primitives/StatusBadge";
import type { CheckResult } from "../../types";
import { getCheckTone, getReadableCheckName } from "./migrationStatus";

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
			<p className="operator-kicker">{title}</p>
			<div className="mt-3 space-y-3">
				{checks.length > 0 ? (
					checks.map((check) => (
						<article
							key={check.name}
							className="list-card text-sm text-slate-600"
						>
							<div className="flex flex-wrap items-center justify-between gap-3">
								<p className="font-semibold text-ink">
									{getReadableCheckName(check.name)}
								</p>
								<StatusBadge tone={getCheckTone(check.status)}>
									{check.status}
								</StatusBadge>
							</div>
							<p className="mt-3 leading-6 text-slate-600">{check.message}</p>
						</article>
					))
				) : (
					<InlineNotice message={emptyMessage} tone="neutral" />
				)}
			</div>
		</div>
	);
}
