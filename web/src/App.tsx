import { AppErrorBoundary } from "./app/AppErrorBoundary";
import { AppRoutes } from "./app/AppRoutes";

export default function App() {
	return (
		<AppErrorBoundary>
			<AppRoutes />
		</AppErrorBoundary>
	);
}
