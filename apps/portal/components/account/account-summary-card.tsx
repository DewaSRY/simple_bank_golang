"use client";

import { Wallet } from "lucide-react";
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AccountWithUserName } from "@/feature/account/type";
import { formatAccountAmount } from "@/feature/account-transaction/utils";

export function AccountSummaryCard({
  account,
}: {
  account: AccountWithUserName;
}) {
  return (
    <Card className="overflow-hidden border-0 bg-primary text-primary-foreground shadow-lg">
      <CardHeader className="gap-6 p-6 sm:p-8">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className="flex size-11 items-center justify-center rounded-xl bg-primary-foreground/15">
              <Wallet className="size-5" aria-hidden />
            </span>
            <div>
              <CardTitle className="text-xl">{account.name}</CardTitle>
              <CardDescription className="mt-1 text-primary-foreground/65">
                {account.number}
              </CardDescription>
            </div>
          </div>
          {account.is_main && (
            <span className="rounded-full bg-primary-foreground/15 px-3 py-1 text-xs font-medium">
              Main account
            </span>
          )}
        </div>
        <div>
          <p className="text-sm text-primary-foreground/65">
            Available balance
          </p>
          <p className="mt-1 font-mono text-4xl font-semibold tracking-tight">
            {formatAccountAmount(account.balance, account.currency)}
          </p>
        </div>
      </CardHeader>
    </Card>
  );
}
