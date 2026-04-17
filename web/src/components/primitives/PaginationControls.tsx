interface PaginationControlsProps {
	currentPage: number;
	totalPages: number;
	totalItems: number;
	itemLabel: string;
	onPageChange: (page: number) => void;
}

export function PaginationControls({
	currentPage,
	totalPages,
	totalItems,
	itemLabel,
	onPageChange,
}: PaginationControlsProps) {
	return (
		<div className="pagination-shell">
			<p className="text-sm text-slate-600">
				Page {currentPage} of {totalPages} • {totalItems.toLocaleString()}{" "}
				{itemLabel}
			</p>
			<div className="flex flex-wrap items-center gap-2">
				<button
					type="button"
					onClick={() => onPageChange(Math.max(1, currentPage - 1))}
					disabled={currentPage <= 1}
					className="operator-button-secondary px-3.5 py-2"
				>
					Previous
				</button>
				<button
					type="button"
					onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
					disabled={currentPage >= totalPages}
					className="operator-button-secondary px-3.5 py-2"
				>
					Next
				</button>
			</div>
		</div>
	);
}
