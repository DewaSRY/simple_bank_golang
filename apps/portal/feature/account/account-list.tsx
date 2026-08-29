"use client";

import { useTranslations } from "next-intl";
import { getApiErrorMessage } from "@/lib/api/error";
import { useAccounts } from "./hooks/query";

export function AccountList() {
  const t = useTranslations("Common");
  const {
    data: accounts = [],
    error,
    isPending,
  } = useAccounts({ page: 1, limit: 10 });

  if (isPending) {
    return (
      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("loadingAccounts")}
      </p>
    );
  }

  if (error) {
    return (
      <p className="text-sm text-red-600">
        {getApiErrorMessage(error, t("loadAccountsError"))}
      </p>
    );
  }

  if (accounts.length === 0) {
    return (
      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("noAccounts")}
      </p>
    );
  }

  return (
    <ul className="flex flex-col gap-2">
      {accounts.map((account) => (
        <li
          key={account.id}
          className="flex items-center justify-between rounded border border-black/[.08] px-4 py-3 dark:border-white/[.145]"
        >
          <span>{account.owner}</span>
          <span className="font-mono">
            {account.balance} {account.currency}
          </span>
        </li>
      ))}
    </ul>
  );
}
