import { useEffect, useRef } from "react";
import type { RefObject } from "react";

export function useFocusTrap(
	containerRef: RefObject<HTMLElement | null>,
	enabled: boolean,
	onEscape: () => void,
) {
	const onEscapeRef = useRef(onEscape);
	const warnedBodyFallbackRef = useRef(false);

	useEffect(() => {
		onEscapeRef.current = onEscape;
	}, [onEscape]);

	useEffect(() => {
		if (!enabled) {
			return;
		}

		const trapRoot = containerRef.current;
		const previousActiveElement = document.activeElement as HTMLElement | null;
		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		const focusable = getFocusableElements(trapRoot);
		if (focusable.length > 0) {
			focusable[0].focus();
		} else {
			trapRoot?.focus();
		}

		function handleKeyDown(event: KeyboardEvent) {
			if (!trapRoot) {
				return;
			}

			if (event.key === "Escape") {
				event.preventDefault();
				onEscapeRef.current();
				return;
			}

			if (event.key !== "Tab") {
				return;
			}

			const items = getFocusableElements(trapRoot);
			if (items.length === 0) {
				event.preventDefault();
				trapRoot.focus();
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
			if (previousActiveElement && document.contains(previousActiveElement)) {
				previousActiveElement.focus();
				return;
			}
			if (trapRoot && document.contains(trapRoot)) {
				trapRoot.focus();
				return;
			}
			document.body.focus();
			if (!warnedBodyFallbackRef.current) {
				warnedBodyFallbackRef.current = true;
				console.warn("useFocusTrap fell back to document.body during cleanup");
			}
		};
	}, [containerRef, enabled]);
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
