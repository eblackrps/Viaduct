import { AlertTriangle } from "lucide-react";
import { SectionCard } from "./SectionCard";

interface ErrorStateProps {
  title: string;
  message: string;
}

export function ErrorState({ title, message }: ErrorStateProps) {
  return (
    <SectionCard className="border-rose-200 bg-rose-50/90">
      <div className="flex items-start gap-4">
        <div className="rounded-2xl bg-rose-100 p-3 text-rose-700">
          <AlertTriangle className="h-5 w-5" />
        </div>
        <div>
          <p className="font-display text-2xl text-ink">{title}</p>
          <p className="mt-2 max-w-2xl text-sm text-rose-700">{message}</p>
        </div>
      </div>
    </SectionCard>
  );
}
