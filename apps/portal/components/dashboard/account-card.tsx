"use client";

import { Star } from "lucide-react";
import { Card } from "../ui/card";
import type { AccountWithUserName } from "../../feature/account/client";

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
    <Card className="flex-row items-start gap-2 px-4 transition-colors hover:bg-muted/40 py-2!">
      <div className="flex size-10 h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-sm font-medium text-primary">
        {initials(account.name || account.username)}
      </div>

      <div className="flex flex-col min-w-0 items-end gap-3 justify-between w-full">
        <div className="flex min-w-0 flex-col">
          <span className="flex items-center gap-1.5 truncate font-medium self-end">
            {account.name}
            {account.is_main && (
              <Star
                className="size-3.5 shrink-0 fill-amber-400 text-amber-400"
                aria-hidden
              />
            )}
          </span>
          <span className="truncate text-xs text-muted-foreground self-end">
            {account.number}
          </span>
        </div>

        <div className="flex shrink-0 justify-end flex-row items-end gap-2 self-end">
          <span className="font-mono font-medium">
            {formatBalance(account.balance)}
          </span>
          <span className="text-xs text-muted-foreground">
            {account.currency}
          </span>
        </div>
      </div>
    </Card>
  );
}
