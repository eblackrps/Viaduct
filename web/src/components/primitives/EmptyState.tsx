import { FolderOpenDot } from "lucide-react";
import type { ReactNode } from "react";
import { SectionCard } from "./SectionCard";

interface EmptyStateProps {
	title: string;
	message: string;
	actions?: ReactNode;
	titleAs?: "h1" | "h2" | "h3";
}

export function EmptyState({
	title,
	message,
	actions,
	titleAs = "h2",
}: EmptyStateProps) {
	const TitleTag = titleAs;

	return (
		<SectionCard className="border-dashed border-slate-300 bg-white/70">
			<div className="flex items-start gap-4">
				<div className="rounded-2xl bg-slate-100 p-3 text-slate-600">
					<FolderOpenDot className="h-5 w-5" />
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
