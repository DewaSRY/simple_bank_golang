import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

interface Props {
  name: string;
  number: string;
  subtitle?: string;
  /** Pre-formatted balance; shown right-aligned when present. */
  balance?: string;
  selected: boolean;
  onClick: () => void;
}

function initials(name: string) {
  return name.trim().slice(0, 2).toUpperCase() || "?";
}

/** A selectable account row for pickers inside the transfer/deposit flows. */
export function AccountOption({
  name,
  number,
  subtitle,
  balance,
  selected,
  onClick,
}: Props) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={cn(
        "flex w-full items-center gap-3 rounded-lg border bg-card px-3 py-2.5 text-left transition-[background-color,border-color,box-shadow] outline-none hover:border-ring/40 hover:bg-muted/40 focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50",
        selected && "border-primary/50 bg-primary/5 ring-1 ring-primary/20",
      )}
    >
      <span
        className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-xs font-semibold text-primary"
        aria-hidden
      >
        {initials(name)}
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm font-medium">{name}</span>
        <span className="truncate text-xs text-muted-foreground">
          {number}
          {subtitle ? ` · ${subtitle}` : ""}
        </span>
      </span>
      {balance ? (
        <span className="shrink-0 font-mono text-sm font-medium tabular-nums">
          {balance}
        </span>
      ) : null}
      <span
        className={cn(
          "flex size-5 shrink-0 items-center justify-center rounded-full border transition-colors",
          selected
            ? "border-primary bg-primary text-primary-foreground"
            : "border-input",
        )}
        aria-hidden
      >
        {selected ? <Check className="size-3" /> : null}
      </span>
    </button>
  );
}

export function AccountOptionSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-2" aria-hidden>
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="flex items-center gap-3 rounded-lg border px-3 py-2.5"
        >
          <Skeleton className="size-9 rounded-lg" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      ))}
    </div>
  );
}
