import type { ReactNode } from "react";
import { SectionCard } from "./SectionCard";

interface EmptyStateProps {
  title: string;
  message: string;
  actions?: ReactNode;
}

export function EmptyState({ title, message, actions }: EmptyStateProps) {
  return (
    <SectionCard className="border-dashed border-slate-300 bg-white/70">
      <p className="font-display text-2xl text-ink">{title}</p>
      <p className="mt-2 max-w-2xl text-sm text-slate-500">{message}</p>
      {actions && <div className="mt-4 flex flex-wrap gap-2">{actions}</div>}
    </SectionCard>
  );
}
