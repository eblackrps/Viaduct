import type { ReactNode } from "react";
import { AlertTriangle } from "lucide-react";

interface ErrorStateProps {
	title: string;
	message: string;
	technicalDetails?: string[];
	actions?: ReactNode;
	titleAs?: "h1" | "h2" | "h3";
}

export function ErrorState({
	title,
	message,
	technicalDetails = [],
	actions,
	titleAs = "h2",
}: ErrorStateProps) {
	const requestDetail = technicalDetails.find((detail) =>
		detail.startsWith("Request ID:"),
	);
	const remainingDetails = technicalDetails.filter(
		(detail) => detail !== requestDetail,
	);
	const TitleTag = titleAs;

	return (
		<section
			className="state-shell border-rose-200/90 bg-rose-50/90"
			role="alert"
		>
			<div className="flex items-start gap-4">
				<div className="state-icon bg-rose-100/90 text-rose-700">
					<AlertTriangle className="h-5 w-5" />
				</div>
				<div>
					<TitleTag className="font-display text-[1.8rem] leading-tight tracking-[-0.03em] text-ink">
						{title}
					</TitleTag>
					<p className="mt-3 max-w-2xl text-sm leading-7 text-rose-800">
						{message}
					</p>
					{requestDetail && (
						<div className="mt-5 rounded-[22px] border border-rose-200 bg-white/80 px-4 py-3 text-xs text-rose-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]">
							<p className="font-semibold text-rose-900">
								Capture this request ID when escalating or retrying the
								workflow.
							</p>
							<p className="mt-1">{requestDetail}</p>
						</div>
					)}
					{remainingDetails.length > 0 && (
						<div className="mt-4 rounded-[22px] border border-rose-200/80 bg-white/72 px-4 py-3 text-xs text-rose-900 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]">
							{remainingDetails.map((detail, index) => (
								<p
									key={`${detail}-${index}`}
									className={index === 0 ? undefined : "mt-1"}
								>
									{detail}
								</p>
							))}
						</div>
					)}
					{actions && (
						<div className="mt-5 flex flex-wrap gap-2">{actions}</div>
					)}
				</div>
			</div>
		</section>
	);
}
