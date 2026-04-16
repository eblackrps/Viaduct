import type { ReactNode } from "react";
import { RefreshCcw } from "lucide-react";
import { SectionCard } from "./SectionCard";

interface LoadingStateProps {
	title: string;
	message: string;
	actions?: ReactNode;
	titleAs?: "h1" | "h2" | "h3";
}

export function LoadingState({
	title,
	message,
	actions,
	titleAs = "h2",
}: LoadingStateProps) {
	const TitleTag = titleAs;

	return (
		<SectionCard className="border-slate-200 bg-white/90">
			<div className="flex items-start gap-4" role="status" aria-live="polite">
				<div className="rounded-2xl bg-slate-100 p-3 text-slate-600">
					<RefreshCcw className="h-5 w-5 animate-spin" />
				</div>
				<div>
					<TitleTag className="font-display text-2xl text-ink">
						{title}
					</TitleTag>
					<p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">
						{message}
					</p>
					{actions && (
						<div className="mt-4 flex flex-wrap gap-2">{actions}</div>
					)}
				</div>
			</div>
		</SectionCard>
	);
}
