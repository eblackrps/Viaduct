import { fireEvent, render, screen } from "@testing-library/react";
import { useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { MobileSidebarDrawer } from "./MobileSidebarDrawer";
import { useFocusTrap } from "./useFocusTrap";

function MobileDrawerHarness() {
	const [open, setOpen] = useState(false);

	return (
		<>
			<button type="button" onClick={() => setOpen(true)}>
				Open navigation
			</button>
			<MobileSidebarDrawer
				drawerID="mobile-nav"
				open={open}
				onClose={() => setOpen(false)}
				currentPath="/workspaces"
			/>
		</>
	);
}

function EmptyDialogHarness() {
	const [open, setOpen] = useState(false);
	const dialogRef = useRef<HTMLDivElement | null>(null);
	useFocusTrap(dialogRef, open, () => setOpen(false));

	return (
		<>
			<button type="button" onClick={() => setOpen(true)}>
				Open empty dialog
			</button>
			{open ? (
				<div ref={dialogRef} role="dialog" tabIndex={-1}>
					No focusable controls
				</div>
			) : null}
		</>
	);
}

describe("useFocusTrap", () => {
	it("restores focus to the original trigger when the drawer closes", () => {
		render(<MobileDrawerHarness />);

		const trigger = screen.getByRole("button", { name: "Open navigation" });
		trigger.focus();

		fireEvent.click(trigger);

		const closeButton = screen.getByRole("button", { name: "Close navigation" });
		expect(closeButton).toHaveFocus();

		fireEvent.click(closeButton);
		expect(trigger).toHaveFocus();
	});

	it("focuses the dialog container when no focusable elements exist", () => {
		render(<EmptyDialogHarness />);

		fireEvent.click(screen.getByRole("button", { name: "Open empty dialog" }));

		expect(screen.getByRole("dialog")).toHaveFocus();
	});
});
