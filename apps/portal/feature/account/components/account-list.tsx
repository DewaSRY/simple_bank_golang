"use client";

import { Plus, Wallet } from "lucide-react";
import { useTranslation } from "react-i18next";
import { getApiErrorMessage } from "@/lib/api/error";

import { Card } from "@/components/ui/card";
import EmptyState from "@/components/ui/empty-state";
import ErrorState from "@/components/ui/error-state";
import { useQuickActionStore } from "@/components/navigation/quick-action-store";
import { useAccounts } from "@/feature/account";

import { AccountListItem } from "./account-card";
import { AccountCardSkeleton } from "./account-card-skeleton";
import { BalanceOverview } from "./balance-overview";

export function AccountList() {
  const { t } = useTranslation("common");
  const { t: tAccount } = useTranslation("account");
  const openCreateAccount = useQuickActionStore((s) => s.openCreateAccount);
  const {
    data: accounts,
    error,
    isPending,
    isRefetching,
    refetch,
  } = useAccounts({ page: 1, limit: 10, name: "" });

  if (isPending) {
    return <AccountCardSkeleton />;
  }

  if (error) {
    return (
      <Card>
        <ErrorState
          title={t("loadAccountsError")}
          description={getApiErrorMessage(error, t("error.description"))}
          onRetry={() => void refetch()}
          retrying={isRefetching}
        />
      </Card>
    );
  }

  if (!accounts || accounts.data.length === 0) {
    return (
      <Card>
        <EmptyState
          icon={Wallet}
          title={t("noAccounts")}
          description={t("noAccountsDescription")}
          primaryAction={{
            label: tAccount("createNewAccount"),
            icon: Plus,
            onClick: openCreateAccount,
          }}
        />
      </Card>
    );
  }

  const total = accounts.meta?.total ?? accounts.data.length;

  return (
    <div className="space-y-8">
      <BalanceOverview accounts={accounts.data} totalCount={total} />

      <section aria-labelledby="accounts-heading" className="space-y-4">
        <div className="flex items-baseline justify-between gap-4">
          <h2
            id="accounts-heading"
            className="text-base font-semibold tracking-tight"
          >
            {t("yourAccounts")}
          </h2>
          <span className="text-sm text-muted-foreground tabular-nums">
            {t("accountsCount", { count: total })}
          </span>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {accounts.data.map((account) => (
            <AccountListItem key={account.id} account={account} />
          ))}
          <button
            type="button"
            onClick={openCreateAccount}
            className="flex min-h-32 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-input text-sm font-medium text-muted-foreground transition-colors outline-none hover:border-primary/50 hover:bg-primary/5 hover:text-primary focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <span className="flex size-9 items-center justify-center rounded-full bg-muted">
              <Plus className="size-4" aria-hidden />
            </span>
            {tAccount("createNewAccount")}
          </button>
        </div>
      </section>
    </div>
  );
}
