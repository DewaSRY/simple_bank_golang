import type { ReactNode } from "react";
import {
  CircleAlert,
  CircleCheck,
  Info,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

type Variant = "destructive" | "warning" | "info" | "success";

const VARIANTS: Record<Variant, { Icon: LucideIcon; className: string }> = {
  destructive: {
    Icon: CircleAlert,
    className: "border-destructive/30 bg-destructive/5 [&_svg]:text-destructive",
  },
  warning: {
    Icon: TriangleAlert,
    className: "border-warning/30 bg-warning/5 [&_svg]:text-warning",
  },
  info: {
    Icon: Info,
    className: "border-border bg-muted/40 [&_svg]:text-primary",
  },
  success: {
    Icon: CircleCheck,
    className: "border-success/30 bg-success/5 [&_svg]:text-success",
  },
};

interface Props {
  variant?: Variant;
  title?: ReactNode;
  children?: ReactNode;
  icon?: LucideIcon;
  className?: string;
}

/**
 * In-flow message for form/step-level feedback (a failed submit, a policy
 * notice). Use a toast for transient, app-level feedback instead.
 */
export function InlineAlert({
  variant = "destructive",
  title,
  children,
  icon,
  className,
}: Props) {
  const { Icon: DefaultIcon, className: variantClassName } = VARIANTS[variant];
  const Icon = icon ?? DefaultIcon;

  return (
    <div
      role={variant === "destructive" ? "alert" : "status"}
      className={cn(
        "flex gap-3 rounded-lg border p-3 text-sm",
        variantClassName,
        className,
      )}
    >
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden />
      <div className="min-w-0 space-y-0.5">
        {title ? (
          <p
            className={cn(
              "font-medium",
              variant === "destructive" && "text-destructive",
            )}
          >
            {title}
          </p>
        ) : null}
        {children ? (
          <div
            className={cn(
              title ? "text-muted-foreground" : "",
              !title && variant === "destructive" && "text-destructive",
            )}
          >
            {children}
          </div>
        ) : null}
      </div>
    </div>
  );
}
