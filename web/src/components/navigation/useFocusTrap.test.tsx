import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useRef, useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
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

function RemovedTriggerHarness() {
	const [open, setOpen] = useState(false);
	const [showTrigger, setShowTrigger] = useState(true);
	const dialogRef = useRef<HTMLDivElement | null>(null);
	useFocusTrap(dialogRef, open, () => setOpen(false));

	return (
		<>
			{showTrigger ? (
				<button type="button" onClick={() => setOpen(true)}>
					Open removable dialog
				</button>
			) : null}
			<div ref={dialogRef} role="dialog" tabIndex={-1} hidden={!open}>
				<button type="button" onClick={() => setShowTrigger(false)}>
					Remove trigger
				</button>
				<button type="button" onClick={() => setOpen(false)}>
					Close removable dialog
				</button>
			</div>
		</>
	);
}

function EscapeCallbackHarness({
	onEscape,
	open,
}: {
	onEscape: () => void;
	open: boolean;
}) {
	const dialogRef = useRef<HTMLDivElement | null>(null);
	useFocusTrap(dialogRef, open, onEscape);

	return open ? (
		<div ref={dialogRef} role="dialog" tabIndex={-1}>
			<button type="button">Focusable control</button>
		</div>
	) : null;
}

function DetachedFallbackHarness() {
	const [open, setOpen] = useState(false);
	const [showTrigger, setShowTrigger] = useState(true);
	const dialogRef = useRef<HTMLDivElement | null>(null);
	useFocusTrap(dialogRef, open, () => setOpen(false));

	return (
		<>
			{showTrigger ? (
				<button type="button" onClick={() => setOpen(true)}>
					Open detached dialog
				</button>
			) : null}
			{open ? (
				<div ref={dialogRef} role="dialog" tabIndex={-1}>
					<button type="button" onClick={() => setShowTrigger(false)}>
						Remove detached trigger
					</button>
					<button type="button" onClick={() => setOpen(false)}>
						Close detached dialog
					</button>
				</div>
			) : null}
		</>
	);
}

describe("useFocusTrap", () => {
	afterEach(() => {
		cleanup();
	});

	it("restores focus to the original trigger when the drawer closes", () => {
		render(<MobileDrawerHarness />);

		const trigger = screen.getByRole("button", { name: "Open navigation" });
		trigger.focus();

		fireEvent.click(trigger);

		const closeButton = screen.getByRole("button", {
			name: "Close navigation",
		});
		expect(closeButton).toHaveFocus();

		fireEvent.click(closeButton);
		expect(trigger).toHaveFocus();
	});

	it("focuses the dialog container when no focusable elements exist", () => {
		render(<EmptyDialogHarness />);

		fireEvent.click(screen.getByRole("button", { name: "Open empty dialog" }));

		expect(screen.getByRole("dialog")).toHaveFocus();
	});

	it("keeps focus on the trap container when tabbing with no focusable elements", () => {
		render(<EmptyDialogHarness />);

		fireEvent.click(screen.getByRole("button", { name: "Open empty dialog" }));

		const dialog = screen.getByRole("dialog");
		fireEvent.keyDown(document, { key: "Tab" });
		expect(dialog).toHaveFocus();
	});

	it("falls back to the trap container when the original trigger is removed", () => {
		render(<RemovedTriggerHarness />);

		const trigger = screen.getByRole("button", {
			name: "Open removable dialog",
		});
		trigger.focus();
		fireEvent.click(trigger);

		fireEvent.click(screen.getByRole("button", { name: "Remove trigger" }));

		const dialog = screen.getByRole("dialog");
		fireEvent.click(
			screen.getByRole("button", { name: "Close removable dialog" }),
		);
		expect(dialog).toHaveFocus();
		expect(document.body).not.toHaveFocus();
	});

	it("falls back to document.body and warns once when neither trigger nor trap root remain", () => {
		const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
		render(<DetachedFallbackHarness />);

		const trigger = screen.getByRole("button", {
			name: "Open detached dialog",
		});
		trigger.focus();
		fireEvent.click(trigger);
		fireEvent.click(
			screen.getByRole("button", { name: "Remove detached trigger" }),
		);
		fireEvent.click(
			screen.getByRole("button", { name: "Close detached dialog" }),
		);

		expect(document.body).toHaveFocus();
		expect(warnSpy).toHaveBeenCalledTimes(1);
	});

	it("uses the latest escape callback without re-registering the trap", () => {
		const firstEscape = vi.fn();
		const secondEscape = vi.fn();
		const { rerender } = render(
			<EscapeCallbackHarness open={true} onEscape={firstEscape} />,
		);

		rerender(<EscapeCallbackHarness open={true} onEscape={secondEscape} />);

		fireEvent.keyDown(document, { key: "Escape" });
		expect(firstEscape).not.toHaveBeenCalled();
		expect(secondEscape).toHaveBeenCalledTimes(1);
	});

	it("TestFocusTrapStaleOnEscape", () => {
		const firstEscape = vi.fn();
		const secondEscape = vi.fn();
		const { rerender } = render(
			<EscapeCallbackHarness open={true} onEscape={firstEscape} />,
		);

		rerender(<EscapeCallbackHarness open={true} onEscape={secondEscape} />);

		fireEvent.keyDown(document, { key: "Escape" });
		expect(firstEscape).not.toHaveBeenCalled();
		expect(secondEscape).toHaveBeenCalledTimes(1);
	});
});
