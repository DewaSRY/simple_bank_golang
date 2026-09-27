"use client";

import { Wallet } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Card } from "@/components/ui/card";
import { CopyButton } from "@/components/common/copy-button";
import type { AccountWithUserName } from "@/feature/account";
import { formatAccountAmount } from "@/lib/number";

export function AccountSummaryCard({
  account,
}: {
  account: AccountWithUserName;
}) {
  const { t } = useTranslation("account");
  const { t: tCommon } = useTranslation("common");

  return (
    <Card className="justify-between gap-8 border-0 bg-brand-surface p-6 text-brand-surface-foreground shadow-md ring-0 sm:p-8">
      <div className="flex items-center gap-3">
        <span
          className="flex size-10 items-center justify-center rounded-lg bg-brand-surface-foreground/15"
          aria-hidden
        >
          <Wallet className="size-5" />
        </span>
        <div className="min-w-0">
          <p className="text-xs text-brand-surface-foreground/75">
            {t("accountNumber")}
          </p>
          <p className="flex items-center gap-1 font-mono text-sm font-medium tracking-wide">
            <span className="truncate">{account.number}</span>
            <CopyButton
              value={account.number}
              label={tCommon("copyAccountNumber")}
              className="hover:bg-brand-surface-foreground/15"
            />
          </p>
        </div>
      </div>
      <div className="min-w-0">
        <p className="text-sm text-brand-surface-foreground/75">
          {tCommon("availableBalance")}
        </p>
        <p className="mt-1 truncate font-mono text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl">
          {formatAccountAmount(account.balance, account.currency)}
        </p>
      </div>
    </Card>
  );
}
