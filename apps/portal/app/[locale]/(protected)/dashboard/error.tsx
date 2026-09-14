"use client";

import { useEffect } from "react";
import { AlertCircle } from "lucide-react";

import { AccountListMessage } from "@/components/dashboard/account-list-message";
import { Button } from "@/components/ui/button";

export default function DashboardError({
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
    <div className="flex my-2 flex-1 flex-col gap-4 bg-zinc-50 px-6 font-sans dark:bg-black">
      <AccountListMessage icon={AlertCircle} className="text-destructive">
        Failed to load your accounts.
      </AccountListMessage>
      <Button onClick={reset} className="self-center">
        Try again
      </Button>
    </div>
  );
}
