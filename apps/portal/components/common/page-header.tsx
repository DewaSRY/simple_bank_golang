import type { ReactNode } from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface Props {
  title: ReactNode;
  description?: ReactNode;
  /** Primary/secondary actions, right-aligned on desktop, below on mobile. */
  actions?: ReactNode;
  /** Ancestor links, rendered as `a › b › title` above the heading. */
  breadcrumbs?: ReactNode[];
  /** Label for the breadcrumb landmark (localized by the caller). */
  breadcrumbLabel?: string;
  /** Rendered inline after the title (e.g. a status badge). */
  titleAddon?: ReactNode;
  className?: string;
}

/**
 * The one page-level heading used across the logged-in app, so every page
 * shares the same title scale, spacing and action placement.
 */
export function PageHeader({
  title,
  description,
  actions,
  breadcrumbs,
  breadcrumbLabel,
  titleAddon,
  className,
}: Props) {
  return (
    <header
      className={cn(
        "flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between",
        className,
      )}
    >
      <div className="min-w-0 space-y-1.5">
        {breadcrumbs?.length ? (
          <nav aria-label={breadcrumbLabel}>
            <ol className="flex flex-wrap items-center gap-1 text-sm text-muted-foreground">
              {breadcrumbs.map((crumb, index) => (
                <li key={index} className="flex items-center gap-1">
                  {crumb}
                  <ChevronRight className="size-3.5 opacity-60" aria-hidden />
                </li>
              ))}
              <li
                aria-current="page"
                className="max-w-[16rem] truncate font-medium text-foreground"
              >
                {title}
              </li>
            </ol>
          </nav>
        ) : null}
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h1 className="truncate text-2xl font-semibold tracking-tight">
            {title}
          </h1>
          {titleAddon}
        </div>
        {description ? (
          <p className="text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex flex-wrap items-center gap-2 sm:shrink-0 sm:justify-end">
          {actions}
        </div>
      ) : null}
    </header>
  );
}

/** Standard page gutter + max width for every logged-in page. */
export function PageContainer({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <main className="flex flex-1 flex-col px-4 py-6 sm:px-6 lg:py-8">
      <div className={cn("mx-auto w-full max-w-6xl space-y-6", className)}>
        {children}
      </div>
    </main>
  );
}
