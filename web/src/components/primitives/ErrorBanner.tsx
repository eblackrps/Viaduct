import { AlertTriangle, X } from "lucide-react";

interface ErrorBannerProps {
	message: string;
	onDismiss?: () => void;
}

export function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
	return (
		<div
			role="alert"
			className="flex items-start gap-3 rounded-[22px] border border-rose-200/90 bg-rose-50/92 px-4 py-3 text-sm text-rose-800 shadow-[inset_0_1px_0_rgba(255,255,255,0.8)]"
		>
			<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
			<p className="flex-1 leading-5">{message}</p>
			{onDismiss && (
				<button
					type="button"
					onClick={onDismiss}
					aria-label="Dismiss error"
					className="ml-auto shrink-0 rounded-full p-1 transition hover:bg-rose-100"
				>
					<X className="h-4 w-4" />
				</button>
			)}
		</div>
	);
}
