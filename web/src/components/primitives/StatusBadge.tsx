import type { ReactNode } from "react";

export type StatusTone =
	| "neutral"
	| "info"
	| "success"
	| "warning"
	| "danger"
	| "accent";

interface StatusBadgeProps {
	children: ReactNode;
	tone?: StatusTone;
	className?: string;
}

const toneClasses: Record<StatusTone, string> = {
	neutral:
		"border-slate-200/90 bg-slate-100/95 text-slate-700 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
	info: "border-sky-200/90 bg-sky-50/95 text-sky-800 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
	success:
		"border-emerald-200/90 bg-emerald-50/95 text-emerald-800 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
	warning:
		"border-amber-200/90 bg-amber-50/95 text-amber-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
	danger:
		"border-rose-200/90 bg-rose-50/95 text-rose-800 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
	accent:
		"border-steel/15 bg-steel/10 text-steel shadow-[inset_0_1px_0_rgba(255,255,255,0.75)]",
};

const dotClasses: Record<StatusTone, string> = {
	neutral: "bg-slate-400",
	info: "bg-sky-500",
	success: "bg-emerald-500",
	warning: "bg-amber-500",
	danger: "bg-rose-500",
	accent: "bg-steel",
};

export function StatusBadge({
	children,
	tone = "neutral",
	className,
}: StatusBadgeProps) {
	const classes = [
		"inline-flex items-center gap-2 rounded-full border px-3.5 py-1.5 text-[0.72rem] font-semibold leading-none tracking-[0.06em]",
		toneClasses[tone],
		className,
	]
		.filter(Boolean)
		.join(" ");

	return (
		<span className={classes}>
			<span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${dotClasses[tone]}`} />
			<span>{children}</span>
		</span>
	);
}
