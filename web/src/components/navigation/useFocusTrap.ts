import { useEffect } from "react";
import type { RefObject } from "react";

export function useFocusTrap(
	containerRef: RefObject<HTMLElement | null>,
	enabled: boolean,
	onEscape: () => void,
) {
	useEffect(() => {
		if (!enabled) {
			return;
		}

		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		const focusable = getFocusableElements(containerRef.current);
		focusable[0]?.focus();

		function handleKeyDown(event: KeyboardEvent) {
			if (!containerRef.current) {
				return;
			}

			if (event.key === "Escape") {
				event.preventDefault();
				onEscape();
				return;
			}

			if (event.key !== "Tab") {
				return;
			}

			const items = getFocusableElements(containerRef.current);
			if (items.length === 0) {
				event.preventDefault();
				return;
			}

			const first = items[0];
			const last = items[items.length - 1];
			const activeElement = document.activeElement;

			if (event.shiftKey && activeElement === first) {
				event.preventDefault();
				last.focus();
				return;
			}

			if (!event.shiftKey && activeElement === last) {
				event.preventDefault();
				first.focus();
			}
		}

		document.addEventListener("keydown", handleKeyDown);
		return () => {
			document.body.style.overflow = previousOverflow;
			document.removeEventListener("keydown", handleKeyDown);
		};
	}, [containerRef, enabled, onEscape]);
}

function getFocusableElements(element: HTMLElement | null) {
	if (!element) {
		return [];
	}

	return Array.from(
		element.querySelectorAll<HTMLElement>(
			'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
		),
	).filter((candidate) => !candidate.hasAttribute("disabled"));
}
