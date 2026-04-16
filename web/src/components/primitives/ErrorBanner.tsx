import { AlertTriangle, X } from "lucide-react";

interface ErrorBannerProps {
	message: string;
	onDismiss?: () => void;
}

export function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
	return (
		<div
			role="alert"
			className="flex items-start gap-3 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
		>
			<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-rose-500" />
			<p className="flex-1 leading-5">{message}</p>
			{onDismiss && (
				<button
					type="button"
					onClick={onDismiss}
					aria-label="Dismiss error"
					className="ml-auto shrink-0 rounded p-0.5 hover:bg-rose-100"
				>
					<X className="h-4 w-4" />
				</button>
			)}
		</div>
	);
}
