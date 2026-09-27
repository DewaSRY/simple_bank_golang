import { Inbox, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "cn";
import { Button } from "@/components/ui/button";

interface EmptyStateProps {
  title: string;
  description?: string;
  /** Defaults to a neutral inbox glyph. */
  icon?: LucideIcon;
  /** Hides the icon tile (kept under its old name for existing call sites). */
  hideImage?: boolean;
  /** `sm` for tight spots: sidebars, dialog lists, table rows. */
  size?: "default" | "sm";
  primaryAction?: {
    label: string;
    onClick: () => void;
    icon?: LucideIcon;
  };
  secondaryAction?: {
    label: string;
    onClick: () => void;
  };
  /** Arbitrary action content (e.g. a Link styled as a button). */
  children?: ReactNode;
  className?: string;
}

export default function EmptyState({
  title,
  description,
  icon: Icon = Inbox,
  hideImage = false,
  size = "default",
  primaryAction,
  secondaryAction,
  children,
  className,
}: EmptyStateProps) {
  const isSmall = size === "sm";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center",
        isSmall ? "gap-1 px-3 py-6" : "gap-1.5 px-6 py-12",
        className,
      )}
      data-testid="empty-state"
    >
      {!hideImage && (
        <span
          className={cn(
            "mb-2 flex items-center justify-center rounded-full bg-muted text-muted-foreground ring-8 ring-muted/40",
            isSmall ? "size-9" : "size-12",
          )}
          aria-hidden
        >
          <Icon className={isSmall ? "size-4" : "size-5"} />
        </span>
      )}
      <p
        className={cn(
          "font-semibold text-foreground",
          isSmall ? "text-sm" : "text-base",
        )}
      >
        {title}
      </p>
      {description ? (
        <p
          className={cn(
            "max-w-sm text-muted-foreground",
            isSmall ? "text-xs" : "text-sm",
          )}
        >
          {description}
        </p>
      ) : null}
      {(primaryAction || secondaryAction || children) && (
        <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
          {secondaryAction && (
            <Button variant="outline" onClick={secondaryAction.onClick}>
              {secondaryAction.label}
            </Button>
          )}
          {primaryAction && (
            <Button onClick={primaryAction.onClick}>
              {primaryAction.icon && <primaryAction.icon aria-hidden />}
              {primaryAction.label}
            </Button>
          )}
          {children}
        </div>
      )}
    </div>
  );
}
