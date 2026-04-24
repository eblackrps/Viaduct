import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const observabilityMocks = vi.hoisted(() => ({
	reportFrontendError: vi.fn(),
}));

vi.mock("./observability", () => ({
	reportFrontendError: observabilityMocks.reportFrontendError,
}));

import { AppErrorBoundary } from "./AppErrorBoundary";

function ThrowingChild(): never {
	throw new Error("boundary explosion");
}

describe("AppErrorBoundary", () => {
	afterEach(() => {
		vi.clearAllMocks();
	});

	it("renders a fallback state and reports render failures", () => {
		const consoleError = vi
			.spyOn(console, "error")
			.mockImplementation(() => {});

		render(
			<AppErrorBoundary>
				<ThrowingChild />
			</AppErrorBoundary>,
		);

		expect(
			screen.getByRole("heading", { name: "Dashboard problem" }),
		).toBeInTheDocument();
		expect(
			screen.getByText(
				/dashboard error before the page could finish rendering/i,
			),
		).toBeInTheDocument();
		expect(observabilityMocks.reportFrontendError).toHaveBeenCalledWith(
			expect.objectContaining({
				type: "render",
				message: "boundary explosion",
			}),
		);

		consoleError.mockRestore();
	});
});
