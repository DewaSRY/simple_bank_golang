"use client";

import { AlertCircle, Star, Wallet } from "lucide-react";
import { useTranslations } from "next-intl";
import { getApiErrorMessage } from "@/lib/api/error";

import { useAccounts } from "../../feature/account/hooks/query";

import { AccountListItem } from "./account-card";
import { AccountCardSkeleton } from "./account-card-skeleton";
import { AccountListMessage } from "./account-list-message";

export function AccountList() {
  const t = useTranslations("Common");
  const {
    data: accounts = [],
    error,
    isPending,
  } = useAccounts({ page: 1, limit: 10 });

  if (isPending) {
    return <AccountCardSkeleton />;
  }

  if (error) {
    return (
      <AccountListMessage icon={AlertCircle} className="text-destructive">
        {getApiErrorMessage(error, t("loadAccountsError"))}
      </AccountListMessage>
    );
  }

  if (accounts.length === 0) {
    return (
      <AccountListMessage icon={Wallet}>{t("noAccounts")}</AccountListMessage>
    );
  }

  return (
    <ul className="flex flex-col gap-3">
      {accounts.map((account) => (
        <li key={account.id}>
          <AccountListItem account={account} />
        </li>
      ))}
    </ul>
  );
}
