"use client";

import { useTranslation } from "react-i18next";
import { Star } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Link } from "@/i18n/navigation";
import { formatAccountAmount } from "@/lib/number";
import type { AccountWithUserName } from "@/feature/account";

interface Props {
  accounts: AccountWithUserName[];
  /** Server-side total; when larger than `accounts`, totals would be partial. */
  totalCount: number;
}

function sumByCurrency(accounts: AccountWithUserName[]) {
  const totals = new Map<string, number>();
  for (const account of accounts) {
    const value = Number(account.balance);
    if (Number.isNaN(value)) continue;
    totals.set(account.currency, (totals.get(account.currency) ?? 0) + value);
  }
  return [...totals.entries()];
}

/**
 * The dashboard's hero: combined balance (per currency) plus a shortcut to
 * the main account. The total is only shown when every account is loaded,
 * so it never silently under-reports a paginated list.
 */
export function BalanceOverview({ accounts, totalCount }: Props) {
  const { t } = useTranslation("common");
  const isComplete = totalCount <= accounts.length;
  const totals = isComplete ? sumByCurrency(accounts) : [];
  const [primary, ...others] = totals;
  const mainAccount = accounts.find((account) => account.is_main);

  if (!primary && !mainAccount) return null;

  return (
    <Card className="border-0 bg-brand-surface p-6 text-brand-surface-foreground shadow-md ring-0 sm:p-8">
      <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
        <div className="min-w-0 space-y-1">
          <p className="text-sm text-brand-surface-foreground/75">
            {primary ? t("totalBalance") : t("mainAccount")}
          </p>
          <p className="truncate font-mono text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl">
            {primary
              ? formatAccountAmount(String(primary[1]), primary[0])
              : formatAccountAmount(mainAccount!.balance, mainAccount!.currency)}
          </p>
          {primary ? (
            <p className="text-sm text-brand-surface-foreground/75">
              {t("acrossAccounts", { count: accounts.length })}
              {others.length > 0 &&
                ` · ${others
                  .map(([currency, amount]) =>
                    formatAccountAmount(String(amount), currency),
                  )
                  .join(" · ")}`}
            </p>
          ) : null}
        </div>

        {mainAccount && primary && accounts.length > 1 ? (
          <Link
            href={`/account/${mainAccount.id}`}
            className="flex min-w-0 items-center gap-3 rounded-lg bg-brand-surface-foreground/10 px-4 py-3 text-sm transition-colors outline-none hover:bg-brand-surface-foreground/15 focus-visible:ring-2 focus-visible:ring-brand-surface-foreground/60 md:max-w-xs"
          >
            <Star
              className="size-4 shrink-0 fill-current text-brand-surface-foreground/90"
              aria-hidden
            />
            <span className="min-w-0">
              <span className="block text-xs text-brand-surface-foreground/75">
                {t("mainAccount")}
              </span>
              <span className="block truncate font-medium">
                {mainAccount.name} ·{" "}
                <span className="font-mono tabular-nums">
                  {formatAccountAmount(
                    mainAccount.balance,
                    mainAccount.currency,
                  )}
                </span>
              </span>
            </span>
          </Link>
        ) : null}
      </div>
    </Card>
  );
}
