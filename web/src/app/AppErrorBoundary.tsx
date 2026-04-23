import { Component, type ErrorInfo, type ReactNode } from "react";
import { ErrorState } from "../components/primitives/ErrorState";
import { reportFrontendError } from "./observability";

interface AppErrorBoundaryProps {
	children: ReactNode;
}

interface AppErrorBoundaryState {
	error: Error | null;
}

export class AppErrorBoundary extends Component<
	AppErrorBoundaryProps,
	AppErrorBoundaryState
> {
	override state: AppErrorBoundaryState = {
		error: null,
	};

	static getDerivedStateFromError(error: Error): AppErrorBoundaryState {
		return { error };
	}

	override componentDidCatch(error: Error, errorInfo: ErrorInfo) {
		reportFrontendError({
			type: "render",
			message: error.message || "Dashboard render failure",
			stack: error.stack,
			componentStack: errorInfo.componentStack ?? undefined,
			timestamp: new Date().toISOString(),
		});
	}

	override render() {
		if (this.state.error) {
			return (
				<ErrorState
					title="Dashboard problem"
					message="Viaduct hit a dashboard error before the page could finish rendering. Refresh the page or reopen the session, then capture the browser console if the problem repeats."
					technicalDetails={[
						this.state.error.message,
						...(this.state.error.stack ? [this.state.error.stack] : []),
					]}
					titleAs="h1"
				/>
			);
		}

		return this.props.children;
	}
}
