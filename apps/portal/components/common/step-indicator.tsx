"use client";

import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface Props {
  /** 1-based index of the active step. */
  current: number;
  total: number;
  className?: string;
}

/** Compact "Step 2 of 4" + segmented bar for multi-step dialogs. */
export function StepIndicator({ current, total, className }: Props) {
  const { t } = useTranslation("common");

  return (
    <div className={cn("flex items-center gap-3", className)}>
      <span className="text-xs font-medium whitespace-nowrap text-muted-foreground tabular-nums">
        {t("stepOf", { current, total })}
      </span>
      <div className="flex flex-1 gap-1" aria-hidden>
        {Array.from({ length: total }).map((_, index) => (
          <span
            key={index}
            className={cn(
              "h-1 flex-1 rounded-full transition-colors duration-300",
              index < current ? "bg-primary" : "bg-muted",
            )}
          />
        ))}
      </div>
    </div>
  );
}
