const splitLayoutMediaQuery = "(min-width: 1800px)";
const reducedMotionMediaQuery = "(prefers-reduced-motion: reduce)";

export function revealWorkloadDetailPanel(element: HTMLElement | null) {
	if (typeof window === "undefined" || !element) {
		return;
	}

	if (window.matchMedia(splitLayoutMediaQuery).matches) {
		return;
	}

	const prefersReducedMotion = window.matchMedia(
		reducedMotionMediaQuery,
	).matches;

	window.requestAnimationFrame(() => {
		element.focus({ preventScroll: true });
		element.scrollIntoView({
			behavior: prefersReducedMotion ? "auto" : "smooth",
			block: "start",
		});
	});
}
