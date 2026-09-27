import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface Props {
  label: ReactNode;
  value: ReactNode;
  subValue?: ReactNode;
  /** Visually promotes the row (e.g. the amount on a review step). */
  emphasis?: boolean;
}

/**
 * One label/value line of a review summary. Stack several inside
 * `PreviewList` so they share dividers and padding.
 */
export function PreviewRow({ label, value, subValue, emphasis }: Props) {
  return (
    <div className="flex flex-col gap-0.5 py-3 first:pt-0 last:pb-0 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
      <dt className="text-sm text-muted-foreground sm:pt-0.5">{label}</dt>
      <dd className="min-w-0 sm:text-right">
        <p
          className={cn(
            "font-medium wrap-break-word",
            emphasis && "font-mono text-lg font-semibold tracking-tight",
          )}
        >
          {value}
        </p>
        {subValue ? (
          <p className="text-sm text-muted-foreground">{subValue}</p>
        ) : null}
      </dd>
    </div>
  );
}

export function PreviewList({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <dl
      className={cn(
        "divide-y divide-border rounded-lg border bg-card px-4 py-4",
        className,
      )}
    >
      {children}
    </dl>
  );
}
