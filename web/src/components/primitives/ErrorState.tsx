import type { ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { SectionCard } from "./SectionCard";

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
		<SectionCard className="border-rose-200 bg-rose-50/90">
			<div className="flex items-start gap-4" role="alert">
				<div className="rounded-2xl bg-rose-100 p-3 text-rose-700">
					<AlertTriangle className="h-5 w-5" />
				</div>
				<div>
					<TitleTag className="font-display text-2xl text-ink">
						{title}
					</TitleTag>
					<p className="mt-2 max-w-2xl text-sm leading-6 text-rose-700">
						{message}
					</p>
					{requestDetail && (
						<div className="mt-4 rounded-2xl border border-rose-200 bg-white/80 px-4 py-3 text-xs text-rose-800">
							<p className="font-semibold text-rose-900">
								Capture this request ID when escalating or retrying the
								workflow.
							</p>
							<p className="mt-1">{requestDetail}</p>
						</div>
					)}
					{remainingDetails.length > 0 && (
						<div className="mt-4 rounded-2xl bg-white/70 px-4 py-3 text-xs text-rose-800">
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
						<div className="mt-4 flex flex-wrap gap-2">{actions}</div>
					)}
				</div>
			</div>
		</SectionCard>
	);
}
