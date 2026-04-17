import type { ReactNode } from "react";

interface SectionCardProps {
	eyebrow?: string;
	title?: string;
	description?: string;
	actions?: ReactNode;
	children?: ReactNode;
	className?: string;
	bodyClassName?: string;
	titleAs?: "h2" | "h3" | "h4";
}

export function SectionCard({
	eyebrow,
	title,
	description,
	actions,
	children,
	className,
	bodyClassName,
	titleAs = "h2",
}: SectionCardProps) {
	const wrapperClassName = ["panel px-5 py-5 lg:px-6 lg:py-6", className]
		.filter(Boolean)
		.join(" ");
	const contentClassName = [
		title || description || actions ? "mt-6" : "",
		bodyClassName,
	]
		.filter(Boolean)
		.join(" ");
	const TitleTag = titleAs;

	return (
		<section className={wrapperClassName}>
			{(title || description || actions) && (
				<div className="relative flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
					<div className="min-w-0 max-w-4xl">
						{eyebrow && <p className="operator-kicker">{eyebrow}</p>}
						{title && (
							<TitleTag className="mt-2 font-display text-[1.55rem] leading-tight tracking-[-0.03em] text-ink lg:text-[1.75rem]">
								{title}
							</TitleTag>
						)}
						{description && (
							<p className="mt-3 max-w-3xl text-sm leading-7 text-slate-600">
								{description}
							</p>
						)}
					</div>
					{actions && (
						<div className="flex max-w-xl flex-wrap items-center gap-2 xl:justify-end">
							{actions}
						</div>
					)}
				</div>
			)}
			{children && <div className={contentClassName}>{children}</div>}
		</section>
	);
}
