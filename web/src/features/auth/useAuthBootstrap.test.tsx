import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const apiMocks = vi.hoisted(() => ({
	createDashboardAuthSession: vi.fn(),
	deleteDashboardAuthSession: vi.fn(),
	describeError: vi.fn((_reason: unknown, options: { fallback: string }) => ({
		message: options.fallback,
		technicalDetails: [],
	})),
	getAbout: vi.fn(),
	getCurrentTenant: vi.fn(),
	isAPIError: vi.fn(
		(reason: unknown) => reason instanceof Error && "status" in reason,
	),
	requestManager: {
		cancelAll: vi.fn(),
	},
}));

const runtimeAuthMocks = vi.hoisted(() => ({
	clearDashboardAuthSession: vi.fn(),
	getDashboardAuthSession: vi.fn(),
	setDashboardAuthSession: vi.fn(),
}));

vi.mock("../../api", () => ({
	createDashboardAuthSession: apiMocks.createDashboardAuthSession,
	deleteDashboardAuthSession: apiMocks.deleteDashboardAuthSession,
	describeError: apiMocks.describeError,
	getAbout: apiMocks.getAbout,
	getCurrentTenant: apiMocks.getCurrentTenant,
	isAPIError: apiMocks.isAPIError,
	requestManager: apiMocks.requestManager,
}));

vi.mock("../../runtimeAuth", () => ({
	clearDashboardAuthSession: runtimeAuthMocks.clearDashboardAuthSession,
	getDashboardAuthSession: runtimeAuthMocks.getDashboardAuthSession,
	setDashboardAuthSession: runtimeAuthMocks.setDashboardAuthSession,
}));

import { useAuthBootstrap } from "./useAuthBootstrap";

describe("useAuthBootstrap", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		apiMocks.getAbout.mockResolvedValue({
			name: "Viaduct",
			api_version: "v1",
			version: "3.2.1",
			commit: "abc123",
			built_at: "2026-04-22T00:00:00Z",
			go_version: "go1.26.0",
			plugin_protocol: "v1",
			local_operator_session_enabled: true,
			supported_platforms: ["kvm"],
			supported_permissions: [],
			store_backend: "memory",
			persistent_store: false,
		});
		apiMocks.createDashboardAuthSession.mockResolvedValue({
			session_id: "session-123",
			mode: "service-account",
		});
		runtimeAuthMocks.getDashboardAuthSession.mockReturnValue({
			mode: "none",
			apiKey: "",
			source: "none",
		});
		const unauthenticatedError = Object.assign(new Error("missing"), {
			status: 401,
		});
		apiMocks.getCurrentTenant
			.mockRejectedValueOnce(unauthenticatedError)
			.mockResolvedValue({
				tenant_id: "tenant-1",
				name: "Default Tenant",
				active: true,
				role: "operator",
				permissions: [],
				auth_method: "service-account",
				service_account_count: 1,
			});
	});

	it("creates a runtime session marker without retaining the raw key", async () => {
		const { result } = renderHook(() => useAuthBootstrap());

		await waitFor(() => {
			expect(result.current.status).toBe("unauthenticated");
		});

		await act(async () => {
			await result.current.connect("service-account", "raw-service-key", true);
		});

		expect(apiMocks.createDashboardAuthSession).toHaveBeenCalledWith(
			"service-account",
			"raw-service-key",
			true,
		);
		expect(runtimeAuthMocks.setDashboardAuthSession).toHaveBeenCalledWith(
			"service-account",
			{
				remember: true,
				sessionID: "session-123",
			},
		);
		expect(result.current.status).toBe("authenticated");
	});
});
