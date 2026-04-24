export interface FrontendErrorEvent {
	type: "render" | "error" | "unhandledrejection";
	message: string;
	stack?: string;
	componentStack?: string;
	source?: string;
	timestamp: string;
}

declare global {
	interface Window {
		__VIADUCT_FRONTEND_MONITOR__?: {
			capture: (event: FrontendErrorEvent) => void;
		};
	}
}

export function reportFrontendError(event: FrontendErrorEvent) {
	if (typeof window === "undefined") {
		return;
	}

	window.dispatchEvent(
		new CustomEvent<FrontendErrorEvent>("viaduct:frontend-error", {
			detail: event,
		}),
	);
	window.__VIADUCT_FRONTEND_MONITOR__?.capture(event);
}

export function installGlobalErrorHandlers() {
	if (typeof window === "undefined") {
		return () => undefined;
	}

	const handleError = (event: ErrorEvent) => {
		reportFrontendError({
			type: "error",
			message: normalizeErrorMessage(
				event.error,
				event.message,
				"Unhandled window error",
			),
			stack: normalizeErrorStack(event.error),
			source: [event.filename, event.lineno, event.colno]
				.filter(Boolean)
				.join(":"),
			timestamp: new Date().toISOString(),
		});
	};

	const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
		reportFrontendError({
			type: "unhandledrejection",
			message: normalizeErrorMessage(
				event.reason,
				"",
				"Unhandled promise rejection",
			),
			stack: normalizeErrorStack(event.reason),
			timestamp: new Date().toISOString(),
		});
	};

	window.addEventListener("error", handleError);
	window.addEventListener("unhandledrejection", handleUnhandledRejection);

	return () => {
		window.removeEventListener("error", handleError);
		window.removeEventListener("unhandledrejection", handleUnhandledRejection);
	};
}

function normalizeErrorMessage(
	value: unknown,
	fallbackMessage: string,
	ultimateFallback: string,
) {
	if (value instanceof Error && value.message.trim() !== "") {
		return value.message.trim();
	}
	if (typeof fallbackMessage === "string" && fallbackMessage.trim() !== "") {
		return fallbackMessage.trim();
	}
	if (typeof value === "string" && value.trim() !== "") {
		return value.trim();
	}
	return ultimateFallback;
}

function normalizeErrorStack(value: unknown) {
	if (value instanceof Error && typeof value.stack === "string") {
		return value.stack;
	}
	return undefined;
}
