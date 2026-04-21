import type { ReactNode } from "react";
import type { StatusTone } from "./StatusBadge";

interface InlineNoticeProps {
	title?: string;
	message: ReactNode;
	tone?: StatusTone;
	className?: string;
}

const toneClasses: Record<StatusTone, string> = {
	neutral:
		"border-slate-200/80 bg-slate-50/90 text-slate-700 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
	info: "border-sky-200/80 bg-sky-50/90 text-sky-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
	success:
		"border-emerald-200/80 bg-emerald-50/90 text-emerald-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
	warning:
		"border-amber-200/80 bg-amber-50/90 text-amber-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
	danger:
		"border-rose-200/80 bg-rose-50/90 text-rose-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
	accent:
		"border-steel/20 bg-steel/10 text-steel shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]",
};

export function InlineNotice({
	title,
	message,
	tone = "neutral",
	className,
}: InlineNoticeProps) {
	const classes = [
		"rounded-xl border px-4 py-3.5 text-body-sm",
		toneClasses[tone],
		className,
	]
		.filter(Boolean)
		.join(" ");

	return (
		<div className={classes}>
			{title ? <p className="font-semibold text-ink">{title}</p> : null}
			<div className={title ? "mt-1.5" : undefined}>{message}</div>
		</div>
	);
}
