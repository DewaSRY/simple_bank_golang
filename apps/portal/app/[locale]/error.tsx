"use client";

import { useEffect } from "react";
import { AlertTriangle } from "lucide-react";

import { Card } from "@/components/ui/card";
import ErrorState from "@/components/ui/error-state";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="flex min-h-screen flex-1 items-center justify-center bg-background px-4 py-12">
      <Card className="w-full max-w-md">
        <ErrorState icon={AlertTriangle} onRetry={reset} />
      </Card>
    </main>
  );
}
