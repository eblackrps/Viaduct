import type { ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { SectionCard } from "./SectionCard";

interface ErrorStateProps {
  title: string;
  message: string;
  technicalDetails?: string[];
  actions?: ReactNode;
}

export function ErrorState({ title, message, technicalDetails = [], actions }: ErrorStateProps) {
  return (
    <SectionCard className="border-rose-200 bg-rose-50/90">
      <div className="flex items-start gap-4">
        <div className="rounded-2xl bg-rose-100 p-3 text-rose-700">
          <AlertTriangle className="h-5 w-5" />
        </div>
        <div>
          <p className="font-display text-2xl text-ink">{title}</p>
          <p className="mt-2 max-w-2xl text-sm text-rose-700">{message}</p>
          {technicalDetails.length > 0 && (
            <div className="mt-4 rounded-2xl bg-white/70 px-4 py-3 text-xs text-rose-800">
              {technicalDetails.map((detail, index) => (
                <p key={`${detail}-${index}`} className={index === 0 ? undefined : "mt-1"}>
                  {detail}
                </p>
              ))}
            </div>
          )}
          {actions && <div className="mt-4 flex flex-wrap gap-2">{actions}</div>}
        </div>
      </div>
    </SectionCard>
  );
}
