"use client";

import { AlertCircle, Star, Wallet } from "lucide-react";
import { useTranslations } from "next-intl";
import { getApiErrorMessage } from "@/lib/api/error";
import { cn } from "@/lib/utils";
import { Card } from "../ui/card";
import { Skeleton } from "../ui/skeleton";
import type { AccountWithUserName } from "../../feature/account/client";
import { useAccounts } from "../../feature/account/hooks/query";

function formatBalance(balance: string) {
  const value = Number(balance);
  if (Number.isNaN(value)) return balance;
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

function initials(name: string) {
  return name.trim().slice(0, 2).toUpperCase() || "?";
}

export function AccountListItem({ account }: { account: AccountWithUserName }) {
  return (
    <Card className="flex-row items-center justify-between px-4 transition-colors hover:bg-muted/40">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-medium text-primary">
          {initials(account.name || account.username)}
        </div>
        <div className="flex min-w-0 flex-col">
          <span className="flex items-center gap-1.5 truncate font-medium">
            {account.name || account.username}
            {account.is_main && (
              <Star
                className="size-3.5 shrink-0 fill-amber-400 text-amber-400"
                aria-hidden
              />
            )}
          </span>
          <span className="truncate text-xs text-muted-foreground">
            {account.username} &middot; {account.number}
          </span>
        </div>
      </div>
      <div className="flex shrink-0 flex-col items-end">
        <span className="font-mono font-medium">
          {formatBalance(account.balance)}
        </span>
        <span className="text-xs text-muted-foreground">
          {account.currency}
        </span>
      </div>
    </Card>
  );
}
