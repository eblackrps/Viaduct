import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "./StatusBadge";

interface PageHeaderBadge {
  label: string;
  tone?: StatusTone;
}

interface PageHeaderProps {
  eyebrow?: string;
  title: string;
  description: string;
  badges?: PageHeaderBadge[];
  actions?: ReactNode;
}

export function PageHeader({ eyebrow, title, description, badges, actions }: PageHeaderProps) {
  return (
    <header className="flex flex-col gap-4 border-b border-slate-200/80 pb-5 md:flex-row md:items-end md:justify-between">
      <div>
        {eyebrow && <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{eyebrow}</p>}
        <p className="mt-2 font-display text-4xl text-ink">{title}</p>
        <p className="mt-2 max-w-3xl text-sm text-slate-500">{description}</p>
        {badges && badges.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {badges.map((badge) => (
              <StatusBadge key={`${badge.label}-${badge.tone ?? "neutral"}`} tone={badge.tone}>
                {badge.label}
              </StatusBadge>
            ))}
          </div>
        )}
      </div>

      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}
