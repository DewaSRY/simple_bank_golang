import type { ReactNode } from "react";

export function AuthBackdrop({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-1 items-center justify-center overflow-hidden bg-background">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_40%,var(--brand-soft),transparent_60%)]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -left-24 top-1/3 size-72 animate-pulse rounded-full bg-brand/20 blur-3xl"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -right-24 bottom-1/3 size-72 animate-pulse rounded-full bg-brand-300/20 blur-3xl [animation-delay:300ms]"
      />

      {children}
    </div>
  );
}
