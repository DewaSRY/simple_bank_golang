"use client";

import { useEffect } from "react";

import { AccountStateMessage } from "@/components/account/account-state-message";

export default function AccountDetailError({
  error,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <AccountStateMessage
      title="Unable to load account"
      description="Please try again in a moment."
    />
  );
}
