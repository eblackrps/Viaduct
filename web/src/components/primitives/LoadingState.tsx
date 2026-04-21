import type { ReactNode } from "react";
import { RefreshCcw } from "lucide-react";

interface LoadingStateProps {
	title: string;
	message: string;
	actions?: ReactNode;
	className?: string;
	titleAs?: "h1" | "h2" | "h3";
}

export function LoadingState({
	title,
	message,
	actions,
	className,
	titleAs = "h2",
}: LoadingStateProps) {
	const TitleTag = titleAs;

	return (
		<section
			className={className ? `state-shell ${className}` : "state-shell"}
			role="status"
			aria-live="polite"
		>
			<div className="flex items-start gap-4">
				<div className="state-icon bg-slate-100/90 text-slate-600">
					<RefreshCcw className="h-5 w-5 animate-spin" />
				</div>
				<div>
					<TitleTag className="font-display text-title text-ink">
						{title}
					</TitleTag>
					<p className="mt-3 max-w-2xl text-body-sm text-slate-600">
						{message}
					</p>
					{actions && (
						<div className="mt-5 flex flex-wrap gap-2">{actions}</div>
					)}
				</div>
			</div>
		</section>
	);
}
