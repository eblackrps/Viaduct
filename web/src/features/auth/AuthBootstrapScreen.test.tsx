import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AuthBootstrapScreen } from "./AuthBootstrapScreen";
import type { AuthBootstrapState } from "./useAuthBootstrap";

function buildAuthState(
	overrides: Partial<AuthBootstrapState> = {},
): AuthBootstrapState {
	return {
		status: "unauthenticated",
		about: {
			name: "Viaduct",
			api_version: "v1",
			version: "3.1.0",
			commit: "abc123",
			built_at: "2026-04-22T00:00:00Z",
			go_version: "go1.26.0",
			plugin_protocol: "v1",
			local_operator_session_enabled: true,
			supported_platforms: ["kvm"],
			supported_permissions: [],
			store_backend: "memory",
			persistent_store: false,
		},
		currentTenant: null,
		localOperatorAvailable: true,
		error: null,
		refresh: vi.fn().mockResolvedValue(undefined),
		connect: vi.fn().mockResolvedValue(undefined),
		connectLocal: vi.fn().mockResolvedValue(undefined),
		signOut: vi.fn(),
		...overrides,
	};
}

describe("AuthBootstrapScreen", () => {
	it("keeps the local session path primary and tucks key sign-in behind a disclosure", () => {
		render(<AuthBootstrapScreen auth={buildAuthState()} />);

		expect(
			screen.getByRole("heading", { name: "Get started" }),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "Start local session" }),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "Use a key instead" }),
		).toBeInTheDocument();
		expect(
			screen.queryByPlaceholderText("Service account key"),
		).not.toBeInTheDocument();
		expect(
			screen.queryByText("Runtime credential bootstrap"),
		).not.toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "Use a key instead" }));

		expect(
			screen.getByPlaceholderText("Service account key"),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "Show advanced options" }),
		).toBeInTheDocument();
		expect(
			screen.queryByRole("button", { name: "Tenant key (advanced)" }),
		).not.toBeInTheDocument();

		fireEvent.click(
			screen.getByRole("button", { name: "Show advanced options" }),
		);

		expect(
			screen.getByRole("button", { name: "Tenant key (advanced)" }),
		).toBeInTheDocument();
	});

	it("clears the key field after a successful key-based sign-in", async () => {
		const connect = vi.fn().mockResolvedValue(undefined);
		render(
			<AuthBootstrapScreen
				auth={buildAuthState({
					localOperatorAvailable: false,
					connect,
				})}
			/>,
		);

		const input = screen.getByPlaceholderText("Service account key");
		fireEvent.change(input, { target: { value: "sa-test-key" } });
		fireEvent.click(screen.getByRole("button", { name: "Start session" }));

		await waitFor(() => {
			expect(connect).toHaveBeenCalledWith(
				"service-account",
				"sa-test-key",
				false,
			);
		});
		expect(input).toHaveValue("");
	});

	it("restores the key field when key-based sign-in fails", async () => {
		const connect = vi.fn().mockRejectedValue(new Error("bad key"));
		render(
			<AuthBootstrapScreen
				auth={buildAuthState({
					localOperatorAvailable: false,
					connect,
				})}
			/>,
		);

		const input = screen.getByPlaceholderText("Service account key");
		fireEvent.change(input, { target: { value: "bad-key" } });
		fireEvent.click(screen.getByRole("button", { name: "Start session" }));

		await waitFor(() => {
			expect(connect).toHaveBeenCalledWith("service-account", "bad-key", false);
		});
		expect(input).toHaveValue("bad-key");
	});
});
