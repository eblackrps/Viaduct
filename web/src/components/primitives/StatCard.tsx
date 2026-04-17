import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "./StatusBadge";

interface StatCardProps {
	label: string;
	value: ReactNode;
	detail?: ReactNode;
	badge?: {
		label: ReactNode;
		tone?: StatusTone;
	};
	emphasis?: "standard" | "large";
	className?: string;
}

export function StatCard({
	label,
	value,
	detail,
	badge,
	emphasis = "standard",
	className,
}: StatCardProps) {
	const classes = ["metric-card", className].filter(Boolean).join(" ");
	const valueClassName =
		emphasis === "large"
			? "mt-3 font-display text-[2rem] leading-none tracking-[-0.03em] text-ink lg:text-[2.25rem]"
			: "mt-2 text-sm font-semibold text-ink";

	return (
		<article className={classes}>
			<div className="flex items-start justify-between gap-3">
				<p className="operator-kicker">{label}</p>
				{badge ? (
					<StatusBadge tone={badge.tone}>{badge.label}</StatusBadge>
				) : null}
			</div>
			<div className={valueClassName}>{value}</div>
			{detail ? (
				<div className="mt-2 text-sm leading-6 text-slate-600">{detail}</div>
			) : null}
		</article>
	);
}
