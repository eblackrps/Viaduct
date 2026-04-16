/// <reference types="vite/client" />

interface ImportMetaEnv {
	readonly VITE_VIADUCT_API_KEY?: string;
	readonly VITE_VIADUCT_SERVICE_ACCOUNT_KEY?: string;
	readonly VITE_VIADUCT_API_TIMEOUT_MS?: string;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
