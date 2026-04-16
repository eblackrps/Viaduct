import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ErrorState } from "./ErrorState";
import { PageHeader } from "./PageHeader";
import { StatusBadge } from "./StatusBadge";

describe("primitives", () => {
	it("renders status badge content", () => {
		render(<StatusBadge tone="success">Healthy</StatusBadge>);

		expect(screen.getByText("Healthy")).toBeInTheDocument();
	});

	it("highlights request IDs inside error state details", () => {
		render(
			<ErrorState
				title="Authentication failed"
				message="Unable to validate the configured dashboard credentials."
				technicalDetails={["HTTP status: 401", "Request ID: req-123"]}
			/>,
		);

		expect(
			screen.getByText(
				"Capture this request ID when escalating or retrying the workflow.",
			),
		).toBeInTheDocument();
		expect(screen.getByText("Request ID: req-123")).toBeInTheDocument();
	});

	it("renders page header badges and actions", () => {
		render(
			<PageHeader
				eyebrow="Inventory"
				title="Fleet inventory"
				description="Normalized operator view"
				badges={[{ label: "75 workloads", tone: "info" }]}
				actions={<button type="button">Refresh</button>}
			/>,
		);

		expect(
			screen.getByRole("heading", { name: "Fleet inventory" }),
		).toBeInTheDocument();
		expect(screen.getByText("75 workloads")).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Refresh" })).toBeInTheDocument();
	});
});
