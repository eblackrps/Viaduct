import type { ReactNode } from "react";

export type StatusTone = "neutral" | "info" | "success" | "warning" | "danger" | "accent";

interface StatusBadgeProps {
  children: ReactNode;
  tone?: StatusTone;
  className?: string;
}

const toneClasses: Record<StatusTone, string> = {
  neutral: "bg-slate-100 text-slate-700",
  info: "bg-sky-100 text-sky-700",
  success: "bg-emerald-100 text-emerald-700",
  warning: "bg-amber-100 text-amber-800",
  danger: "bg-rose-100 text-rose-700",
  accent: "bg-ink text-white",
};

export function StatusBadge({ children, tone = "neutral", className }: StatusBadgeProps) {
  const classes = ["inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold", toneClasses[tone], className]
    .filter(Boolean)
    .join(" ");

  return <span className={classes}>{children}</span>;
}
