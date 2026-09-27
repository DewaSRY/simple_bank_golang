"use client";

import type { ReactNode } from "react";
import { RotateCcw, ServerCrash, type LucideIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "cn";
import { Button } from "@/components/ui/button";

interface ErrorStateProps {
  icon?: LucideIcon;
  title?: string;
  description?: string;
  /** Renders a "Try again" button when provided. */
  onRetry?: () => void;
  retrying?: boolean;
  /** Extra actions (e.g. a "Return to dashboard" link). */
  children?: ReactNode;
  className?: string;
}

export default function ErrorState({
  icon: Icon = ServerCrash,
  title,
  description,
  onRetry,
  retrying = false,
  children,
  className,
}: ErrorStateProps) {
  const { t } = useTranslation("common");
  const resolvedTitle = title ?? t("error.title");
  const resolvedDescription = description ?? t("error.description");

  return (
    <div
      role="alert"
      className={cn(
        "flex flex-1 flex-col items-center justify-center gap-1.5 px-6 py-12 text-center",
        className,
      )}
      data-testid="error-state"
    >
      <span
        className="mb-2 flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive ring-8 ring-destructive/5"
        aria-hidden
      >
        <Icon className="size-5" />
      </span>
      <h3 className="text-base font-semibold text-foreground">
        {resolvedTitle}
      </h3>
      <p className="max-w-sm text-sm text-muted-foreground">
        {resolvedDescription}
      </p>
      {(onRetry || children) && (
        <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
          {children}
          {onRetry && (
            <Button onClick={onRetry} loading={retrying}>
              {!retrying && <RotateCcw aria-hidden />}
              {t("tryAgain")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
