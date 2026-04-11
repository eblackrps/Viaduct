import type { ReactNode } from "react";

export type StatusTone = "neutral" | "info" | "success" | "warning" | "danger" | "accent";

interface StatusBadgeProps {
  children: ReactNode;
  tone?: StatusTone;
  className?: string;
}

const toneClasses: Record<StatusTone, string> = {
  neutral: "border border-slate-200 bg-slate-100/90 text-slate-700",
  info: "border border-sky-200 bg-sky-100/90 text-sky-700",
  success: "border border-emerald-200 bg-emerald-100/90 text-emerald-700",
  warning: "border border-amber-200 bg-amber-100/90 text-amber-800",
  danger: "border border-rose-200 bg-rose-100/90 text-rose-700",
  accent: "border border-ink bg-ink text-white",
};

export function StatusBadge({ children, tone = "neutral", className }: StatusBadgeProps) {
  const classes = ["inline-flex items-center rounded-full px-3 py-1.5 text-xs font-semibold tracking-[0.04em]", toneClasses[tone], className]
    .filter(Boolean)
    .join(" ");

  return <span className={classes}>{children}</span>;
}
