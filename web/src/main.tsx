import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { installGlobalErrorHandlers } from "./app/observability";
import "./index.css";

const detachGlobalErrorHandlers = installGlobalErrorHandlers();
if (import.meta.hot) {
	import.meta.hot.dispose(detachGlobalErrorHandlers);
}

ReactDOM.createRoot(document.getElementById("root")!).render(
	<React.StrictMode>
		<App />
	</React.StrictMode>,
);
