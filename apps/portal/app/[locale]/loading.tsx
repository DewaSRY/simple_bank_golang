import { Loader2 } from "lucide-react";

export default function Loading() {
  return (
    <div className="flex min-h-screen flex-1 items-center justify-center">
      <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden />
      <span className="sr-only">Loading…</span>
    </div>
  );
}
