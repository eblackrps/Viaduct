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
    <header className="rounded-xl border border-slate-200/80 bg-slate-50/80 p-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          {eyebrow && <p className="operator-kicker">{eyebrow}</p>}
          <p className="mt-2 font-display text-2xl text-ink">{title}</p>
          <p className="mt-2 max-w-3xl text-sm leading-5 text-slate-600">{description}</p>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2 md:justify-end">{actions}</div>}
      </div>

      {badges && badges.length > 0 && (
        <div className="mt-4 flex flex-wrap gap-2">
          {badges.map((badge) => (
            <StatusBadge key={`${badge.label}-${badge.tone ?? "neutral"}`} tone={badge.tone}>
              {badge.label}
            </StatusBadge>
          ))}
        </div>
      )}
    </header>
  );
}
