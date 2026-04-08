import type { ReactNode } from "react";

interface SectionCardProps {
  title?: string;
  description?: string;
  actions?: ReactNode;
  children?: ReactNode;
  className?: string;
  bodyClassName?: string;
}

export function SectionCard({
  title,
  description,
  actions,
  children,
  className,
  bodyClassName,
}: SectionCardProps) {
  const wrapperClassName = ["panel p-5", className].filter(Boolean).join(" ");
  const contentClassName = [title || description || actions ? "mt-5" : "", bodyClassName].filter(Boolean).join(" ");

  return (
    <section className={wrapperClassName}>
      {(title || description || actions) && (
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            {title && <p className="font-display text-2xl text-ink">{title}</p>}
            {description && <p className="mt-1 max-w-3xl text-sm text-slate-500">{description}</p>}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
        </div>
      )}
      {children && <div className={contentClassName}>{children}</div>}
    </section>
  );
}
