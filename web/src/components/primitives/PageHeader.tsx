import type { ReactNode } from "react";
import { StatusBadge, type StatusTone } from "./StatusBadge";

interface PageHeaderBadge {
	label: string;
	tone?: StatusTone;
}

interface PageHeaderProps {
	eyebrow?: string;
	title: string;
	description: string;
	badges?: PageHeaderBadge[];
	actions?: ReactNode;
	titleAs?: "h1" | "h2" | "h3";
}

export function PageHeader({
	eyebrow,
	title,
	description,
	badges,
	actions,
	titleAs = "h1",
}: PageHeaderProps) {
	const TitleTag = titleAs;

	return (
		<header className="panel px-6 py-6 lg:px-7 lg:py-7">
			<div className="absolute inset-x-0 top-0 h-28 bg-gradient-to-r from-sky-100/80 via-white/0 to-amber-100/70" />
			<div className="relative flex flex-col gap-6 xl:flex-row xl:items-start xl:justify-between">
				<div className="min-w-0 max-w-4xl">
					{eyebrow && <p className="operator-kicker">{eyebrow}</p>}
					<TitleTag className="mt-3 font-display text-[2rem] leading-tight tracking-[-0.03em] text-ink lg:text-[2.5rem]">
						{title}
					</TitleTag>
					<p className="mt-3 max-w-3xl text-sm leading-7 text-slate-600 lg:text-[0.98rem]">
						{description}
					</p>
					{badges && badges.length > 0 && (
						<div className="mt-5 flex flex-wrap gap-2">
							{badges.map((badge) => (
								<StatusBadge
									key={`${badge.label}-${badge.tone ?? "neutral"}`}
									tone={badge.tone}
								>
									{badge.label}
								</StatusBadge>
							))}
						</div>
					)}
				</div>
				{actions && (
					<div className="flex max-w-xl flex-wrap items-center gap-2 xl:justify-end">
						{actions}
					</div>
				)}
			</div>
		</header>
	);
}
